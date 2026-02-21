package editor

import (
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	debugOnce sync.Once
	debugFile *os.File
)

func debugf(format string, args ...any) {
	if os.Getenv("KOPR_DEBUG_RESIZE") == "" {
		return
	}
	debugOnce.Do(func() {
		f, err := os.OpenFile("/tmp/kopr-resize-debug.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return
		}
		debugFile = f
	})
	if debugFile == nil {
		return
	}
	fmt.Fprintf(debugFile, "%s "+format+"\n", append([]any{time.Now().Format(time.RFC3339Nano)}, args...)...) //nolint:errcheck // debug log
}
