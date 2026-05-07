package internal

type ResetHook func()

var resetHooks []ResetHook

func AddResetHook(hook ResetHook) {
	resetHooks = append(resetHooks, hook)
}

func Reset() {
	for _, h := range resetHooks {
		h()
	}
}
