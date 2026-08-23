package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/generator"
)

func TestRunGeneratePassesDifferentiatedSubtitleStyle(t *testing.T) {
	var gotRequest generator.Request
	fakeGenerate := func(request generator.Request) (generator.Result, error) {
		gotRequest = request
		return generator.Result{
			InstallPath:   `C:\Documents\kingdomcome_mods\kcd_dual_subtitles`,
			SubtitleStyle: request.SubtitleStyle,
		}, nil
	}

	var stdout, stderr bytes.Buffer
	exitCode := run([]string{
		"generate",
		"--game", "somewhere",
		"--subtitle-style", "DiFfErEnTiAtEd",
	}, strings.NewReader(""), &stdout, &stderr, "dev", fakeGenerate)
	if exitCode != ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stderr=%q", exitCode, ExitSuccess, stderr.String())
	}
	if gotRequest.SubtitleStyle != generator.SubtitleStyleDifferentiated {
		t.Fatalf("SubtitleStyle = %q, want %q", gotRequest.SubtitleStyle, generator.SubtitleStyleDifferentiated)
	}
	if !strings.Contains(stdout.String(), "Experimental subtitle style: differentiated") {
		t.Fatalf("stdout = %q, want differentiated style report", stdout.String())
	}
}

func TestRunGenerateDefaultsToTaggedSubtitleStyle(t *testing.T) {
	var gotRequest generator.Request
	fakeGenerate := func(request generator.Request) (generator.Result, error) {
		gotRequest = request
		return generator.Result{
			InstallPath:   `C:\Documents\kingdomcome_mods\kcd_dual_subtitles`,
			SubtitleStyle: request.SubtitleStyle,
		}, nil
	}

	var stdout, stderr bytes.Buffer
	exitCode := run([]string{
		"generate",
		"--game", "somewhere",
	}, strings.NewReader(""), &stdout, &stderr, "dev", fakeGenerate)
	if exitCode != ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stderr=%q", exitCode, ExitSuccess, stderr.String())
	}
	if gotRequest.SubtitleStyle != generator.SubtitleStyleTagged {
		t.Fatalf("SubtitleStyle = %q, want %q", gotRequest.SubtitleStyle, generator.SubtitleStyleTagged)
	}
	if strings.Contains(stdout.String(), "subtitle style") {
		t.Fatalf("default tagged mode changed normal CLI output: %q", stdout.String())
	}
}

func TestRunGenerateRejectsUnsupportedSubtitleStyle(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := Run([]string{
		"generate",
		"--game", "somewhere",
		"--subtitle-style", "rainbow",
	}, strings.NewReader(""), &stdout, &stderr, "dev")
	if exitCode != ExitUsage {
		t.Fatalf("exit code = %d, want %d", exitCode, ExitUsage)
	}
	if !strings.Contains(stderr.String(), "unsupported subtitle style") {
		t.Fatalf("stderr = %q, want unsupported subtitle style error", stderr.String())
	}
}
