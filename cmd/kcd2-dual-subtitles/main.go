package main

import (
	"os"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/cli"
)

var version = "dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, version))
}
