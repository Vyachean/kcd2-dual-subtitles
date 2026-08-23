package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/generator"
)

func TestRunGeneratePropagatesBuildVersionAndCanary(t *testing.T) {
	var got generator.Request
	fakeGenerate := func(request generator.Request) (generator.Result, error) {
		got = request
		return generator.Result{
			InstallPath: `C:\Users\Player\Documents\kingdomcome_mods\kcd_dual_subtitles`,
			PatchRows:   42,
			CanaryID:    request.CanaryID,
		}, nil
	}

	var stdout, stderr bytes.Buffer
	exitCode := run([]string{
		"generate",
		"--game", `C:\XboxGames\KCD2\Content`,
		"--canary-id", "visible_row",
	}, strings.NewReader(""), &stdout, &stderr, "v0.1.0-rc.4", fakeGenerate)
	if exitCode != ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stderr=%q", exitCode, ExitSuccess, stderr.String())
	}
	if got.Version != "v0.1.0-rc.4" {
		t.Fatalf("Request.Version = %q, want v0.1.0-rc.4", got.Version)
	}
	if got.CanaryID != "visible_row" {
		t.Fatalf("Request.CanaryID = %q, want visible_row", got.CanaryID)
	}
	if !strings.Contains(stdout.String(), "Patch rows: 42") || !strings.Contains(stdout.String(), "Diagnostic canary enabled") {
		t.Fatalf("stdout = %q, want patch count and canary warning", stdout.String())
	}
}

func TestRunGenerateWithoutCanaryLeavesCanaryEmpty(t *testing.T) {
	var got generator.Request
	fakeGenerate := func(request generator.Request) (generator.Result, error) {
		got = request
		return generator.Result{InstallPath: `C:\Documents\kingdomcome_mods\kcd_dual_subtitles`}, nil
	}

	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"generate", "--game", "somewhere"}, strings.NewReader(""), &stdout, &stderr, "dev", fakeGenerate)
	if exitCode != ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stderr=%q", exitCode, ExitSuccess, stderr.String())
	}
	if got.CanaryID != "" {
		t.Fatalf("Request.CanaryID = %q, want empty", got.CanaryID)
	}
	if strings.Contains(stdout.String(), "Diagnostic canary") {
		t.Fatalf("normal generation reported canary: %q", stdout.String())
	}
}
