package gui

import (
	"errors"
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/gamedetect"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/modinstall"
)

func TestNewModelAutoSelectsOnlyUniqueDetection(t *testing.T) {
	model := NewModel(
		gamedetect.Result{Candidates: []string{"game"}},
		nil,
		modinstall.Status{Installed: true, Path: "mod"},
		nil,
	)
	if model.GameRoot != "game" || !model.AutoDetected || model.CandidateCount != 1 {
		t.Fatalf("detection model = %+v", model)
	}
	if !model.Installed || !model.InstallationKnown || model.InstallPath != "mod" {
		t.Fatalf("installation model = %+v", model)
	}
	if model.GenerateButtonLabel() != "Regenerate" {
		t.Fatalf("button label = %q", model.GenerateButtonLabel())
	}
}

func TestNewModelRequiresBrowseForZeroOrMultipleCandidates(t *testing.T) {
	for _, candidates := range [][]string{nil, []string{"one", "two"}} {
		model := NewModel(gamedetect.Result{Candidates: candidates}, nil, modinstall.Status{}, nil)
		if model.GameRoot != "" || model.AutoDetected {
			t.Fatalf("candidates=%v model=%+v", candidates, model)
		}
		if !strings.Contains(model.Status, "Browse") {
			t.Fatalf("status = %q, want Browse guidance", model.Status)
		}
	}
}

func TestNewModelKeepsBrowseFallbackOnDetectionError(t *testing.T) {
	model := NewModel(gamedetect.Result{}, errors.New("drive enumeration failed"), modinstall.Status{}, nil)
	if model.AutoDetected || model.GameRoot != "" {
		t.Fatalf("model = %+v", model)
	}
	if !strings.Contains(model.Status, "Game detection failed") || !strings.Contains(model.Status, "Browse") {
		t.Fatalf("status = %q, want detection failure and Browse guidance", model.Status)
	}
	if model.GenerateButtonLabel() != "Generate and install" {
		t.Fatalf("button label = %q", model.GenerateButtonLabel())
	}
}

func TestApplyInstallationStateRefreshesButtonsWithoutReplacingOperationStatus(t *testing.T) {
	model := Model{Status: "Generation failed."}
	model.ApplyInstallationState(modinstall.Status{Installed: true, Path: "restored-mod"}, nil)
	if !model.InstallationKnown || !model.Installed || model.InstallPath != "restored-mod" {
		t.Fatalf("refreshed installation model = %+v", model)
	}
	if model.Status != "Generation failed." {
		t.Fatalf("operation status changed to %q", model.Status)
	}
	if model.GenerateButtonLabel() != "Regenerate" {
		t.Fatalf("button label = %q, want Regenerate", model.GenerateButtonLabel())
	}
}

func TestApplyInstallationStateMarksUnknownOnInspectionFailure(t *testing.T) {
	model := Model{Installed: true, InstallPath: "stale", InstallationKnown: true}
	model.ApplyInstallationState(modinstall.Status{}, errors.New("inspect denied"))
	if model.InstallationKnown || model.Installed || model.InstallPath != "" {
		t.Fatalf("failed inspection model = %+v", model)
	}
}
