package docker

// K3SProvider returns values to pass to [k3s.WithProvider]
func K3SProvider() (string, []string, func()) {
	addon.CheckNotInitialized()
	return "docker", []string{"--docker"}, func() {
		Configure()
	}
}
