package modarchive

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

const (
	ManifestFilename                = "mod.manifest"
	GeneratedDialogueXMLArchivePath = "text_kcd_dual_subtitles.xml"

	// ZIP's legacy MS-DOS date representation of 1980-01-01. We deliberately
	// leave FileHeader.Modified zero so Go does not emit extended timestamp
	// extra fields that are known to cause CryPak compatibility problems.
	deterministicDOSDate uint16 = 33
	deterministicDOSTime uint16 = 0
)

var (
	ErrUnsupportedLanguage = errors.New("unsupported language")
	ErrOutputExists        = errors.New("output path already exists")
)

type archiveEntry struct {
	name string
	data []byte
}

// Build writes a directly installable KCD2 mod distribution ZIP to outputPath.
// Extracting it into the game's Mods directory creates the ModID folder.
// outputPath must not already exist.
func Build(outputPath string, mainLanguage localization.Language, rows []localization.DialogueRow) error {
	archiveData, err := buildArchiveBytes(mainLanguage, rows)
	if err != nil {
		return err
	}

	return publishArchive(outputPath, archiveData)
}

func buildArchiveBytes(mainLanguage localization.Language, rows []localization.DialogueRow) ([]byte, error) {
	languageInfo, ok := localization.LookupLanguage(mainLanguage)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedLanguage, mainLanguage)
	}

	localizationPAK, err := buildLocalizationPAK(rows)
	if err != nil {
		return nil, fmt.Errorf("build localization PAK: %w", err)
	}

	return buildZip([]archiveEntry{
		{name: modArchivePath(ManifestFilename), data: []byte(manifest)},
		{name: modArchivePath(filepath.ToSlash(filepath.Join("Localization", languageInfo.PakFilename))), data: localizationPAK},
	}, zip.Deflate)
}

func buildLocalizationPAK(rows []localization.DialogueRow) ([]byte, error) {
	dialogueXML, err := localization.MarshalDialogueXML(rows)
	if err != nil {
		return nil, err
	}

	// Store is the conservative format documented by the official KCD2 wiki.
	// More recent tooling also accepts Deflate, but compression is immaterial for
	// this small generated PAK and Store minimizes format variance.
	return buildZip([]archiveEntry{
		{name: GeneratedDialogueXMLArchivePath, data: dialogueXML},
	}, zip.Store)
}

func modArchivePath(relativePath string) string {
	return filepath.ToSlash(filepath.Join(ModID, relativePath))
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
