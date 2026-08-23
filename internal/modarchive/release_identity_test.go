package modarchive

import (
	"os"
	"strings"
	"testing"
)

func TestReleaseManifestVersionEnvironment(t *testing.T) {
	expected := strings.TrimSpace(os.Getenv("KCD2DS_EXPECTED_VERSION"))
	if expected == "" {
		t.Skip("KCD2DS_EXPECTED_VERSION is set only by release-candidate CI")
	}
	manifest := string(manifestForVersion(expected))
	want := "<version>" + expected + "</version>"
	if !strings.Contains(manifest, want) {
		t.Fatalf("manifest does not contain release version %q:\n%s", want, manifest)
	}
}
