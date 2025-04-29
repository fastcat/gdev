package containerd

// K3SProvider returns values to pass to [k3s.WithProvider]
func K3SProvider() (string, []string, func()) {
	return "containerd", nil, func() {
		Configure(
			WithAddress("/run/k3s/containerd/containerd.sock"),
		)
	}
}
