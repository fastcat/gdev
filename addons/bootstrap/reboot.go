package bootstrap

var needsRebootKey = NewKey[bool]("need-reboot-after-bootstrap")

func SetNeedsReboot(ctx *Context) {
	ctx.Set(needsRebootKey, true)
}

func needsReboot(ctx *Context) bool {
	v, ok := ctx.Get(needsRebootKey)
	return ok && v
}
