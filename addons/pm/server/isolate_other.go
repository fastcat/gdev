//go:build !linux

package server

import (
	"fmt"
	"runtime"
)

func init() {
	getIsolator = func() (isolator, error) {
		return nil, fmt.Errorf("no process isolation defined for %s", runtime.GOOS)
	}
}
