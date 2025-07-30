package docker

import (
	"fmt"
	"os/user"
	"slices"
	"sync"

	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/shx"
)

var configureBootstrap = sync.OnceFunc(func() {
	bootstrap.Configure(
		bootstrap.WithAptPackages(
			"Select Docker packages",
			"docker.io",
		),
		bootstrap.WithSteps(bootstrap.NewStep(
			"Add user to docker group",
			addUserToDockerGroup,
			bootstrap.WithSim(func(*bootstrap.Context) error {
				if inGroup, un, err := userInDockerGroup(); err != nil {
					return err
				} else if inGroup {
					fmt.Printf("User %s is already in group %s\n", un, dockerGroupName)
				} else {
					fmt.Printf("Would add user %s to group %s\n", un, dockerGroupName)
				}
				return nil
			}),
			bootstrap.WithAfter(bootstrap.StepNameAptInstall),
		)),
	)
})

const dockerGroupName = "docker"

func userInDockerGroup() (inGroup bool, userName string, err error) {
	u, err := user.Current()
	if err != nil {
		return false, "", err
	}
	dg, err := user.LookupGroup(dockerGroupName)
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

func addUserToDockerGroup(ctx *bootstrap.Context) error {
	inGroup, un, err := userInDockerGroup()
	if err != nil {
		return err
	}
	if inGroup {
		fmt.Printf("User %s is already in group %s\n", un, dockerGroupName)
		return nil
	}

	fmt.Printf("Adding user %s to group %s\n", un, dockerGroupName)

	res, err := shx.Run(
		ctx,
		[]string{"usermod", "-aG", dockerGroupName, un},
		shx.WithSudo("add user to docker group"),
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
