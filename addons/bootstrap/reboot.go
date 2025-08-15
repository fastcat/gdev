package bootstrap

var needsRebootKey = NewKey[bool]("need-reboot-after-bootstrap")

func SetNeedsReboot(ctx *Context) {
	Set(ctx, needsRebootKey, true)
}

func needsReboot(ctx *Context) bool {
	v, ok := Get(ctx, needsRebootKey)
	return ok && v
}
