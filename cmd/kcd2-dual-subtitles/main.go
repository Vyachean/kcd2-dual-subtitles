package main

import (
	"os"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/launcher"
)

var version = "dev"

func main() {
	os.Exit(launcher.Run(os.Args[1:], version))
}
