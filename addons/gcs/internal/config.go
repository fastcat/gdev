package internal

type Config struct {
	FakeServerImage string
	ExposedPort     int
	StackHooks      []func(*Config) error
}
type Option func(*Config)
