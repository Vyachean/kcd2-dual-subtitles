package cli

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/generator"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

func TestRunHelpAndVersion(t *testing.T) {
	t.Run("help", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := Run([]string{"--help"}, strings.NewReader(""), &stdout, &stderr, "1.2.3")
		if exitCode != ExitSuccess {
			t.Fatalf("exit code = %d, want %d", exitCode, ExitSuccess)
		}
		if !strings.Contains(stdout.String(), "Usage:") || !strings.Contains(stdout.String(), "--output") {
			t.Fatalf("stdout does not contain install/archive usage: %q", stdout.String())
		}
		if stderr.Len() != 0 {
			t.Fatalf("stderr = %q, want empty", stderr.String())
		}
	})

	t.Run("version", func(t *testing.T) {
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
	})
}

func TestRunRejectsUnknownCommandAndMissingGame(t *testing.T) {
	t.Run("unknown command", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := Run([]string{"unknown"}, strings.NewReader(""), &stdout, &stderr, "dev")
		if exitCode != ExitUsage || !strings.Contains(stderr.String(), "unknown command") {
			t.Fatalf("exit=%d stderr=%q, want unknown-command usage error", exitCode, stderr.String())
		}
	})

	t.Run("missing game", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := Run([]string{"generate"}, strings.NewReader(""), &stdout, &stderr, "dev")
		if exitCode != ExitUsage || !strings.Contains(stderr.String(), "--game is required") {
			t.Fatalf("exit=%d stderr=%q, want missing-game usage error", exitCode, stderr.String())
		}
	})
}

func TestRunGenerateDefaultsToInstallAndRussianEnglish(t *testing.T) {
	var gotRequest generator.Request
	fakeGenerate := func(request generator.Request) (generator.Result, error) {
		gotRequest = request
		return generator.Result{
			InstallPath: `C:\Users\Player\Documents\kingdomcome_mods\kcd_dual_subtitles`,
			Stats: localization.MergeStats{
				Processed: 10,
				Bilingual: 9,
				Identical: 1,
			},
		}, nil
	}

	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"generate", "--game", `C:\XboxGames\KCD2\Content`}, strings.NewReader(""), &stdout, &stderr, "dev", fakeGenerate)
	if exitCode != ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stderr=%q", exitCode, ExitSuccess, stderr.String())
	}
	if gotRequest.MainLanguage != localization.Russian || gotRequest.SecondaryLanguage != localization.English {
		t.Fatalf("languages = %q/%q, want Russian/English", gotRequest.MainLanguage, gotRequest.SecondaryLanguage)
	}
	if gotRequest.OutputPath != "" {
		t.Fatalf("OutputPath = %q, want install mode", gotRequest.OutputPath)
	}
	if !strings.Contains(stdout.String(), "Installed:") || !strings.Contains(stdout.String(), "Bilingual: 9") {
		t.Fatalf("stdout = %q, want installed path and statistics", stdout.String())
	}
}

func TestRunGenerateAllowsLanguageOverridesInInstallMode(t *testing.T) {
	var gotRequest generator.Request
	fakeGenerate := func(request generator.Request) (generator.Result, error) {
		gotRequest = request
		return generator.Result{InstallPath: `C:\Documents\kingdomcome_mods\kcd_dual_subtitles`}, nil
	}

	var stdout, stderr bytes.Buffer
	exitCode := run([]string{
		"generate",
		"--game", "somewhere",
		"--main", "English",
		"--secondary", "Russian",
	}, strings.NewReader(""), &stdout, &stderr, "dev", fakeGenerate)
	if exitCode != ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stderr=%q", exitCode, ExitSuccess, stderr.String())
	}
	if gotRequest.MainLanguage != localization.English || gotRequest.SecondaryLanguage != localization.Russian {
		t.Fatalf("languages = %q/%q, want English/Russian", gotRequest.MainLanguage, gotRequest.SecondaryLanguage)
	}
}

func TestRunGenerateRejectsUnsupportedOrSameLanguage(t *testing.T) {
	t.Run("unsupported", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := Run([]string{
			"generate", "--game", "somewhere", "--main", "Klingon",
		}, strings.NewReader(""), &stdout, &stderr, "dev")
		if exitCode != ExitUsage || !strings.Contains(stderr.String(), "unsupported main language") {
			t.Fatalf("exit=%d stderr=%q, want unsupported-language usage error", exitCode, stderr.String())
		}
	})

	t.Run("same language", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := Run([]string{
			"generate",
			"--game", "somewhere",
			"--main", "Russian",
			"--secondary", "russian",
			"--output", "mod.zip",
		}, strings.NewReader(""), &stdout, &stderr, "dev")
		if exitCode != ExitUsage || !strings.Contains(stderr.String(), "main and secondary languages must differ") {
			t.Fatalf("exit=%d stderr=%q, want same-language usage error", exitCode, stderr.String())
		}
	})
}

func TestRunGenerateOutputKeepsPortableArchiveMode(t *testing.T) {
	gameRoot := makeValidGameRoot(t)
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
	if info, err := os.Stat(outputPath); err != nil || info.IsDir() {
		t.Fatalf("generated archive missing: info=%v err=%v", info, err)
	}
	if !strings.Contains(stdout.String(), "Created: "+outputPath) || !strings.Contains(stdout.String(), "Bilingual: 1") {
		t.Fatalf("stdout = %q, want archive path and statistics", stdout.String())
	}
	if strings.Contains(stdout.String(), "Installed:") {
		t.Fatalf("archive mode reported installation: %q", stdout.String())
	}
}

func TestRunGenerateArchiveErrorsKeepExitClasses(t *testing.T) {
	t.Run("invalid root", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := Run([]string{
			"generate", "--game", t.TempDir(), "--output", filepath.Join(t.TempDir(), "mod.zip"),
		}, strings.NewReader(""), &stdout, &stderr, "dev")
		if exitCode != ExitUsage || !strings.Contains(stderr.String(), "Localization directory not found") {
			t.Fatalf("exit=%d stderr=%q, want invalid-root usage error", exitCode, stderr.String())
		}
	})

	t.Run("existing output", func(t *testing.T) {
		gameRoot := makeValidGameRoot(t)
		outputPath := filepath.Join(t.TempDir(), "already-exists.zip")
		if err := os.WriteFile(outputPath, []byte("existing"), 0o644); err != nil {
			t.Fatalf("create existing output: %v", err)
		}
		var stdout, stderr bytes.Buffer
		exitCode := Run([]string{
			"generate", "--game", gameRoot, "--output", outputPath,
		}, strings.NewReader(""), &stdout, &stderr, "dev")
		if exitCode != ExitRuntime || !strings.Contains(stderr.String(), "output path already exists") {
			t.Fatalf("exit=%d stderr=%q, want existing-output runtime error", exitCode, stderr.String())
		}
	})
}

func TestInteractiveDefaultsToInstallAndNormalizesQuotedGamePath(t *testing.T) {
	gameRoot := `C:\XboxGames\Kingdom Come- Deliverance II\Content`
	input := "\"" + gameRoot + "\"\n\n\n"

	var gotRequest generator.Request
	fakeGenerate := func(request generator.Request) (generator.Result, error) {
		gotRequest = request
		return generator.Result{InstallPath: `C:\Users\Player\Documents\kingdomcome_mods\kcd_dual_subtitles`}, nil
	}

	var stdout, stderr bytes.Buffer
	exitCode := run(nil, strings.NewReader(input), &stdout, &stderr, "dev", fakeGenerate)
	if exitCode != ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stderr=%q", exitCode, ExitSuccess, stderr.String())
	}
	if gotRequest.GameRoot != gameRoot {
		t.Fatalf("GameRoot = %q, want %q", gotRequest.GameRoot, gameRoot)
	}
	if gotRequest.MainLanguage != localization.Russian || gotRequest.SecondaryLanguage != localization.English {
		t.Fatalf("languages = %q/%q, want Russian/English", gotRequest.MainLanguage, gotRequest.SecondaryLanguage)
	}
	if gotRequest.OutputPath != "" {
		t.Fatalf("OutputPath = %q, want install mode", gotRequest.OutputPath)
	}
	if strings.Contains(stdout.String(), "Output path") {
		t.Fatalf("interactive install mode unexpectedly prompted for output ZIP: %q", stdout.String())
	}
}

func TestExecuteGenerationClassifiesRuntimeFailure(t *testing.T) {
	fakeGenerate := func(generator.Request) (generator.Result, error) {
		return generator.Result{}, errors.New("disk failed")
	}

	var stdout, stderr bytes.Buffer
	exitCode := executeGeneration(generator.Request{}, &stdout, &stderr, fakeGenerate)
	if exitCode != ExitRuntime || !strings.Contains(stderr.String(), "disk failed") {
		t.Fatalf("exit=%d stderr=%q, want runtime failure", exitCode, stderr.String())
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
