package main

import (
	"os"
	"path/filepath"

	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/addons/bootstrap/apt"
	apt_common "fastcat.org/go/gdev/addons/bootstrap/apt/common"
	"fastcat.org/go/gdev/addons/bootstrap/input"
	"fastcat.org/go/gdev/addons/bootstrap/textedit"
	"fastcat.org/go/gdev/addons/build"
	"fastcat.org/go/gdev/addons/containerd"
	"fastcat.org/go/gdev/addons/docker"
	"fastcat.org/go/gdev/addons/gcs"
	gcs_k8s "fastcat.org/go/gdev/addons/gcs/k8s"
	"fastcat.org/go/gdev/addons/gocache"
	gocache_gcs "fastcat.org/go/gdev/addons/gocache/gcs"
	gocache_http "fastcat.org/go/gdev/addons/gocache/http"
	gocache_s3 "fastcat.org/go/gdev/addons/gocache/s3"
	gocache_sftp "fastcat.org/go/gdev/addons/gocache/sftp"
	"fastcat.org/go/gdev/addons/golang"
	"fastcat.org/go/gdev/addons/k3s"
	"fastcat.org/go/gdev/addons/k8s"
	"fastcat.org/go/gdev/addons/nodejs"
	"fastcat.org/go/gdev/addons/pm"
	"fastcat.org/go/gdev/addons/postgres"
	"fastcat.org/go/gdev/addons/valkey"
	"fastcat.org/go/gdev/cmd"
	"fastcat.org/go/gdev/instance"
)

// Normally you want to build your own wrapper around gdev to register your
// custom services and commands.
func main() {
	instance.SetAppName("gdev")

	// enable all addons we can in the main build so everything gets compiled, etc.

	bootstrap.Configure(
		bootstrap.WithAptPackages("Select Go packages for install", "golang"),
		bootstrap.WithAptPackages("Select git packages for install", "git", "git-lfs", "git-crypt"),
		bootstrap.WithSteps(shellRCSteps()...),
		bootstrap.WithSteps(
			input.UserInfoStep(),
			apt.SourceInstallStep(apt_common.GitHubCLIInstaller().AsDeb822()).With(
				bootstrap.BeforeSteps(bootstrap.StepNameAptUpdate),
			),
		),
		// many things will add more options
	)
	pm.Configure()
	k8s.Configure()        // k3s will tweak it
	containerd.Configure() // k3s will tweak it
	docker.Configure()     // k3s would tweak it if we told k3s to use docker
	k3s.Configure(
		k3s.WithProvider(docker.K3SProvider()),
		k3s.WithK3SArgs(
			// allow using any unprivileged port as a node port
			"--service-node-port-range=1024-65535",
		),
	)
	postgres.Configure(
		postgres.WithService(),
	)
	valkey.Configure(
		valkey.WithService(
			valkey.WithConfig(
				// don't use too much memory in a dev demo setup
				"maxmemory 100mb",
				// evict anything to stay below the limit, on an LRU basis
				"maxmemory-policy allkeys-lru",
			),
		),
	)
	build.Configure() // strategies will be registered by other addons
	nodejs.Configure()
	golang.Configure()
	gocache_sftp.Configure()
	gocache_gcs.Configure()
	gocache_http.Configure()
	gocache_s3.Configure(
		gocache_s3.WithRegion("us-east-1"),
	)
	gocache.Configure(
		// NOTE: you will not have access to these buckets, it is just here as an
		// example and for author testing
		gocache.WithDefaultRemotes(
			"gs://gdev-go-build-cache/v1",
			"s3://gdev-go-build-cache/v1",
		),
	)
	gcs.Configure(gcs_k8s.WithK8SService())

	cmd.Main()
}

func shellRCSteps() []*bootstrap.Step {
	var ret []*bootstrap.Step
	ret = append(ret, bootstrap.NewStep(
		"Set GOPRIVATE in ~/.bashrc",
		func(ctx *bootstrap.Context) error {
			e := textedit.AppendLine(
				`export GOPRIVATE="${GOPRIVATE:+${GOPRIVATE},}fastcat.org/go"`,
			)
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			return textedit.EditFile(filepath.Join(home, ".bashrc"), e)
		},
	))
	return ret
}
