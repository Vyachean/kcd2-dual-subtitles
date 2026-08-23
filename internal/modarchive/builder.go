package modarchive

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

const (
	ManifestArchivePath             = "mod.manifest"
	GeneratedDialogueXMLArchivePath = "text_dualdialog.xml"
)

var (
	ErrUnsupportedLanguage = errors.New("unsupported language")
	ErrOutputExists        = errors.New("output path already exists")
)

var deterministicZipTime = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)

type archiveEntry struct {
	name string
	data []byte
}

// Build writes a standalone KCD2 dual-subtitle mod archive to outputPath.
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
		{name: ManifestArchivePath, data: []byte(manifest)},
		{name: filepath.ToSlash(filepath.Join("Localization", languageInfo.PakFilename)), data: localizationPAK},
	})
}

func buildLocalizationPAK(rows []localization.DialogueRow) ([]byte, error) {
	dialogueXML, err := localization.MarshalDialogueXML(rows)
	if err != nil {
		return nil, err
	}

	return buildZip([]archiveEntry{
		{name: GeneratedDialogueXMLArchivePath, data: dialogueXML},
	})
}

func buildZip(entries []archiveEntry) ([]byte, error) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)

	for _, entry := range entries {
		header := &zip.FileHeader{
			Name:   entry.name,
			Method: zip.Deflate,
		}
		header.SetModTime(deterministicZipTime)
		header.SetMode(0o644)

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
