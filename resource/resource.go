package resource

type Resource interface {
	ID() string
	Start(ctx *Context) error
	Stop(ctx *Context) error
	Ready(ctx *Context) (bool, error) // TODO: provide not-ready details
}
