package editor

import (
	"os"
	"strings"
	"sync"
)

// kittySupport caches the result of Kitty graphics protocol detection.
var (
	kittyOnce   sync.Once
	kittyResult bool
)

// SupportsKittyGraphics returns true if the terminal supports the Kitty
// graphics protocol. The result is cached for the process lifetime.
//
// Detection is based on the TERM_PROGRAM environment variable since we cannot
// reliably query the terminal through the Bubble Tea render pipeline (the VT
// emulator sits between us and the real terminal). Supported terminals:
// kitty, WezTerm, ghostty.
func SupportsKittyGraphics() bool {
	kittyOnce.Do(func() {
		kittyResult = detectKittyGraphics()
	})
	return kittyResult
}

func detectKittyGraphics() bool {
	tp := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	switch tp {
	case "kitty", "wezterm", "ghostty":
		return true
	}
	// Also check TERM for kitty direct mode
	term := strings.ToLower(os.Getenv("TERM"))
	return strings.HasPrefix(term, "xterm-kitty")
}

// ResetKittyDetection is only used by tests to reset the cached detection.
func ResetKittyDetection() {
	kittyOnce = sync.Once{}
	kittyResult = false
}
