package internal

import "strconv"

type Config struct {
	FakeServerImage string
	ExposedPort     int
	StackHooks      []func(*Config) error
}
type Option func(*Config)

func (cfg *Config) Args() []string {
	return []string{
		"-scheme", "http",
		"-port", strconv.Itoa(cfg.ExposedPort),
		"-external-url", "http://localhost:" + strconv.Itoa(cfg.ExposedPort),
		"-public-host", "localhost",
	}
}
