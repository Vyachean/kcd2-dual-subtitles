package modinstall

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// InstallLayout identifies only the filesystem convention needed to place
// KCD2 mods. It is deliberately not a storefront/product identifier.
type InstallLayout string

const (
	InstallLayoutGameRoot     InstallLayout = "game-root"
	InstallLayoutGDKDocuments InstallLayout = "gdk-documents"
)

// InstallLocation is the resolved mod root for one selected KCD2 installation.
// All install/status/uninstall operations must use the same resolved ModsRoot.
type InstallLocation struct {
	ModsRoot string
	Layout   InstallLayout
}

var errEmptyGameRoot = errors.New("game root is empty")

// ResolveInstallLocation selects the KCD2 mod root for automatic installation.
// Automatic publication remains Windows-only.
func ResolveInstallLocation(gameRoot string) (InstallLocation, error) {
	if runtime.GOOS != "windows" {
		return InstallLocation{}, ErrAutomaticInstallUnsupported
	}
	return ResolveModSourceLocation(gameRoot)
}

// ResolveModSourceLocation uses the same filesystem-layout resolver as
// automatic installation, but is also available to read installed mod content
// during portable generation. Standard <game-root>/Mods layouts are therefore
// resolvable cross-platform; a GDK layout still requires a working Windows
// Documents resolver.
func ResolveModSourceLocation(gameRoot string) (InstallLocation, error) {
	return resolveInstallLocation(gameRoot, documentsPath)
}

func resolveInstallLocation(gameRoot string, resolveDocuments func() (string, error)) (InstallLocation, error) {
	gameRoot = strings.TrimSpace(gameRoot)
	if gameRoot == "" {
		return InstallLocation{}, errEmptyGameRoot
	}
	absolute, err := filepath.Abs(gameRoot)
	if err != nil {
		return InstallLocation{}, fmt.Errorf("normalize game root %q: %w", gameRoot, err)
	}
	gameRoot = filepath.Clean(absolute)

	if isGDKContentRoot(gameRoot) {
		documents, err := resolveDocuments()
		if err != nil {
			return InstallLocation{}, fmt.Errorf("resolve GDK mod root: %w", err)
		}
		documents = strings.TrimSpace(documents)
		if documents == "" {
			return InstallLocation{}, errors.New("resolve GDK mod root: Documents path is empty")
		}
		return InstallLocation{
			ModsRoot: filepath.Join(documents, ModsDirectoryName),
			Layout:   InstallLayoutGDKDocuments,
		}, nil
	}

	return InstallLocation{
		ModsRoot: filepath.Join(gameRoot, "Mods"),
		Layout:   InstallLayoutGameRoot,
	}, nil
}

// isGDKContentRoot uses package artifacts from the selected installation
// rather than drive/folder names. gamelaunchhelper.exe is present in the
// retail KCD2 GDK build; the metadata names cover standard GDK packaging.
func isGDKContentRoot(gameRoot string) bool {
	markers := []string{
		"gamelaunchhelper.exe",
		"MicrosoftGame.config",
		"appxmanifest.xml",
	}
	if directoryContainsAnyFile(gameRoot, markers) {
		return true
	}
	return directoryContainsAnyFile(filepath.Dir(gameRoot), []string{
		"MicrosoftGame.config",
		"appxmanifest.xml",
	})
}

func directoryContainsAnyFile(directory string, names []string) bool {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		for _, name := range names {
			if strings.EqualFold(entry.Name(), name) {
				return true
			}
		}
	}
	return false
}
