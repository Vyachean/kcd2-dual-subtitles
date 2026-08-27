package modarchive

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"hash/crc32"
	"math"
	"os"
	"path/filepath"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

const (
	ManifestFilename = "mod.manifest"

	// KCD2 recognizes localization patch resources named text_ui__<modid>.xml.
	LocalizationPatchArchivePath = "text_ui__" + ModID + ".xml"

	// ZIP's legacy MS-DOS date representation of 1980-01-01. We deliberately
	// leave FileHeader.Modified zero so Go does not emit extended timestamp
	// extra fields that are known to cause CryPak compatibility problems.
	deterministicDOSDate uint16 = 33
	deterministicDOSTime uint16 = 0

	// Retail KCD2 data PAKs accept stored ZIP entries whose headers identify
	// themselves as Windows ZIP 1.0. Go's CreateRaw deliberately preserves zero
	// version fields unless callers populate them, and retail CryPak rejects that
	// shape for mod Data/*.pak even though the localization loader is more lenient.
	kcd2StoredZIPVersion      uint16 = 10
	kcd2WindowsCreatorVersion uint16 = 10 // platform byte 0 (Windows), spec byte 10.
)

var (
	ErrUnsupportedLanguage = errors.New("unsupported language")
	ErrOutputExists        = errors.New("output path already exists")
)

type archiveEntry struct {
	name string
	data []byte
}

// Build writes a development-version mod archive. Production callers should
// use BuildVersioned so the generated manifest identifies the executable that
// created it.
func Build(outputPath string, mainLanguage localization.Language, rows []localization.DialogueRow) error {
	return BuildVersioned(outputPath, mainLanguage, rows, "dev")
}

// BuildVersioned writes a directly installable KCD2 mod distribution ZIP to
// outputPath. Extracting it into the platform's KCD2 mod directory creates the
// ModID folder. outputPath must not already exist.
func BuildVersioned(outputPath string, mainLanguage localization.Language, rows []localization.DialogueRow, version string) error {
	archiveData, err := buildArchiveBytesVersioned(mainLanguage, rows, version)
	if err != nil {
		return err
	}

	return publishArchive(outputPath, archiveData)
}

func buildArchiveBytes(mainLanguage localization.Language, rows []localization.DialogueRow) ([]byte, error) {
	return buildArchiveBytesVersioned(mainLanguage, rows, "dev")
}

func buildArchiveBytesVersioned(mainLanguage localization.Language, rows []localization.DialogueRow, version string) ([]byte, error) {
	languageInfo, ok := localization.LookupLanguage(mainLanguage)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedLanguage, mainLanguage)
	}

	localizationPAK, err := buildLocalizationPAK(rows)
	if err != nil {
		return nil, fmt.Errorf("build localization PAK: %w", err)
	}

	return buildZip([]archiveEntry{
		{name: modArchivePath(ManifestFilename), data: manifestForVersion(version)},
		{name: modArchivePath(filepath.ToSlash(filepath.Join("Localization", languageInfo.PakFilename))), data: localizationPAK},
	}, zip.Deflate)
}

func buildLocalizationPAK(rows []localization.DialogueRow) ([]byte, error) {
	patchXML, err := localization.MarshalDialogueXML(rows)
	if err != nil {
		return nil, err
	}

	// Current KCD2 localization mods patch existing string IDs through a
	// text_ui__<modid>.xml resource. Retail localization PAKs and the live-proven
	// Chineses Fix use raw DEFLATE entries with no data descriptor or extra
	// fields, which avoids storing the large generated XML verbatim.
	return buildDeflatedLocalizationCryPak([]archiveEntry{
		{name: LocalizationPatchArchivePath, data: patchXML},
	})
}

func modArchivePath(relativePath string) string {
	return filepath.ToSlash(filepath.Join(ModID, relativePath))
}

// buildCryPak preserves the accepted stored-entry CryPak byte contract for
// callers that require it. Generated localization uses the separately proven
// DEFLATE contract in buildDeflatedLocalizationCryPak.
func buildCryPak(entries []archiveEntry) ([]byte, error) {
	return buildRawCryPak(entries, 0, 0)
}

// buildDataCryPak emits the ZIP header version fields used by working KCD2 data
// PAK builders. This is intentionally separate from the accepted localization
// PAK contract because retail evidence only showed Data/*.pak being rejected.
func buildDataCryPak(entries []archiveEntry) ([]byte, error) {
	return buildRawCryPak(entries, kcd2WindowsCreatorVersion, kcd2StoredZIPVersion)
}

func buildRawCryPak(entries []archiveEntry, creatorVersion, readerVersion uint16) ([]byte, error) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)

	for _, entry := range entries {
		if uint64(len(entry.data)) > math.MaxUint32 {
			_ = writer.Close()
			return nil, fmt.Errorf("CryPak entry %q exceeds 32-bit ZIP size limit", entry.name)
		}
		size := uint32(len(entry.data))
		header := &zip.FileHeader{
			Name:               entry.name,
			Method:             zip.Store,
			Flags:              0,
			CreatorVersion:     creatorVersion,
			ReaderVersion:      readerVersion,
			CRC32:              crc32.ChecksumIEEE(entry.data),
			CompressedSize:     size,
			UncompressedSize:   size,
			CompressedSize64:   uint64(size),
			UncompressedSize64: uint64(size),
			ModifiedTime:       deterministicDOSTime,
			ModifiedDate:       deterministicDOSDate,
		}

		entryWriter, err := writer.CreateRaw(header)
		if err != nil {
			_ = writer.Close()
			return nil, fmt.Errorf("create CryPak entry %q: %w", entry.name, err)
		}
		if _, err := entryWriter.Write(entry.data); err != nil {
			_ = writer.Close()
			return nil, fmt.Errorf("write CryPak entry %q: %w", entry.name, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close CryPak archive: %w", err)
	}
	return buffer.Bytes(), nil
}

func buildZip(entries []archiveEntry, method uint16) ([]byte, error) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)

	for _, entry := range entries {
		header := &zip.FileHeader{
			Name:         entry.name,
			Method:       method,
			ModifiedTime: deterministicDOSTime,
			ModifiedDate: deterministicDOSDate,
		}

		entryWriter, err := writer.CreateHeader(header)
		if err != nil {
			_ = writer.Close()
			return nil, fmt.Errorf("create ZIP entry %q: %w", entry.name, err)
		}
		if _, err := entryWriter.Write(entry.data); err != nil {
			_ = writer.Close()
			return nil, fmt.Errorf("write ZIP entry %q: %w", entry.name, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close ZIP archive: %w", err)
	}
	return buffer.Bytes(), nil
}

func publishArchive(outputPath string, data []byte) error {
	if err := ensureOutputAbsent(outputPath); err != nil {
		return err
	}

	directory := filepath.Dir(outputPath)
	temporary, err := os.CreateTemp(directory, "."+filepath.Base(outputPath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary output beside %q: %w", outputPath, err)
	}

	temporaryPath := temporary.Name()
	closed := false
	defer func() {
		if !closed {
			_ = temporary.Close()
		}
		_ = os.Remove(temporaryPath)
	}()

	if _, err := temporary.Write(data); err != nil {
		return fmt.Errorf("write temporary output %q: %w", temporaryPath, err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync temporary output %q: %w", temporaryPath, err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary output %q: %w", temporaryPath, err)
	}
	closed = true

	if err := ensureOutputAbsent(outputPath); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, outputPath); err != nil {
		return fmt.Errorf("publish mod archive %q: %w", outputPath, err)
	}

	return nil
}

func ensureOutputAbsent(outputPath string) error {
	_, err := os.Stat(outputPath)
	if err == nil {
		return fmt.Errorf("%w: %q", ErrOutputExists, outputPath)
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("check output path %q: %w", outputPath, err)
}
