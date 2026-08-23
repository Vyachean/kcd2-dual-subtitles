package localization

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
)

const DialogueXMLArchivePath = "text_ui_dialog.xml"

var (
	ErrDialogueXMLNotFound = errors.New("dialogue XML not found")
	ErrDialogueXMLDuplicate = errors.New("duplicate dialogue XML entry")
)

// ReadDialogueXML reads the raw text_ui_dialog.xml bytes from a localization PAK.
func ReadDialogueXML(pakPath string) ([]byte, error) {
	reader, err := zip.OpenReader(pakPath)
	if err != nil {
		return nil, fmt.Errorf("open localization PAK %q: %w", pakPath, err)
	}

	data, readErr := readDialogueXMLFromArchive(reader.File, pakPath)
	closeErr := reader.Close()

	if readErr != nil {
		if closeErr != nil {
			return nil, errors.Join(readErr, fmt.Errorf("close localization PAK %q: %w", pakPath, closeErr))
		}
		return nil, readErr
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close localization PAK %q: %w", pakPath, closeErr)
	}

	return data, nil
}

func readDialogueXMLFromArchive(files []*zip.File, pakPath string) ([]byte, error) {
	var target *zip.File
	for _, file := range files {
		if file.Name != DialogueXMLArchivePath {
			continue
		}
		if target != nil {
			return nil, fmt.Errorf("%w in localization PAK %q", ErrDialogueXMLDuplicate, pakPath)
		}
		target = file
	}

	if target == nil {
		return nil, fmt.Errorf("%w in localization PAK %q", ErrDialogueXMLNotFound, pakPath)
	}

	entry, err := target.Open()
	if err != nil {
		return nil, fmt.Errorf("open %s in localization PAK %q: %w", DialogueXMLArchivePath, pakPath, err)
	}

	data, readErr := io.ReadAll(entry)
	closeErr := entry.Close()
	if readErr != nil {
		if closeErr != nil {
			return nil, errors.Join(
				fmt.Errorf("read %s from localization PAK %q: %w", DialogueXMLArchivePath, pakPath, readErr),
				fmt.Errorf("close %s in localization PAK %q: %w", DialogueXMLArchivePath, pakPath, closeErr),
			)
		}
		return nil, fmt.Errorf("read %s from localization PAK %q: %w", DialogueXMLArchivePath, pakPath, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close %s in localization PAK %q: %w", DialogueXMLArchivePath, pakPath, closeErr)
	}

	return data, nil
}
