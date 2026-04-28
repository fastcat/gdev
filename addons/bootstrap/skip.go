package bootstrap

type SkipHandler func(skippableItem string)

func WithSkipper(item string, handler SkipHandler) Option {
	return func(c *config) {
		if c.skipHandlers == nil {
			c.skipHandlers = make(map[string]SkipHandler)
		}
		c.skipHandlers[item] = handler
	}
}
