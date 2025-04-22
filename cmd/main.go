package cmd

// AppName is what the app will call itself. When customizing, overwrite it
// before calling Main().
var AppName = "gdev"

func Main() error {
	Root.Use = AppName
	return Root.Execute()
}
