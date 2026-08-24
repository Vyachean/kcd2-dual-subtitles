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

// Inspect keeps the historical Documents-root behavior for focused legacy
// callers. New application code should use InspectForGameRoot.
func Inspect() (Status, error) {
	documents, err := documentsPath()
	if err != nil {
		return Status{}, err
	}
	return inspectInDocuments(documents)
}

// InspectForGameRoot resolves the same target used by automatic installation
// for gameRoot, then inspects this tool's generated mod there.
func InspectForGameRoot(gameRoot string) (Status, error) {
	location, err := ResolveInstallLocation(gameRoot)
	if err != nil {
		return Status{}, err
	}
	return inspectInModsRoot(location.ModsRoot)
}

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
