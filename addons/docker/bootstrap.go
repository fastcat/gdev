package docker

import (
	"fmt"
	"sync"

	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/addons/bootstrap/apt"
)

var configureBootstrap = sync.OnceFunc(func() {
	bootstrap.Configure(
		apt.WithPackages(
			"Select common Docker packages",
			"docker.io",
			"docker-buildx",
		),
		bootstrap.WithSteps(
			bootstrap.NewStep(
				"Select docker-ce packages for removal",
				func(ctx *bootstrap.Context) error {
					// asking for removal of packages that are neither installed nor
					// available creates an error
					toRemove := []string{
						// docker-ce packages conflict with docker.io packages
						"docker-ce",
						"docker-ce-cli",
						"docker-buildx-plugin",
						"docker-compose-plugin",
						// confusingly while docker.io is the vendor package, containerd.io is the
						// docker-ce package
						"containerd.io",
					}
					aptAvail, err := apt.AptAvailable(ctx)
					if err != nil {
						return err
					}
					for _, pkg := range toRemove {
						if _, ok := aptAvail[pkg]; ok {
							// "-" suffix tells apt to remove the package instead of installing it
							apt.AddPackages(ctx, pkg+"-")
						}
					}
					return nil
				},
				bootstrap.BeforeSteps(apt.StepNameInstall),
			),
			apt.AddPackagesStep(
				"Select docker credential helper(s)",
				"golang-docker-credential-helpers",
			).With(
				bootstrap.BeforeSteps(apt.StepNameInstall),
				bootstrap.SkipInContainer(),
			),
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
			func(ctx *bootstrap.Context) error {
				return bootstrap.EnsureCurrentUserInGroup(ctx, dockerGroupName)
			},
			bootstrap.SimFunc(func(*bootstrap.Context) error {
				if inGroup, un, err := bootstrap.IsCurrentUserInGroup(dockerGroupName); err != nil {
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
		// TODO: configure secretsstore as docker credential helper
	)
})

const dockerGroupName = "docker"
