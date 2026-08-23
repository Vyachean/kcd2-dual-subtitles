package cli

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/generator"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer

	exitCode := Run([]string{"--help"}, strings.NewReader(""), &stdout, &stderr, "1.2.3")

	if exitCode != ExitSuccess {
		t.Fatalf("exit code = %d, want %d", exitCode, ExitSuccess)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("stdout does not contain usage: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer

	exitCode := Run([]string{"--version"}, strings.NewReader(""), &stdout, &stderr, "1.2.3")

	if exitCode != ExitSuccess {
		t.Fatalf("exit code = %d, want %d", exitCode, ExitSuccess)
	}
	if got, want := stdout.String(), "kcd2-dual-subtitles 1.2.3\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer

	exitCode := Run([]string{"unknown"}, strings.NewReader(""), &stdout, &stderr, "dev")

	if exitCode != ExitUsage {
		t.Fatalf("exit code = %d, want %d", exitCode, ExitUsage)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("stderr = %q, want unknown-command error", stderr.String())
	}
}

func TestRunGenerateRejectsMissingArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer

	exitCode := Run([]string{"generate", "--game", "somewhere"}, strings.NewReader(""), &stdout, &stderr, "dev")

	if exitCode != ExitUsage {
		t.Fatalf("exit code = %d, want %d", exitCode, ExitUsage)
	}
	if !strings.Contains(stderr.String(), "are required") {
		t.Fatalf("stderr = %q, want required-arguments error", stderr.String())
	}
}

func TestRunGenerateRejectsUnsupportedLanguage(t *testing.T) {
	var stdout, stderr bytes.Buffer

	exitCode := Run([]string{
		"generate",
		"--game", "somewhere",
		"--main", "Klingon",
		"--secondary", "English",
		"--output", "mod.zip",
	}, strings.NewReader(""), &stdout, &stderr, "dev")

	if exitCode != ExitUsage {
		t.Fatalf("exit code = %d, want %d", exitCode, ExitUsage)
	}
	if !strings.Contains(stderr.String(), "unsupported main language") {
		t.Fatalf("stderr = %q, want unsupported-language error", stderr.String())
	}
}

func TestRunGenerateRejectsSameLanguageAsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer

	exitCode := Run([]string{
		"generate",
		"--game", "somewhere",
		"--main", "Russian",
		"--secondary", "russian",
		"--output", "mod.zip",
	}, strings.NewReader(""), &stdout, &stderr, "dev")

	if exitCode != ExitUsage {
		t.Fatalf("exit code = %d, want %d", exitCode, ExitUsage)
	}
	if !strings.Contains(stderr.String(), "main and secondary languages must differ") {
		t.Fatalf("stderr = %q, want same-language error", stderr.String())
	}
}

func TestRunGenerateEndToEndXboxStyleRoot(t *testing.T) {
	gameRoot := t.TempDir()
	localizationDir := filepath.Join(gameRoot, "Localization")
	if err := os.Mkdir(localizationDir, 0o755); err != nil {
		t.Fatalf("create Localization: %v", err)
	}

	writeDialoguePAK(t, filepath.Join(localizationDir, "Russian_xml.pak"), `<?xml version="1.0" encoding="utf-8"?>
<Table>
<Row><Cell>dialog_one</Cell><Cell>source</Cell><Cell>Привет.</Cell></Row>
</Table>`)
	writeDialoguePAK(t, filepath.Join(localizationDir, "English_xml.pak"), `<?xml version="1.0" encoding="utf-8"?>
<Table>
<Row><Cell>dialog_one</Cell><Cell>source</Cell><Cell>Hello.</Cell></Row>
</Table>`)

	outputPath := filepath.Join(t.TempDir(), "dual.zip")
	var stdout, stderr bytes.Buffer

	exitCode := Run([]string{
		"generate",
		"--game", gameRoot,
		"--main", "rUsSiAn",
		"--secondary", " english ",
		"--output", outputPath,
	}, strings.NewReader(""), &stdout, &stderr, "dev")

	if exitCode != ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stderr=%q", exitCode, ExitSuccess, stderr.String())
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("generated output not found: %v", err)
	}
	if !strings.Contains(stdout.String(), "Bilingual: 1") {
		t.Fatalf("stdout = %q, want bilingual statistics", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(gameRoot, "KingdomCome.exe")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("test root unexpectedly contains KingdomCome.exe: %v", err)
	}
}

func TestRunGenerateInvalidRootReturnsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer

	exitCode := Run([]string{
		"generate",
		"--game", t.TempDir(),
		"--main", "Russian",
		"--secondary", "English",
		"--output", filepath.Join(t.TempDir(), "mod.zip"),
	}, strings.NewReader(""), &stdout, &stderr, "dev")

	if exitCode != ExitUsage {
		t.Fatalf("exit code = %d, want %d", exitCode, ExitUsage)
	}
	if !strings.Contains(stderr.String(), "Localization directory not found") {
		t.Fatalf("stderr = %q, want invalid-root error", stderr.String())
	}
}

func TestRunGenerateMissingLanguagePAKReturnsUsageError(t *testing.T) {
	gameRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(gameRoot, "Localization"), 0o755); err != nil {
		t.Fatalf("create Localization: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := Run([]string{
		"generate",
		"--game", gameRoot,
		"--main", "Russian",
		"--secondary", "English",
		"--output", filepath.Join(t.TempDir(), "mod.zip"),
	}, strings.NewReader(""), &stdout, &stderr, "dev")

	if exitCode != ExitUsage {
		t.Fatalf("exit code = %d, want %d", exitCode, ExitUsage)
	}
	if !strings.Contains(stderr.String(), "main language PAK") {
		t.Fatalf("stderr = %q, want missing-PAK error", stderr.String())
	}
}

func TestRunGenerateExistingOutputReturnsRuntimeError(t *testing.T) {
	gameRoot := makeValidGameRoot(t)
	outputPath := filepath.Join(t.TempDir(), "already-exists.zip")
	if err := os.WriteFile(outputPath, []byte("existing"), 0o644); err != nil {
		t.Fatalf("create existing output: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := Run([]string{
		"generate",
		"--game", gameRoot,
		"--main", "Russian",
		"--secondary", "English",
		"--output", outputPath,
	}, strings.NewReader(""), &stdout, &stderr, "dev")

	if exitCode != ExitRuntime {
		t.Fatalf("exit code = %d, want %d", exitCode, ExitRuntime)
	}
	if !strings.Contains(stderr.String(), "output path already exists") {
		t.Fatalf("stderr = %q, want existing-output error", stderr.String())
	}
}

func TestInteractiveDefaultsAndQuotedPaths(t *testing.T) {
	gameRoot := `C:\XboxGames\Kingdom Come- Deliverance II\Content`
	outputPath := `C:\Users\Player\Desktop\dual subtitles.zip`
	input := fmt.Sprintf("\"%s\"\n\n\n\"%s\"\n", gameRoot, outputPath)

	var gotRequest generator.Request
	fakeGenerate := func(request generator.Request) (generator.Result, error) {
		gotRequest = request
		return generator.Result{
			OutputPath: request.OutputPath,
			Stats: localization.MergeStats{
				Processed: 10,
				Bilingual: 9,
				Identical: 1,
			},
		}, nil
	}

	var stdout, stderr bytes.Buffer
	exitCode := run(nil, strings.NewReader(input), &stdout, &stderr, "dev", fakeGenerate)

	if exitCode != ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stderr=%q", exitCode, ExitSuccess, stderr.String())
	}
	if gotRequest.GameRoot != gameRoot {
		t.Fatalf("GameRoot = %q, want %q", gotRequest.GameRoot, gameRoot)
	}
	if gotRequest.MainLanguage != localization.Russian {
		t.Fatalf("MainLanguage = %q, want Russian", gotRequest.MainLanguage)
	}
	if gotRequest.SecondaryLanguage != localization.English {
		t.Fatalf("SecondaryLanguage = %q, want English", gotRequest.SecondaryLanguage)
	}
	if gotRequest.OutputPath != outputPath {
		t.Fatalf("OutputPath = %q, want %q", gotRequest.OutputPath, outputPath)
	}
	if !strings.Contains(stdout.String(), "Main language [Russian]") {
		t.Fatalf("stdout = %q, want interactive prompts", stdout.String())
	}
}

func TestExecuteGenerationClassifiesRuntimeFailure(t *testing.T) {
	fakeGenerate := func(generator.Request) (generator.Result, error) {
		return generator.Result{}, errors.New("disk failed")
	}

	var stdout, stderr bytes.Buffer
	exitCode := executeGeneration(generator.Request{}, &stdout, &stderr, fakeGenerate)

	if exitCode != ExitRuntime {
		t.Fatalf("exit code = %d, want %d", exitCode, ExitRuntime)
	}
	if !strings.Contains(stderr.String(), "disk failed") {
		t.Fatalf("stderr = %q, want runtime error", stderr.String())
	}
}

func makeValidGameRoot(t *testing.T) string {
	t.Helper()

	gameRoot := t.TempDir()
	localizationDir := filepath.Join(gameRoot, "Localization")
	if err := os.Mkdir(localizationDir, 0o755); err != nil {
		t.Fatalf("create Localization: %v", err)
	}

	writeDialoguePAK(t, filepath.Join(localizationDir, "Russian_xml.pak"), `<Table><Row><Cell>id</Cell><Cell>source</Cell><Cell>Привет.</Cell></Row></Table>`)
	writeDialoguePAK(t, filepath.Join(localizationDir, "English_xml.pak"), `<Table><Row><Cell>id</Cell><Cell>source</Cell><Cell>Hello.</Cell></Row></Table>`)
	return gameRoot
}

func writeDialoguePAK(t *testing.T, path, dialogueXML string) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create PAK %s: %v", path, err)
	}

	writer := zip.NewWriter(file)
	entry, err := writer.Create(localization.DialogueXMLArchivePath)
	if err != nil {
		_ = writer.Close()
		_ = file.Close()
		t.Fatalf("create dialogue entry: %v", err)
	}
	if _, err := entry.Write([]byte(dialogueXML)); err != nil {
		_ = writer.Close()
		_ = file.Close()
		t.Fatalf("write dialogue entry: %v", err)
	}
	if err := writer.Close(); err != nil {
		_ = file.Close()
		t.Fatalf("close PAK writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close PAK file: %v", err)
	}
}
