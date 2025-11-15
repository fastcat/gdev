//go:build !linux

package sys

import (
	"fmt"
	"runtime"
)

func init() {
	GetIsolator = func() (Isolator, error) {
		return nil, fmt.Errorf("no process isolation defined for %s", runtime.GOOS)
	}
}
