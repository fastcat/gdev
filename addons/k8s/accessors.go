package k8s

import (
	"context"

	apiAppsV1 "k8s.io/api/apps/v1"
	apiCoreV1 "k8s.io/api/core/v1"
	apiMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applyAppsV1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applyCoreV1 "k8s.io/client-go/applyconfigurations/core/v1"
	applyMetaV1 "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientAppsV1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	clientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"fastcat.org/go/gdev/instance"
	"fastcat.org/go/gdev/internal"
	"fastcat.org/go/gdev/resource"
)

type client[Resource any, Apply apply[Apply]] interface {
	Apply(context.Context, Apply, apiMetaV1.ApplyOptions) (*Resource, error)
	Delete(context.Context, string, apiMetaV1.DeleteOptions) error
	Get(context.Context, string, apiMetaV1.GetOptions) (*Resource, error)

	// could also add Create, Update, DeleteCollection, Watch, Patch as needed
}

// apply represents an "Apply" type for a resource, generally a pointer to the
// struct type.
type apply[T any] interface {
	WithName(string) T
	GetName() *string
	WithNamespace(string) T
	WithKind(string) T
	WithAPIVersion(string) T
}

type accessor[
	Client client[Resource, Apply],
	Resource any,
	Apply apply[Apply],
] struct {
	getClient func(c kubernetes.Interface, ns Namespace) Client
	// list wraps the native List method on Client to avoid extra generics on the
	// <Resource>List type
	list         func(ctx context.Context, c Client, opts apiMetaV1.ListOptions) ([]Resource, error)
	applyMeta    func(a Apply) (*applyMetaV1.TypeMetaApplyConfiguration, *applyMetaV1.ObjectMetaApplyConfiguration)
	resourceMeta func(r *Resource) (*apiMetaV1.TypeMeta, *apiMetaV1.ObjectMeta)
	podTemplate  func(a Apply) *applyCoreV1.PodSpecApplyConfiguration
	ready        func(ctx *resource.Context, r *Resource) (bool, error)
}

var accStatefulSet = accessor[
	clientAppsV1.StatefulSetInterface,
	apiAppsV1.StatefulSet,
	*applyAppsV1.StatefulSetApplyConfiguration,
]{
	getClient: func(c kubernetes.Interface, ns Namespace) clientAppsV1.StatefulSetInterface {
		return c.AppsV1().StatefulSets(string(ns))
	},
	list: func(ctx context.Context, c clientAppsV1.StatefulSetInterface, opts apiMetaV1.ListOptions) ([]apiAppsV1.StatefulSet, error) {
		l, err := c.List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return l.Items, nil
	},
	applyMeta: func(a *applyAppsV1.StatefulSetApplyConfiguration) (*applyMetaV1.TypeMetaApplyConfiguration, *applyMetaV1.ObjectMetaApplyConfiguration) {
		// this will ensure the ObjectMeta... is populated
		a.GetName()
		return &a.TypeMetaApplyConfiguration, a.ObjectMetaApplyConfiguration
	},
	resourceMeta: func(r *apiAppsV1.StatefulSet) (*apiMetaV1.TypeMeta, *apiMetaV1.ObjectMeta) {
		return &r.TypeMeta, &r.ObjectMeta
	},
	podTemplate: func(a *applyAppsV1.StatefulSetApplyConfiguration) *applyCoreV1.PodSpecApplyConfiguration {
		return a.Spec.Template.Spec
	},
	ready: func(ctx *resource.Context, r *apiAppsV1.StatefulSet) (bool, error) {
		s := r.Status
		// statefulset knows what up to date means
		ready := s.ObservedGeneration == r.Generation &&
			// all the old replicas are gone
			s.UpdatedReplicas == s.Replicas &&
			// assume all replicas are required
			s.AvailableReplicas == s.Replicas &&
			s.ReadyReplicas == s.Replicas &&
			s.Replicas == *r.Spec.Replicas
		return ready, nil
	},
}

var accDeployment = accessor[
	clientAppsV1.DeploymentInterface,
	apiAppsV1.Deployment,
	*applyAppsV1.DeploymentApplyConfiguration,
]{
	getClient: func(c kubernetes.Interface, ns Namespace) clientAppsV1.DeploymentInterface {
		return c.AppsV1().Deployments(string(ns))
	},
	list: func(ctx context.Context, c clientAppsV1.DeploymentInterface, opts apiMetaV1.ListOptions) ([]apiAppsV1.Deployment, error) {
		l, err := c.List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return l.Items, nil
	},
	applyMeta: func(a *applyAppsV1.DeploymentApplyConfiguration) (*applyMetaV1.TypeMetaApplyConfiguration, *applyMetaV1.ObjectMetaApplyConfiguration) {
		// this will ensure the ObjectMeta... is populated
		a.GetName()
		return &a.TypeMetaApplyConfiguration, a.ObjectMetaApplyConfiguration
	},
	resourceMeta: func(r *apiAppsV1.Deployment) (*apiMetaV1.TypeMeta, *apiMetaV1.ObjectMeta) {
		return &r.TypeMeta, &r.ObjectMeta
	},
	podTemplate: func(a *applyAppsV1.DeploymentApplyConfiguration) *applyCoreV1.PodSpecApplyConfiguration {
		return a.Spec.Template.Spec
	},
	ready: func(ctx *resource.Context, r *apiAppsV1.Deployment) (bool, error) {
		s := r.Status
		// deployment knows what up to date means
		ready := s.ObservedGeneration == r.Generation &&
			// all the old replicas are gone
			s.UpdatedReplicas == s.Replicas &&
			// at least one replica is good to go
			s.AvailableReplicas > 0 &&
			s.ReadyReplicas > 0
		return ready, nil
	},
}

var accService = accessor[
	clientCoreV1.ServiceInterface,
	apiCoreV1.Service,
	*applyCoreV1.ServiceApplyConfiguration,
]{
	getClient: func(c kubernetes.Interface, ns Namespace) clientCoreV1.ServiceInterface {
		return c.CoreV1().Services(string(ns))
	},
	list: func(ctx context.Context, c clientCoreV1.ServiceInterface, opts apiMetaV1.ListOptions) ([]apiCoreV1.Service, error) {
		l, err := c.List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return l.Items, nil
	},
	applyMeta: func(a *applyCoreV1.ServiceApplyConfiguration) (*applyMetaV1.TypeMetaApplyConfiguration, *applyMetaV1.ObjectMetaApplyConfiguration) {
		// this will ensure the ObjectMeta... is populated
		a.GetName()
		return &a.TypeMetaApplyConfiguration, a.ObjectMetaApplyConfiguration
	},
	resourceMeta: func(r *apiCoreV1.Service) (*apiMetaV1.TypeMeta, *apiMetaV1.ObjectMeta) {
		return &r.TypeMeta, &r.ObjectMeta
	},
	ready: func(*resource.Context, *apiCoreV1.Service) (bool, error) {
		// services have no readiness gates
		return true, nil
	},
}

var accPVC = accessor[
	clientCoreV1.PersistentVolumeClaimInterface,
	apiCoreV1.PersistentVolumeClaim,
	*applyCoreV1.PersistentVolumeClaimApplyConfiguration,
]{
	getClient: func(c kubernetes.Interface, ns Namespace) clientCoreV1.PersistentVolumeClaimInterface {
		return c.CoreV1().PersistentVolumeClaims(string(ns))
	},
	list: func(ctx context.Context, c clientCoreV1.PersistentVolumeClaimInterface, opts apiMetaV1.ListOptions) ([]apiCoreV1.PersistentVolumeClaim, error) {
		l, err := c.List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return l.Items, nil
	},
	applyMeta: func(a *applyCoreV1.PersistentVolumeClaimApplyConfiguration) (*applyMetaV1.TypeMetaApplyConfiguration, *applyMetaV1.ObjectMetaApplyConfiguration) {
		// this will ensure the ObjectMeta... is populated
		a.GetName()
		return &a.TypeMetaApplyConfiguration, a.ObjectMetaApplyConfiguration
	},
	resourceMeta: func(r *apiCoreV1.PersistentVolumeClaim) (*apiMetaV1.TypeMeta, *apiMetaV1.ObjectMeta) {
		return &r.TypeMeta, &r.ObjectMeta
	},
	ready: func(ctx *resource.Context, r *apiCoreV1.PersistentVolumeClaim) (bool, error) {
		if r.Status.Phase != apiCoreV1.ClaimBound {
			// not bound to a PV
			return false, nil
		}
		vn := r.Spec.VolumeName
		if vn == "" {
			// should be impossible with the bound phase?
			return false, nil
		}
		pvClient := accPV.getClient(
			resource.ContextValue[kubernetes.Interface](ctx),
			resource.ContextValue[Namespace](ctx),
		)
		pv, err := pvClient.Get(ctx, vn, getOpts(ctx))
		if err != nil {
			return false, err
		}
		status := pv.Status.Phase
		ready := status == apiCoreV1.VolumeAvailable ||
			status == apiCoreV1.VolumeBound ||
			status == apiCoreV1.VolumeReleased
		return ready, nil

	},
}

var accPV = accessor[
	clientCoreV1.PersistentVolumeInterface,
	apiCoreV1.PersistentVolume,
	*applyCoreV1.PersistentVolumeApplyConfiguration,
]{
	getClient: func(c kubernetes.Interface, _ Namespace) clientCoreV1.PersistentVolumeInterface {
		return c.CoreV1().PersistentVolumes()
	},
	list: func(ctx context.Context, c clientCoreV1.PersistentVolumeInterface, opts apiMetaV1.ListOptions) ([]apiCoreV1.PersistentVolume, error) {
		l, err := c.List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return l.Items, nil
	},
	applyMeta: func(a *applyCoreV1.PersistentVolumeApplyConfiguration) (*applyMetaV1.TypeMetaApplyConfiguration, *applyMetaV1.ObjectMetaApplyConfiguration) {
		// this will ensure the ObjectMeta... is populated
		a.GetName()
		return &a.TypeMetaApplyConfiguration, a.ObjectMetaApplyConfiguration
	},
	resourceMeta: func(r *apiCoreV1.PersistentVolume) (*apiMetaV1.TypeMeta, *apiMetaV1.ObjectMeta) {
		return &r.TypeMeta, &r.ObjectMeta
	},
}

func getOpts(*resource.Context) apiMetaV1.GetOptions {
	return apiMetaV1.GetOptions{
		// nothing here for now
	}
}

func applyOpts(*resource.Context) apiMetaV1.ApplyOptions {
	return apiMetaV1.ApplyOptions{
		Force:        true,
		FieldManager: instance.AppName(),
		// TODO: dry run
	}
}

func deleteOpts(*resource.Context) apiMetaV1.DeleteOptions {
	return apiMetaV1.DeleteOptions{
		PropagationPolicy: internal.Ptr(apiMetaV1.DeletePropagationBackground),
	}
}
