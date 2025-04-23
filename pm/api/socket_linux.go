package api

import (
	"fmt"
	"net"
	"os"

	"fastcat.org/go/gdev/instance"
)

func ListenAddr() net.Addr {
	return &net.UnixAddr{
		Net:  "unixpacket",
		Name: fmt.Sprintf("/run/user/%d/%s-pm", os.Getuid(), instance.AppName),
	}
}
