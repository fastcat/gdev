package main

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	apiCore "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	applyApps "k8s.io/client-go/applyconfigurations/apps/v1"
	applyCore "k8s.io/client-go/applyconfigurations/core/v1"

	"fastcat.org/go/gdev/addons/build"
	"fastcat.org/go/gdev/addons/docker"
	"fastcat.org/go/gdev/addons/golang"
	"fastcat.org/go/gdev/addons/k3s"
	"fastcat.org/go/gdev/addons/k8s"
	"fastcat.org/go/gdev/addons/pm"
	"fastcat.org/go/gdev/addons/pm/api"
	pm_resource "fastcat.org/go/gdev/addons/pm/resource"
	"fastcat.org/go/gdev/addons/postgres"
	"fastcat.org/go/gdev/cmd"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/lib/shx"
	"fastcat.org/go/gdev/resource"
	"fastcat.org/go/gdev/service"
	"fastcat.org/go/gdev/stack"
)

const (
	svcName    = "ent-blog"
	pgNodePort = 55432
)

func main() {
	instance.SetAppName("eb-dev")

	// we're going to run a PG database in kubernetes via k3s, and the service
	// either via docker or from local source.

	pm.Configure()
	k3s.Configure(
		// use the docker provider for now so that we can manually pull images
		k3s.WithProvider(docker.K3SProvider()),
		k3s.WithK3SArgs("--service-node-port-range=1024-65535"),
	)
	build.Configure()
	golang.Configure()
	postgres.Configure(postgres.WithService(
		// avoid collisions with existing PG instances on the host
		postgres.WithNodePort(pgNodePort),
	))

	svcRepo, svcSubdir := svcDirs()
	stack.AddService(service.New(
		svcName,
		service.WithSource(svcRepo, svcSubdir, "git", "https://github.com/fastcat/gdev.git"),
		service.WithModalResourceFuncs(service.ModeDefault, svcDefaultResources),
		service.WithModalResourceFuncs(service.ModeLocal, svcLocalResources),
	))

	cmd.Main()
}

var svcDirs = sync.OnceValues(func() (string, string) {
	svcRepo := filepath.Join(shx.HomeDir(), "src", "gdev")
	svcSubdir := "examples/full-stack/ent-blog"
	return svcRepo, svcSubdir
})

func svcDefaultResources(context.Context) []resource.Resource {
	k8sDSN := fmt.Sprintf(
		"postgresql://postgres-17:%d/ent-blog?sslmode=disable",
		postgres.DefaultPort,
	)
	appContainer := applyCore.Container().
		WithName(svcName).
		WithImage("ghcr.io/fastcat/gdev/ent-blog").
		// should be PullAlways, but we don't have pull secrets setup yet
		WithImagePullPolicy(apiCore.PullIfNotPresent).
		WithEnvFrom(
			applyCore.EnvFromSource().WithSecretRef(
				applyCore.SecretEnvSource().WithName(postgres.CredentialsSecretName()),
			),
		).
		WithArgs("-dsn", k8sDSN).
		WithPorts(
			applyCore.ContainerPort().
				WithName("http").
				WithContainerPort(8080).
				WithProtocol(apiCore.ProtocolTCP),
		)
	initContainer := applyCore.Container().
		WithName("atlas-migrate").
		WithImage("ghcr.io/fastcat/gdev/ent-blog").
		// should be PullAlways, but we don't have pull secrets setup yet
		WithImagePullPolicy(apiCore.PullIfNotPresent).
		WithEnvFrom(
			applyCore.EnvFromSource().WithSecretRef(
				applyCore.SecretEnvSource().WithName(postgres.CredentialsSecretName()),
			),
		).
		WithCommand("/atlas").
		WithArgs(
			"migrate", "apply",
			// TODO: pedantically this should be ${KO_DATA_PATH}/migrations, but that requires shell expansion
			"--dir", "file:///var/run/ko/migrations",
			"--url", k8sDSN,
			// apply all of them
			"1000000000",
		)
	d := k8s.Deployment(applyApps.Deployment(svcName, "").
		WithSpec(applyApps.DeploymentSpec().
			WithTemplate(applyCore.PodTemplateSpec().
				WithSpec(applyCore.PodSpec().
					WithInitContainers(initContainer).
					WithContainers(appContainer),
				),
			),
		),
	)
	// expose on a nodeport
	s := k8s.Service(applyCore.Service(svcName, "").
		WithSpec(applyCore.ServiceSpec().
			WithType(apiCore.ServiceTypeNodePort).
			WithSelector(map[string]string{
				k8s.AppLabel(): svcName,
			}).
			WithPorts(
				applyCore.ServicePort().
					WithName("http").
					WithPort(8080).
					WithTargetPort(intstr.FromString("http")).
					WithNodePort(8080),
			),
		),
	)

	return []resource.Resource{d, s}
}

func svcLocalResources(context.Context) []resource.Resource {
	svcRepo, svcSubdir := svcDirs()
	s := pm_resource.PMStatic(api.Child{
		Name: svcName,
		Main: api.Exec{
			Cmd: "go",
			Cwd: filepath.Join(svcRepo, svcSubdir),
			Env: postgres.Credentials(),
			Args: []string{
				"run",
				".",
				"-dsn", fmt.Sprintf("postgresql://localhost:%d/ent-blog?sslmode=disable", pgNodePort),
			},
		},
	})

	return []resource.Resource{s}
}
