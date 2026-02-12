package gcs_k8s

import (
	"context"

	coreV1 "k8s.io/api/core/v1"
	k8sResource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	coreAppsV1 "k8s.io/client-go/applyconfigurations/apps/v1"
	coreApplyV1 "k8s.io/client-go/applyconfigurations/core/v1"

	"fastcat.org/go/gdev/addons/gcs/internal"
	"fastcat.org/go/gdev/addons/k8s"
	"fastcat.org/go/gdev/resource"
	"fastcat.org/go/gdev/service"
	"fastcat.org/go/gdev/stack"
)

func WithK8SService() internal.Option {
	return func(cfg *internal.Config) {
		cfg.StackHooks = append(cfg.StackHooks, setupK8SService)
	}
}

func setupK8SService(cfg *internal.Config) error {
	appSelector := k8s.AppSelector("fake-gcs-server")
	stack.AddInfrastructure(service.New(
		"fake-gcs-server",
		service.WithResourceFuncs(func(ctx context.Context) ([]resource.Resource, error) {
			pvc := k8s.PersistentVolumeClaim(coreApplyV1.PersistentVolumeClaim("gcs-storage", "").
				WithSpec(coreApplyV1.PersistentVolumeClaimSpec().
					WithAccessModes(coreV1.ReadWriteOnce).
					WithResources(coreApplyV1.VolumeResourceRequirements().
						WithRequests(coreV1.ResourceList{
							coreV1.ResourceStorage: k8sResource.MustParse("1Gi"),
						}),
					),
				),
			)
			// exposing a node port makes localhost:... work from the host, not just
			// inside the cluster, at least for k3s. For other providers, this needs
			// work.
			sr := k8s.Service(coreApplyV1.Service("gcs", "").
				WithSpec(coreApplyV1.ServiceSpec().
					WithType(coreV1.ServiceTypeNodePort).
					WithPorts(coreApplyV1.ServicePort().
						WithName("http").
						WithProtocol(coreV1.ProtocolTCP).
						WithAppProtocol("http").
						WithPort(int32(cfg.ExposedPort)).
						WithNodePort(int32(cfg.ExposedPort)).
						WithTargetPort(intstr.FromInt(cfg.ExposedPort)),
					).
					WithSelector(appSelector.MatchLabels),
				),
			)
			dr := k8s.Deployment(coreAppsV1.Deployment("fake-gcs-server", "").
				WithSpec(coreAppsV1.DeploymentSpec().
					WithReplicas(1).
					WithSelector(appSelector).
					WithTemplate(coreApplyV1.PodTemplateSpec().
						WithLabels(appSelector.MatchLabels).
						WithSpec(coreApplyV1.PodSpec().
							WithContainers(
								coreApplyV1.Container().
									WithName("fake-gcs-server").
									WithImage(cfg.FakeServerImage).
									WithArgs(cfg.Args()...).
									WithPorts(coreApplyV1.ContainerPort().
										WithName("http").
										WithProtocol(coreV1.ProtocolTCP).
										WithContainerPort(int32(cfg.ExposedPort)),
									).
									WithVolumeMounts(coreApplyV1.VolumeMount().
										WithName("gcs-storage").
										WithMountPath("/storage"),
									),
							).
							WithVolumes(
								coreApplyV1.Volume().
									WithName("gcs-storage").
									WithPersistentVolumeClaim(
										coreApplyV1.PersistentVolumeClaimVolumeSource().
											WithClaimName(pvc.K8SName()),
									),
							),
						),
					),
				),
			)
			return []resource.Resource{pvc, sr, dr}, nil
		}),
	))
	return nil
}
