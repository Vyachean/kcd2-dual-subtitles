package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/generator"
	"github.com/Vyachean/kcd2-dual-subtitles/internal/localization"
)

const (
	AppName = "kcd2-dual-subtitles"

	ExitSuccess = 0
	ExitRuntime = 1
	ExitUsage   = 2

	defaultMainLanguage      = "Russian"
	defaultSecondaryLanguage = "English"
)

type generateFunc func(generator.Request) (generator.Result, error)

// Run executes the command-line interface and returns the process exit code.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer, version string) int {
	return run(args, stdin, stdout, stderr, version, generator.Generate)
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, version string, generate generateFunc) int {
	if len(args) == 0 {
		return runInteractive(stdin, stdout, stderr, generate)
	}

	switch args[0] {
	case "help", "--help", "-h":
		printUsage(stdout)
		return ExitSuccess
	case "version", "--version":
		fmt.Fprintf(stdout, "%s %s\n", AppName, version)
		return ExitSuccess
	case "generate":
		return runGenerate(args[1:], stdout, stderr, generate)
	default:
		fmt.Fprintf(stderr, "error: unknown command %q\n\n", args[0])
		printUsage(stderr)
		return ExitUsage
	}
}

func runGenerate(args []string, stdout, stderr io.Writer, generate generateFunc) int {
	flags := flag.NewFlagSet("generate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	gameRoot := flags.String("game", "", "KCD2 game root containing Localization")
	mainName := flags.String("main", defaultMainLanguage, "main subtitle language")
	secondaryName := flags.String("secondary", defaultSecondaryLanguage, "secondary subtitle language")
	outputPath := flags.String("output", "", "write a portable mod ZIP instead of installing it")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printGenerateUsage(stdout)
			return ExitSuccess
		}
		fmt.Fprintf(stderr, "error: %v\n\n", err)
		printGenerateUsage(stderr)
		return ExitUsage
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "error: unexpected positional arguments: %s\n\n", strings.Join(flags.Args(), " "))
		printGenerateUsage(stderr)
		return ExitUsage
	}
	if strings.TrimSpace(*gameRoot) == "" {
		fmt.Fprintln(stderr, "error: --game is required")
		fmt.Fprintln(stderr)
		printGenerateUsage(stderr)
		return ExitUsage
	}

	mainLanguage, ok := localization.ParseLanguage(*mainName)
	if !ok {
		fmt.Fprintf(stderr, "error: unsupported main language %q\n", *mainName)
		return ExitUsage
	}
	secondaryLanguage, ok := localization.ParseLanguage(*secondaryName)
	if !ok {
		fmt.Fprintf(stderr, "error: unsupported secondary language %q\n", *secondaryName)
		return ExitUsage
	}

	request := generator.Request{
		GameRoot:          strings.TrimSpace(*gameRoot),
		MainLanguage:      mainLanguage,
		SecondaryLanguage: secondaryLanguage,
		OutputPath:        strings.TrimSpace(*outputPath),
	}
	return executeGeneration(request, stdout, stderr, generate)
}

func runInteractive(stdin io.Reader, stdout, stderr io.Writer, generate generateFunc) int {
	reader := bufio.NewReader(stdin)

	gameRoot, err := prompt(reader, stdout, "KCD2 game root: ")
	if err != nil {
		fmt.Fprintf(stderr, "error: read game root: %v\n", err)
		return ExitUsage
	}
	gameRoot = normalizeInteractivePath(gameRoot)
	if gameRoot == "" {
		fmt.Fprintln(stderr, "error: game root is required")
		return ExitUsage
	}

	mainName, err := prompt(reader, stdout, "Main language [Russian]: ")
	if err != nil {
		fmt.Fprintf(stderr, "error: read main language: %v\n", err)
		return ExitUsage
	}
	if strings.TrimSpace(mainName) == "" {
		mainName = defaultMainLanguage
	}

	secondaryName, err := prompt(reader, stdout, "Secondary language [English]: ")
	if err != nil {
		fmt.Fprintf(stderr, "error: read secondary language: %v\n", err)
		return ExitUsage
	}
	if strings.TrimSpace(secondaryName) == "" {
		secondaryName = defaultSecondaryLanguage
	}

	mainLanguage, ok := localization.ParseLanguage(mainName)
	if !ok {
		fmt.Fprintf(stderr, "error: unsupported main language %q\n", strings.TrimSpace(mainName))
		return ExitUsage
	}
	secondaryLanguage, ok := localization.ParseLanguage(secondaryName)
	if !ok {
		fmt.Fprintf(stderr, "error: unsupported secondary language %q\n", strings.TrimSpace(secondaryName))
		return ExitUsage
	}

	return executeGeneration(generator.Request{
		GameRoot:          gameRoot,
		MainLanguage:      mainLanguage,
		SecondaryLanguage: secondaryLanguage,
	}, stdout, stderr, generate)
}

func executeGeneration(request generator.Request, stdout, stderr io.Writer, generate generateFunc) int {
	result, err := generate(request)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		if errors.Is(err, generator.ErrInvalidRequest) {
			return ExitUsage
		}
		return ExitRuntime
	}

	if result.InstallPath != "" {
		fmt.Fprintf(stdout, "Installed: %s\n", result.InstallPath)
	} else {
		fmt.Fprintf(stdout, "Created: %s\n", result.OutputPath)
	}
	fmt.Fprintf(stdout, "Processed: %d\n", result.Stats.Processed)
	fmt.Fprintf(stdout, "Bilingual: %d\n", result.Stats.Bilingual)
	fmt.Fprintf(stdout, "Identical: %d\n", result.Stats.Identical)
	fmt.Fprintf(stdout, "Missing secondary: %d\n", result.Stats.MissingSecondary)
	fmt.Fprintf(stdout, "Main empty fallback: %d\n", result.Stats.MainEmptyFallback)
	fmt.Fprintf(stdout, "Secondary empty fallback: %d\n", result.Stats.SecondaryEmptyFallback)
	fmt.Fprintf(stdout, "Secondary only: %d\n", result.Stats.SecondaryOnly)
	return ExitSuccess
}

func prompt(reader *bufio.Reader, output io.Writer, label string) (string, error) {
	fmt.Fprint(output, label)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if errors.Is(err, io.EOF) && line == "" {
		return "", io.EOF
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func normalizeInteractivePath(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		value = strings.TrimSpace(value[1 : len(value)-1])
	}
	return value
}

func printUsage(output io.Writer) {
	fmt.Fprintf(output, "Usage:\n")
	fmt.Fprintf(output, "  %s generate --game <KCD2-root> [--main Russian] [--secondary English]\n", AppName)
	fmt.Fprintf(output, "  %s generate --game <KCD2-root> [--main Russian] [--secondary English] --output <mod.zip>\n", AppName)
	fmt.Fprintf(output, "  %s --help\n", AppName)
	fmt.Fprintf(output, "  %s --version\n", AppName)
	fmt.Fprintf(output, "  %s              # interactive install mode\n", AppName)
}

func printGenerateUsage(output io.Writer) {
	fmt.Fprintf(output, "Usage:\n")
	fmt.Fprintf(output, "  %s generate --game <KCD2-root> [--main Russian] [--secondary English]\n", AppName)
	fmt.Fprintf(output, "  %s generate --game <KCD2-root> [--main Russian] [--secondary English] --output <mod.zip>\n", AppName)
	fmt.Fprintf(output, "\nWithout --output, the Windows build installs into Documents\\kingdomcome_mods.\n")
	fmt.Fprintf(output, "Supported languages: English, Russian\n")
}
