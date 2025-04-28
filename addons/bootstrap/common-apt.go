package bootstrap

// Name of the step registered by [AddAptUpdate]. Steps that modify apt sources
// should reference this as a `before` constraint.
const StepNameAptUpdate = "apt-update"

func AddAptUpdate() {
	AddStep(aptUpdate())
}

func aptUpdate() *step {
	return Step(StepNameAptUpdate, doAptUpdate)
}

func doAptUpdate(ctx *Context) error {
	return Shell(
		ctx,
		[]string{"apt", "update"},
		WithSudo("update available packages"),
	)
}

func init() {
	defaultStepFactories[StepNameAptUpdate] = aptUpdate
}
