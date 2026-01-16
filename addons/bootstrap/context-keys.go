package bootstrap

var skipLoginsKey = NewKey[bool]("bootstrap.skipLogins")

// SkipLogins returns whether login steps should be skipped. This is set by a
// command line argument. Custom bootstrap steps/plans should obey this.
func SkipLogins(ctx *Context) bool {
	v, ok := Get(ctx, skipLoginsKey)
	return ok && v
}

func SkipIfNoLogins() StepOpt {
	return SkipFunc(func(ctx *Context) (bool, error) {
		return SkipLogins(ctx), nil
	})
}
