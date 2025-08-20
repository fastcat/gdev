package main

import (
	"fmt"
	"path/filepath"

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
	"fastcat.org/go/gdev/addons/pm/resource"
	"fastcat.org/go/gdev/addons/postgres"
	"fastcat.org/go/gdev/cmd"
	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/lib/shx"
	"fastcat.org/go/gdev/service"
	"fastcat.org/go/gdev/stack"
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
	const pgNodePort = 55432
	postgres.Configure(postgres.WithService(
		// avoid collisions with existing PG instances on the host
		postgres.WithNodePort(pgNodePort),
	))

	svcRepo := filepath.Join(shx.HomeDir(), "src", "gdev")
	svcSubdir := "examples/full-stack/ent-blog"

	const svcName = "ent-blog"
	stack.AddService(service.New(
		svcName,
		service.WithSource(svcRepo, svcSubdir, "git", "https://github.com/fastcat/gdev.git"),
		service.WithModalResources(service.ModeDefault,
			k8s.Deployment(
				applyApps.Deployment(svcName, "").
					WithSpec(
						applyApps.DeploymentSpec().
							WithTemplate(
								applyCore.PodTemplateSpec().
									WithSpec(
										applyCore.PodSpec().
											WithContainers(
												applyCore.Container().
													WithName(svcName).
													WithImage("ghcr.io/fastcat/gdev/ent-blog").
													// should be PullAlways, but we don't have pull secrets setup yet
													WithImagePullPolicy(apiCore.PullIfNotPresent).
													WithEnv(
														// TODO: pg service should make this available in a secret
														applyCore.EnvVar().WithName("PGUSER").WithValue("postgres"),
														applyCore.EnvVar().WithName("PGPASSWORD").WithValue(instance.AppName()),
													).
													WithArgs(
														"-dsn",
														fmt.Sprintf(
															"postgresql://postgres-17:%d/ent-blog?sslmode=disable",
															postgres.DefaultPort,
														),
													).
													WithPorts(
														applyCore.ContainerPort().
															WithName("http").
															WithContainerPort(8080).
															WithProtocol(apiCore.ProtocolTCP),
													),
											),
									),
							),
					),
			),
			// expose on a nodeport
			k8s.Service(
				applyCore.Service(svcName, "").
					WithSpec(
						applyCore.ServiceSpec().
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
			),
		),
		service.WithModalResources(service.ModeLocal,
			resource.PMStatic(api.Child{
				Name: svcName,
				Main: api.Exec{
					Cmd: "go",
					Cwd: filepath.Join(svcRepo, svcSubdir),
					Env: map[string]string{
						// keep pg auth info out of the command line
						"PGUSER":     "postgres",
						"PGPASSWORD": instance.AppName(), // see pg service setup
					},
					Args: []string{
						"run",
						".",
						"-dsn", fmt.Sprintf("postgresql://localhost:%d/ent-blog?sslmode=disable", pgNodePort),
					},
				},
			}),
		),
	))

	cmd.Main()
}
