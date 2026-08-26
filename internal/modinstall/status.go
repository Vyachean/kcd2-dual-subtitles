package modinstall

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

// Status describes whether this tool's generated mod directory currently
// exists in the resolved mod root for a selected KCD2 installation.
type Status struct {
	Installed bool
	Path      string
}

// InspectForGameRoot resolves the same target used by automatic installation
// for gameRoot, then inspects this tool's generated mod there.
func InspectForGameRoot(gameRoot string) (Status, error) {
	location, err := ResolveInstallLocation(gameRoot)
	if err != nil {
		return Status{}, err
	}
	return InspectInModsRoot(location.ModsRoot)
}

// InspectInModsRoot inspects the project installation in an already-selected
// Mods root. Callers that allow a user override must use this path consistently
// instead of resolving the game root again.
func InspectInModsRoot(modsRoot string) (Status, error) {
	return inspectInModsRoot(modsRoot)
}

// Documents-specific inspection remains only for focused GDK filesystem tests.
func inspectInDocuments(documents string) (Status, error) {
	if documents == "" {
		return Status{}, errors.New("Documents path is empty")
	}
	return inspectInModsRoot(filepath.Join(documents, ModsDirectoryName))
}

func inspectInModsRoot(modsRoot string) (Status, error) {
	modsRoot = strings.TrimSpace(modsRoot)
	if modsRoot == "" {
		return Status{}, errors.New("KCD2 mod root is empty")
	}
	target := filepath.Join(modsRoot, modarchive.ModID)
	info, err := os.Lstat(target)
	switch {
	case err == nil:
		if info.Mode()&os.ModeSymlink != 0 {
			return Status{}, fmt.Errorf("unsafe symlink at mod path %q", target)
		}
		if !info.IsDir() {
			return Status{}, fmt.Errorf("non-directory at mod path %q", target)
		}
		return Status{Installed: true, Path: target}, nil
	case errors.Is(err, os.ErrNotExist):
		return Status{Installed: false, Path: target}, nil
	default:
		return Status{}, fmt.Errorf("inspect mod path %q: %w", target, err)
	}
}
