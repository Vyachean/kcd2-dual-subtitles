package gamedetect

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var ErrInvalidGameRoot = errors.New("selected directory is not a supported KCD2 Content root")

var requiredRelativeFiles = []string{
	filepath.Join("Localization", "English_xml.pak"),
	filepath.Join("Localization", "Russian_xml.pak"),
	filepath.Join("Data", "Scripts.pak"),
	filepath.Join("Data", "Tables.pak"),
}

// Result contains all structurally valid KCD2 Content roots found by the
// platform-specific Xbox installation discovery pass.
type Result struct {
	Candidates []string
}

// Unique returns the detected Content root only when discovery found exactly
// one structurally valid installation.
func (r Result) Unique() (string, bool) {
	if len(r.Candidates) != 1 {
		return "", false
	}
	return r.Candidates[0], true
}

// NormalizeSelection accepts either a KCD2 Content root or its immediate
// parent directory and returns the validated Content root.
func NormalizeSelection(path string) (string, error) {
	path = strings.TrimSpace(path)
	if len(path) >= 2 && strings.HasPrefix(path, `"`) && strings.HasSuffix(path, `"`) {
		path = strings.TrimSpace(path[1 : len(path)-1])
	}
	if path == "" {
		return "", fmt.Errorf("%w: path is empty", ErrInvalidGameRoot)
	}

	cleaned, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("normalize selected path %q: %w", path, err)
	}
	cleaned = filepath.Clean(cleaned)

	if IsGameRoot(cleaned) {
		return cleaned, nil
	}
	content := filepath.Join(cleaned, "Content")
	if IsGameRoot(content) {
		return content, nil
	}
	return "", fmt.Errorf("%w: %q", ErrInvalidGameRoot, cleaned)
}

// IsGameRoot validates the non-proprietary file layout needed by the current
// Russian/English generator. It intentionally does not read localization
// contents during discovery.
func IsGameRoot(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	for _, relative := range requiredRelativeFiles {
		info, err := os.Stat(filepath.Join(path, relative))
		if err != nil || info.IsDir() {
			return false
		}
	}
	return true
}

func detectInXboxRoots(roots []string) Result {
	seen := make(map[string]struct{})
	candidates := make([]string, 0)

	for _, root := range roots {
		root = filepath.Clean(strings.TrimSpace(root))
		if root == "." || root == "" {
			continue
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			candidate := filepath.Join(root, entry.Name(), "Content")
			if !IsGameRoot(candidate) {
				continue
			}
			absolute, err := filepath.Abs(candidate)
			if err != nil {
				continue
			}
			absolute = filepath.Clean(absolute)
			key := strings.ToLower(absolute)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			candidates = append(candidates, absolute)
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return strings.ToLower(candidates[i]) < strings.ToLower(candidates[j])
	})
	return Result{Candidates: candidates}
}
