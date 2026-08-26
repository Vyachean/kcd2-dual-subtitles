package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/generator"
)

func TestExecuteGenerationReportsLocalizationOverrides(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := executeGeneration(generator.Request{}, &stdout, &stderr, func(generator.Request) (generator.Result, error) {
		return generator.Result{
			OutputPath:                     "out.zip",
			MainLocalizationOverrides:      []string{"Main Fix"},
			SecondaryLocalizationOverrides: []string{"Secondary Fix A", "Secondary Fix B"},
		}, nil
	})
	if code != ExitSuccess {
		t.Fatalf("executeGeneration() = %d, stderr=%q", code, stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		"Main localization overrides: Main Fix",
		"Secondary localization overrides: Secondary Fix A, Secondary Fix B",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output %q does not contain %q", output, want)
		}
	}
}
