package cmd

import "fastcat.org/go/gdev/instance"

func Main() error {
	Root.Use = instance.AppName
	return Root.Execute()
}
