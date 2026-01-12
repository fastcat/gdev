package bootstrap

import (
	"fmt"

	"fastcat.org/go/gdev/lib/shx"
)

func addUserToGroup(ctx *Context, userName, groupName string) error {
	res, err := shx.Run(
		ctx,
		[]string{"usermod", "-aG", groupName, userName},
		shx.WithSudo(fmt.Sprintf("add user to %s group", groupName)),
		shx.PassStdio(),
		shx.WithCombinedError(),
	)
	if res != nil {
		defer res.Close() //nolint:errcheck
	}
	if err != nil {
		return err
	}
	return nil
}
