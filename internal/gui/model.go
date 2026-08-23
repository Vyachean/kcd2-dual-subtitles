package gui

import (
	"fmt"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/gamedetect"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modinstall"
)

// Model contains platform-independent GUI state so startup and state
// transitions can be verified without driving native Win32 controls.
type Model struct {
	GameRoot           string
	AutoDetected       bool
	CandidateCount     int
	Installed          bool
	InstallPath        string
	Status             string
	InstallationKnown  bool
}

// NewModel combines best-effort autodetection with current installation state.
// Detection errors are intentionally non-fatal because Browse remains a valid
// fallback path.
func NewModel(detection gamedetect.Result, detectionErr error, installation modinstall.Status, installationErr error) Model {
	model := Model{
		CandidateCount: len(detection.Candidates),
		Installed:      installation.Installed,
		InstallPath:    installation.Path,
		InstallationKnown: installationErr == nil,
	}

	if root, ok := detection.Unique(); ok {
		model.GameRoot = root
		model.AutoDetected = true
	}

	switch {
	case detectionErr != nil && installationErr != nil:
		model.Status = fmt.Sprintf("Game detection failed: %v. Installation status unavailable: %v. Use Browse to select KCD2.", detectionErr, installationErr)
	case detectionErr != nil:
		model.Status = fmt.Sprintf("Game detection failed: %v. Use Browse to select KCD2.", detectionErr)
	case len(detection.Candidates) == 0:
		model.Status = "KCD2 was not found automatically. Use Browse to select the game folder."
	case len(detection.Candidates) > 1:
		model.Status = fmt.Sprintf("Found %d possible KCD2 installations. Use Browse to choose one.", len(detection.Candidates))
	case installationErr != nil:
		model.Status = fmt.Sprintf("KCD2 found automatically. Installation status unavailable: %v", installationErr)
	case installation.Installed:
		model.Status = "KCD2 found automatically. Dual Subtitles is installed."
	default:
		model.Status = "KCD2 found automatically. Ready to generate and install."
	}
	return model
}

// GenerateButtonLabel reflects whether the next operation is an initial install
// or regeneration of the existing generated mod.
func (m Model) GenerateButtonLabel() string {
	if m.Installed {
		return "Regenerate"
	}
	return "Generate and install"
}
