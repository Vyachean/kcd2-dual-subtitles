package modarchive

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

var ErrArtifactVerification = errors.New("generated mod artifact verification failed")

// VerifyDirectoryVersionedForLanguages proves that a generated localization-only
// directory contains exactly the deterministic files expected for this request.
func VerifyDirectoryVersionedForLanguages(directory string, targetLanguages []localization.Language, rows []localization.DialogueRow, version string) error {
	expected, err := expectedVersionedFiles(targetLanguages, rows, nil, version)
	if err != nil {
		return err
	}
	return verifyDirectoryFiles(directory, expected)
}

// VerifyDirectoryVersionedWithHUDForLanguages proves that a generated styled
// directory contains the exact localization payload and derived HUD requested.
func VerifyDirectoryVersionedWithHUDForLanguages(directory string, targetLanguages []localization.Language, rows []localization.DialogueRow, hud []byte, version string) error {
	if len(hud) == 0 {
		return ErrHUDRequired
	}
	expected, err := expectedVersionedFiles(targetLanguages, rows, hud, version)
	if err != nil {
		return err
	}
	return verifyDirectoryFiles(directory, expected)
}

func expectedVersionedFiles(targetLanguages []localization.Language, rows []localization.DialogueRow, hud []byte, version string) (map[string][]byte, error) {
	infos, err := targetLanguageInfos(targetLanguages)
	if err != nil {
		return nil, err
	}
	localizationPAK, err := buildLocalizationPAK(rows)
	if err != nil {
		return nil, fmt.Errorf("build localization PAK for verification: %w", err)
	}
	var dataPAK []byte
	if len(hud) != 0 {
		dataPAK, err = buildHUDPAK(hud)
		if err != nil {
			return nil, fmt.Errorf("build HUD data PAK for verification: %w", err)
		}
	}
	return expectedVersionedFilesFromParts(infos, localizationPAK, dataPAK, version), nil
}

func expectedVersionedFilesFromParts(infos []localization.LanguageInfo, localizationPAK, dataPAK []byte, version string) map[string][]byte {
	expected := make(map[string][]byte, 1+len(infos)+1)
	expected[ManifestFilename] = manifestForVersion(version)
	for _, info := range infos {
		expected[filepath.ToSlash(filepath.Join("Localization", info.PakFilename))] = localizationPAK
	}
	if len(dataPAK) != 0 {
		expected[filepath.ToSlash(filepath.Join("Data", DataPAKFilename))] = dataPAK
	}
	return expected
}

func verifyDirectoryFiles(directory string, expected map[string][]byte) error {
	seen := make(map[string]bool, len(expected))
	err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == directory {
			return nil
		}
		relative, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(relative)
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: symlink %q", ErrArtifactVerification, name)
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%w: unsupported file type %q", ErrArtifactVerification, name)
		}
		want, ok := expected[name]
		if !ok {
			return fmt.Errorf("%w: unexpected file %q", ErrArtifactVerification, name)
		}
		if seen[name] {
			return fmt.Errorf("%w: duplicate file %q", ErrArtifactVerification, name)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.Equal(got, want) {
			return fmt.Errorf("%w: content mismatch for %q", ErrArtifactVerification, name)
		}
		seen[name] = true
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrArtifactVerification) {
			return err
		}
		return fmt.Errorf("%w: inspect generated directory %q: %v", ErrArtifactVerification, directory, err)
	}
	for name := range expected {
		if !seen[name] {
			return fmt.Errorf("%w: missing file %q", ErrArtifactVerification, name)
		}
	}
	return nil
}

func verifyArchiveBytes(data []byte, expected map[string][]byte) error {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("%w: open generated distribution ZIP: %v", ErrArtifactVerification, err)
	}
	seen := make(map[string]bool, len(expected))
	for _, file := range reader.File {
		name := filepath.ToSlash(file.Name)
		prefix := ModID + "/"
		if file.FileInfo().IsDir() || len(name) <= len(prefix) || name[:len(prefix)] != prefix {
			return fmt.Errorf("%w: unexpected archive path %q", ErrArtifactVerification, name)
		}
		relative := name[len(prefix):]
		want, ok := expected[relative]
		if !ok {
			return fmt.Errorf("%w: unexpected archive file %q", ErrArtifactVerification, name)
		}
		if seen[relative] {
			return fmt.Errorf("%w: duplicate archive file %q", ErrArtifactVerification, name)
		}
		entry, err := file.Open()
		if err != nil {
			return fmt.Errorf("%w: open archive file %q: %v", ErrArtifactVerification, name, err)
		}
		got, readErr := io.ReadAll(entry)
		closeErr := entry.Close()
		if readErr != nil {
			return fmt.Errorf("%w: read archive file %q: %v", ErrArtifactVerification, name, readErr)
		}
		if closeErr != nil {
			return fmt.Errorf("%w: close archive file %q: %v", ErrArtifactVerification, name, closeErr)
		}
		if !bytes.Equal(got, want) {
			return fmt.Errorf("%w: content mismatch for archive file %q", ErrArtifactVerification, name)
		}
		seen[relative] = true
	}
	for name := range expected {
		if !seen[name] {
			return fmt.Errorf("%w: missing archive file %q", ErrArtifactVerification, modArchivePath(name))
		}
	}
	return nil
}
