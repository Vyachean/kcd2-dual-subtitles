//go:build !windows

package gamedetect

// Detect returns no automatic candidates on non-Windows systems. Manual path
// selection/explicit CLI paths remain usable for development portability.
func Detect() (Result, error) {
	return Result{}, nil
}
