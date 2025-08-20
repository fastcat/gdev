package docker

import (
	"fmt"
	"os/user"
	"slices"
	"sync"

	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/addons/bootstrap/apt"
	"fastcat.org/go/gdev/lib/shx"
)

var configureBootstrap = sync.OnceFunc(func() {
	bootstrap.Configure(
		apt.WithPackages(
			"Select common Docker packages",
			"docker.io",
			"docker-buildx",
		),
		bootstrap.WithSteps(
			apt.AddPackageIfAvailable(
				"Select docker-cli if needed",
				// this is only on Ubuntu 25.04+ and Debian 13+
				"docker-cli",
			),
			apt.AddFirstAvailable(
				"Select docker-compose",
				// Older Ubuntu has compose v2 in a separate package.
				// Older Debian doesn't have it at all.
				// Newer Ubuntu and Debian have it in the base package.
				"docker-compose-v2",
				"docker-compose",
			),
		),
		bootstrap.WithSteps(bootstrap.NewStep(
			"Add user to docker group",
			addUserToDockerGroup,
			bootstrap.SimFunc(func(*bootstrap.Context) error {
				if inGroup, un, err := userInDockerGroup(); err != nil {
					return err
				} else if inGroup {
					fmt.Printf("User %s is already in group %s\n", un, dockerGroupName)
				} else {
					fmt.Printf("Would add user %s to group %s\n", un, dockerGroupName)
				}
				return nil
			}),
			bootstrap.AfterSteps(apt.StepNameInstall),
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
	bootstrap.SetNeedsReboot(ctx)

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
