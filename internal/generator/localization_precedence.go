package generator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/localizationsource"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modarchive"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modinstall"
)

var ErrUnsafeLocalizationLoadOrder = errors.New("generated mod would not load after a localization source mod")

func validateAutomaticLocalizationPrecedence(request Request, contributionGroups ...[]localizationsource.Contribution) error {
	modsRoot, err := modsRootForRequest(request)
	if err != nil {
		return err
	}
	orderPath := filepath.Join(modsRoot, modinstall.ModOrderFilename)
	if _, err := os.Stat(orderPath); err == nil {
		// The transaction-safe installer enforces exactly one project entry as
		// the final active mod_order.txt entry before commit.
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect mod load order %q: %w", orderPath, err)
	}

	projectFolder := strings.ToLower(modarchive.ModID)
	seen := make(map[string]struct{})
	for _, contributions := range contributionGroups {
		for _, contribution := range contributions {
			folder := filepath.Base(filepath.Clean(contribution.Path))
			key := strings.ToLower(folder)
			if key == "" || key == "." {
				continue
			}
			if _, duplicate := seen[key]; duplicate {
				continue
			}
			seen[key] = struct{}{}
			if key > projectFolder || (key == projectFolder && folder != modarchive.ModID) {
				name := strings.TrimSpace(contribution.Name)
				if name == "" {
					name = strings.TrimSpace(contribution.ModID)
				}
				if name == "" {
					name = folder
				}
				return fmt.Errorf("%w: localization mod %q uses folder %q, which sorts after %q without %s; create an explicit %s with %s last, or change the active Mods folder before generating", ErrUnsafeLocalizationLoadOrder, name, folder, modarchive.ModID, modinstall.ModOrderFilename, modinstall.ModOrderFilename, modarchive.ModID)
			}
		}
	}
	return nil
}

func modsRootForRequest(request Request) (string, error) {
	if modsRoot := strings.TrimSpace(request.ModsRoot); modsRoot != "" {
		location, err := modinstall.ValidateCustomModsRoot(modsRoot)
		if err != nil {
			return "", err
		}
		return location.ModsRoot, nil
	}
	location, err := modinstall.ResolveModSourceLocation(request.GameRoot)
	if err != nil {
		return "", err
	}
	return location.ModsRoot, nil
}
