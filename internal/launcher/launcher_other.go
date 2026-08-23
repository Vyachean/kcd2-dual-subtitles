//go:build !windows

package launcher

import (
	"os"

	"github.com/Vyachean/kcd2-dual-subtitles/internal/cli"
)

// Run preserves the existing CLI and interactive development fallback on
// non-Windows systems.
func Run(args []string, version string) int {
	return cli.Run(args, os.Stdin, os.Stdout, os.Stderr, version)
}
