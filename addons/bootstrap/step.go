package bootstrap

type step struct {
	name         string
	run          func(*Context) error
	sim          func(*Context) error
	dependencies map[string]struct{}
}
