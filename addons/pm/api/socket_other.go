//go:build !linux

package api

import (
	"fmt"
	"net"
	"runtime"
)

func ListenAddr() net.Addr {
	panic(fmt.Errorf("not supported on %s", runtime.GOOS))
}
