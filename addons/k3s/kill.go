package k3s

import "context"

func killPods(ctx context.Context) error {
	// TODO: find all containerd-shim-ish processes whose exec is in
	// /var/lib/rancher/k3s/data and kill them and all their children.
	panic("unimplemented")
}
