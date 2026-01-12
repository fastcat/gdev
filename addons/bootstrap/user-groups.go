package bootstrap

import (
	"fmt"
	"os/user"
	"slices"
)

func IsCurrentUserInGroup(groupName string) (inGroup bool, userName string, err error) {
	u, err := user.Current()
	if err != nil {
		return false, "", err
	}
	dg, err := user.LookupGroup(groupName)
	if err != nil {
		return false, u.Username, err
	}
	ug, err := u.GroupIds()
	if err != nil {
		return false, u.Username, err
	}
	if slices.Contains(ug, dg.Gid) {
		return true, u.Username, nil
	}
	return false, u.Username, nil
}

func EnsureCurrentUserInGroup(ctx *Context, groupName string) error {
	inGroup, userName, err := IsCurrentUserInGroup(groupName)
	if err != nil {
		return err
	}
	if inGroup {
		fmt.Printf("User %s is already in group %s\n", userName, groupName)
		return nil
	}

	fmt.Printf("Adding user %s to group %s\n", userName, groupName)
	SetNeedsReboot(ctx)

	return addUserToGroup(ctx, userName, groupName)
}
