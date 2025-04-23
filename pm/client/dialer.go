package client

import (
	"context"
	"fmt"
	"net"

	"fastcat.org/go/gdev/pm/api"
)

func defaultDialer(ctx context.Context, network, addr string) (net.Conn, error) {
	a := api.ListenAddr()
	// use a unix socket instead of TCP
	if network != "tcp" && network != a.Network() {
		return nil, fmt.Errorf("must use net %s", a.Network())
	} else if addr != "localhost" && addr != "localhost:80" && addr != a.String() {
		return nil, fmt.Errorf("must use addr localhost[:80] or %s", a.String())
	}
	return net.Dial(a.Network(), a.String())
}
