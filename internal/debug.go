package internal

import (
	"os"
	"strings"
)

func IsDebuggerAttached() bool {
	contents, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return false
	}
	for l := range strings.Lines(string(contents)) {
		if !strings.HasPrefix(l, "TracerPid:") {
			continue
		}
		fields := strings.Fields(l)
		if len(fields) < 2 {
			continue
		}
		if fields[1] == "0" {
			return false
		}
		return true
	}
	return false
}
