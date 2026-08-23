//go:build !windows

package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunGenerateDefaultInstallOutsideWindowsExplainsOutputFallback(t *testing.T) {
	gameRoot := makeValidGameRoot(t)
	var stdout, stderr bytes.Buffer

	exitCode := Run([]string{"generate", "--game", gameRoot}, strings.NewReader(""), &stdout, &stderr, "dev")
	if exitCode != ExitRuntime {
		t.Fatalf("exit code = %d, want %d", exitCode, ExitRuntime)
	}
	if !strings.Contains(stderr.String(), "supported only on Windows") || !strings.Contains(stderr.String(), "--output") {
		t.Fatalf("stderr = %q, want Windows-only install guidance with --output fallback", stderr.String())
	}
}
