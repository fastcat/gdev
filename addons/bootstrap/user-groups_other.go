//go:build !linux

package bootstrap

import (
	"fmt"
	"runtime"
)

func addUserToGroup(_ *Context, userName, groupName string) error {
	return fmt.Errorf("adding user %s to group %s is not supported on %s", userName, groupName, runtime.GOOS)
}
