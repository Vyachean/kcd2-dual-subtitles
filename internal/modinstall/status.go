package modinstall

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
)

// Status describes whether this tool's generated mod directory currently
// exists in the current user's KCD2 Documents mod root.
type Status struct {
	Installed bool
	Path      string
}

// Inspect returns the current generated-mod installation state.
func Inspect() (Status, error) {
	documents, err := documentsPath()
	if err != nil {
		return Status{}, err
	}
	return inspectInDocuments(documents)
}

func inspectInDocuments(documents string) (Status, error) {
	if documents == "" {
		return Status{}, errors.New("Documents path is empty")
	}
	target := filepath.Join(documents, ModsDirectoryName, modarchive.ModID)
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
