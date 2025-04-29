package k8s

import (
	"context"

	apiAppsV1 "k8s.io/api/apps/v1"
	apiCoreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applyAppsV1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applyCoreV1 "k8s.io/client-go/applyconfigurations/core/v1"
	applyMetaV1 "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientAppsV1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	clientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type client[Resource any, Apply apply[Apply]] interface {
	Apply(context.Context, Apply, metaV1.ApplyOptions) (*Resource, error)
	Delete(context.Context, string, metaV1.DeleteOptions) error
	Get(context.Context, string, metaV1.GetOptions) (*Resource, error)

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
	list         func(ctx context.Context, c Client, opts metaV1.ListOptions) ([]Resource, error)
	applyMeta    func(a Apply) (*applyMetaV1.TypeMetaApplyConfiguration, *applyMetaV1.ObjectMetaApplyConfiguration)
	resourceMeta func(r *Resource) (*metaV1.TypeMeta, *metaV1.ObjectMeta)
	podTemplate  func(a Apply) *applyCoreV1.PodSpecApplyConfiguration
}

var accStatefulSet = accessor[
	clientAppsV1.StatefulSetInterface,
	apiAppsV1.StatefulSet,
	*applyAppsV1.StatefulSetApplyConfiguration,
]{
	getClient: func(c kubernetes.Interface, ns Namespace) clientAppsV1.StatefulSetInterface {
		return c.AppsV1().StatefulSets(string(ns))
	},
	list: func(ctx context.Context, c clientAppsV1.StatefulSetInterface, opts metaV1.ListOptions) ([]apiAppsV1.StatefulSet, error) {
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
	resourceMeta: func(r *apiAppsV1.StatefulSet) (*metaV1.TypeMeta, *metaV1.ObjectMeta) {
		return &r.TypeMeta, &r.ObjectMeta
	},
	podTemplate: func(a *applyAppsV1.StatefulSetApplyConfiguration) *applyCoreV1.PodSpecApplyConfiguration {
		return a.Spec.Template.Spec
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
	list: func(ctx context.Context, c clientAppsV1.DeploymentInterface, opts metaV1.ListOptions) ([]apiAppsV1.Deployment, error) {
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
	resourceMeta: func(r *apiAppsV1.Deployment) (*metaV1.TypeMeta, *metaV1.ObjectMeta) {
		return &r.TypeMeta, &r.ObjectMeta
	},
	podTemplate: func(a *applyAppsV1.DeploymentApplyConfiguration) *applyCoreV1.PodSpecApplyConfiguration {
		return a.Spec.Template.Spec
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
	list: func(ctx context.Context, c clientCoreV1.ServiceInterface, opts metaV1.ListOptions) ([]apiCoreV1.Service, error) {
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
	resourceMeta: func(r *apiCoreV1.Service) (*metaV1.TypeMeta, *metaV1.ObjectMeta) {
		return &r.TypeMeta, &r.ObjectMeta
	},
}
