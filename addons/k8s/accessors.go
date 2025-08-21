package k8s

import (
	"context"
	"time"

	apiAppsV1 "k8s.io/api/apps/v1"
	apiBatchV1 "k8s.io/api/batch/v1"
	apiCoreV1 "k8s.io/api/core/v1"
	apiDiscoveryV1 "k8s.io/api/discovery/v1"
	apiMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applyAppsV1 "k8s.io/client-go/applyconfigurations/apps/v1"
	applyBatchV1 "k8s.io/client-go/applyconfigurations/batch/v1"
	applyCoreV1 "k8s.io/client-go/applyconfigurations/core/v1"
	applyDiscoveryV1 "k8s.io/client-go/applyconfigurations/discovery/v1"
	applyMetaV1 "k8s.io/client-go/applyconfigurations/meta/v1"
	clientAppsV1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	clientBatchV1 "k8s.io/client-go/kubernetes/typed/batch/v1" // for cronjob
	clientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	clientDiscoveryV1 "k8s.io/client-go/kubernetes/typed/discovery/v1"

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
	typ       apiMetaV1.TypeMeta
	getClient func(c Interface, ns Namespace) Client
	// list wraps the native List method on Client to avoid extra generics on the
	// <Resource>List type
	list      func(ctx context.Context, c Client, opts apiMetaV1.ListOptions) ([]Resource, error)
	applyMeta func(a Apply) (
		*applyMetaV1.TypeMetaApplyConfiguration,
		*applyMetaV1.ObjectMetaApplyConfiguration,
	)
	resourceMeta func(r *Resource) (*apiMetaV1.TypeMeta, *apiMetaV1.ObjectMeta)
	podTemplate  func(a Apply) *applyCoreV1.PodSpecApplyConfiguration
	ready        func(ctx context.Context, c Interface, r *Resource) (bool, error)
}

func applyToAPITypeMeta(tm applyMetaV1.TypeMetaApplyConfiguration) apiMetaV1.TypeMeta {
	return apiMetaV1.TypeMeta{
		Kind:       *tm.Kind,
		APIVersion: *tm.APIVersion,
	}
}

var accStatefulSet = accessor[
	clientAppsV1.StatefulSetInterface,
	apiAppsV1.StatefulSet,
	*applyAppsV1.StatefulSetApplyConfiguration,
]{
	typ: applyToAPITypeMeta(applyAppsV1.StatefulSet("", "").TypeMetaApplyConfiguration),
	getClient: func(c Interface, ns Namespace) clientAppsV1.StatefulSetInterface {
		return c.AppsV1().StatefulSets(string(ns))
	},
	list: func(
		ctx context.Context,
		c clientAppsV1.StatefulSetInterface,
		opts apiMetaV1.ListOptions,
	) ([]apiAppsV1.StatefulSet, error) {
		l, err := c.List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return l.Items, nil
	},
	applyMeta: func(a *applyAppsV1.StatefulSetApplyConfiguration) (
		*applyMetaV1.TypeMetaApplyConfiguration,
		*applyMetaV1.ObjectMetaApplyConfiguration,
	) {
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
	ready: func(ctx context.Context, _ Interface, r *apiAppsV1.StatefulSet) (bool, error) {
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
	typ: applyToAPITypeMeta(applyAppsV1.Deployment("", "").TypeMetaApplyConfiguration),
	getClient: func(c Interface, ns Namespace) clientAppsV1.DeploymentInterface {
		return c.AppsV1().Deployments(string(ns))
	},
	list: func(
		ctx context.Context,
		c clientAppsV1.DeploymentInterface,
		opts apiMetaV1.ListOptions,
	) ([]apiAppsV1.Deployment, error) {
		l, err := c.List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return l.Items, nil
	},
	applyMeta: func(a *applyAppsV1.DeploymentApplyConfiguration) (
		*applyMetaV1.TypeMetaApplyConfiguration,
		*applyMetaV1.ObjectMetaApplyConfiguration,
	) {
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
	ready: func(ctx context.Context, _ Interface, r *apiAppsV1.Deployment) (bool, error) {
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
	typ: applyToAPITypeMeta(applyCoreV1.Service("", "").TypeMetaApplyConfiguration),
	getClient: func(c Interface, ns Namespace) clientCoreV1.ServiceInterface {
		return c.CoreV1().Services(string(ns))
	},
	list: func(
		ctx context.Context,
		c clientCoreV1.ServiceInterface,
		opts apiMetaV1.ListOptions,
	) ([]apiCoreV1.Service, error) {
		l, err := c.List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return l.Items, nil
	},
	applyMeta: func(a *applyCoreV1.ServiceApplyConfiguration) (
		*applyMetaV1.TypeMetaApplyConfiguration,
		*applyMetaV1.ObjectMetaApplyConfiguration,
	) {
		// this will ensure the ObjectMeta... is populated
		a.GetName()
		return &a.TypeMetaApplyConfiguration, a.ObjectMetaApplyConfiguration
	},
	resourceMeta: func(r *apiCoreV1.Service) (*apiMetaV1.TypeMeta, *apiMetaV1.ObjectMeta) {
		return &r.TypeMeta, &r.ObjectMeta
	},
	ready: func(ctx context.Context, c Interface, svc *apiCoreV1.Service) (bool, error) {
		// service is ready if it has a backend to talk to
		epc := accEPSlice.getClient(c, Namespace(svc.Namespace))
		lo := listOpts(ctx)
		lo.LabelSelector = apiMetaV1.FormatLabelSelector(&apiMetaV1.LabelSelector{
			MatchLabels: map[string]string{
				`kubernetes.io/service-name`: svc.Name,
			},
		})
		eps, err := accEPSlice.list(ctx, epc, lo)
		if err != nil {
			return false, err
		}
		for _, ep := range eps {
			if ready, err := accEPSlice.ready(ctx, c, &ep); err != nil {
				return false, err
			} else if ready {
				return true, nil
			}
		}
		return false, nil
	},
}

var accEPSlice = accessor[
	clientDiscoveryV1.EndpointSliceInterface,
	apiDiscoveryV1.EndpointSlice,
	*applyDiscoveryV1.EndpointSliceApplyConfiguration,
]{
	typ: applyToAPITypeMeta(applyDiscoveryV1.EndpointSlice("", "").TypeMetaApplyConfiguration),
	getClient: func(c Interface, ns Namespace) clientDiscoveryV1.EndpointSliceInterface {
		return c.DiscoveryV1().EndpointSlices(string(ns))
	},
	list: func(
		ctx context.Context,
		c clientDiscoveryV1.EndpointSliceInterface,
		opts apiMetaV1.ListOptions,
	) ([]apiDiscoveryV1.EndpointSlice, error) {
		l, err := c.List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return l.Items, nil
	},
	applyMeta: func(a *applyDiscoveryV1.EndpointSliceApplyConfiguration) (
		*applyMetaV1.TypeMetaApplyConfiguration,
		*applyMetaV1.ObjectMetaApplyConfiguration,
	) {
		// this will ensure the ObjectMeta... is populated
		a.GetName()
		return &a.TypeMetaApplyConfiguration, a.ObjectMetaApplyConfiguration
	},
	resourceMeta: func(r *apiDiscoveryV1.EndpointSlice) (*apiMetaV1.TypeMeta, *apiMetaV1.ObjectMeta) {
		return &r.TypeMeta, &r.ObjectMeta
	},
	ready: func(_ context.Context, _ Interface, eps *apiDiscoveryV1.EndpointSlice) (bool, error) {
		// endpoint slice is ready if it has at least one healthy endpoint, and if
		// it's a bit old, as nodeport stuff seems to require a bit of extra time
		// before it is actually ready.
		if mtime, err := time.Parse(
			time.RFC3339Nano,
			eps.Annotations[`endpoints.kubernetes.io/last-change-trigger-time`],
		); err != nil {
			return false, err
		} else if time.Since(mtime) < 1500*time.Millisecond {
			// Sadly this is only reported at second-level precision, so we can't
			// insert a sub-second delay here. We really want a half-second delay, but
			// we don't know when in the listed second the actual event happened, so
			// we have to pessimistically assume it was at .999, but is reporting at
			// .000, so we need to wait for 1.500.
			return false, nil
		}

		for _, ep := range eps.Endpoints {
			if len(ep.Addresses) > 0 &&
				internal.ValueOrZero(ep.Conditions.Ready) &&
				internal.ValueOrZero(ep.Conditions.Serving) &&
				!internal.ValueOrZero(ep.Conditions.Terminating) {
				return true, nil
			}
		}
		return false, nil
	},
}

var accConfigMap = accessor[
	clientCoreV1.ConfigMapInterface,
	apiCoreV1.ConfigMap,
	*applyCoreV1.ConfigMapApplyConfiguration,
]{
	typ: applyToAPITypeMeta(applyCoreV1.ConfigMap("", "").TypeMetaApplyConfiguration),
	getClient: func(c Interface, ns Namespace) clientCoreV1.ConfigMapInterface {
		return c.CoreV1().ConfigMaps(string(ns))
	},
	list: func(
		ctx context.Context,
		c clientCoreV1.ConfigMapInterface,
		opts apiMetaV1.ListOptions,
	) ([]apiCoreV1.ConfigMap, error) {
		l, err := c.List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return l.Items, nil
	},
	applyMeta: func(a *applyCoreV1.ConfigMapApplyConfiguration) (
		*applyMetaV1.TypeMetaApplyConfiguration,
		*applyMetaV1.ObjectMetaApplyConfiguration,
	) {
		// this will ensure the ObjectMeta... is populated
		a.GetName()
		return &a.TypeMetaApplyConfiguration, a.ObjectMetaApplyConfiguration
	},
	resourceMeta: func(r *apiCoreV1.ConfigMap) (*apiMetaV1.TypeMeta, *apiMetaV1.ObjectMeta) {
		return &r.TypeMeta, &r.ObjectMeta
	},
	ready: func(context.Context, Interface, *apiCoreV1.ConfigMap) (bool, error) {
		// config maps have no readiness gates
		return true, nil
	},
}

var accPVC = accessor[
	clientCoreV1.PersistentVolumeClaimInterface,
	apiCoreV1.PersistentVolumeClaim,
	*applyCoreV1.PersistentVolumeClaimApplyConfiguration,
]{
	typ: applyToAPITypeMeta(applyCoreV1.PersistentVolumeClaim("", "").TypeMetaApplyConfiguration),
	getClient: func(c Interface, ns Namespace) clientCoreV1.PersistentVolumeClaimInterface {
		return c.CoreV1().PersistentVolumeClaims(string(ns))
	},
	list: func(
		ctx context.Context,
		c clientCoreV1.PersistentVolumeClaimInterface,
		opts apiMetaV1.ListOptions,
	) ([]apiCoreV1.PersistentVolumeClaim, error) {
		l, err := c.List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return l.Items, nil
	},
	applyMeta: func(a *applyCoreV1.PersistentVolumeClaimApplyConfiguration) (
		*applyMetaV1.TypeMetaApplyConfiguration,
		*applyMetaV1.ObjectMetaApplyConfiguration,
	) {
		// this will ensure the ObjectMeta... is populated
		a.GetName()
		return &a.TypeMetaApplyConfiguration, a.ObjectMetaApplyConfiguration
	},
	resourceMeta: func(r *apiCoreV1.PersistentVolumeClaim) (*apiMetaV1.TypeMeta, *apiMetaV1.ObjectMeta) {
		return &r.TypeMeta, &r.ObjectMeta
	},
	ready: func(ctx context.Context, _ Interface, r *apiCoreV1.PersistentVolumeClaim) (bool, error) {
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
			resource.ContextValue[Interface](ctx),
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
	typ: applyToAPITypeMeta(applyCoreV1.PersistentVolume("").TypeMetaApplyConfiguration),
	getClient: func(c Interface, _ Namespace) clientCoreV1.PersistentVolumeInterface {
		return c.CoreV1().PersistentVolumes()
	},
	list: func(
		ctx context.Context,
		c clientCoreV1.PersistentVolumeInterface,
		opts apiMetaV1.ListOptions,
	) ([]apiCoreV1.PersistentVolume, error) {
		l, err := c.List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return l.Items, nil
	},
	applyMeta: func(a *applyCoreV1.PersistentVolumeApplyConfiguration) (
		*applyMetaV1.TypeMetaApplyConfiguration,
		*applyMetaV1.ObjectMetaApplyConfiguration,
	) {
		// this will ensure the ObjectMeta... is populated
		a.GetName()
		return &a.TypeMetaApplyConfiguration, a.ObjectMetaApplyConfiguration
	},
	resourceMeta: func(r *apiCoreV1.PersistentVolume) (*apiMetaV1.TypeMeta, *apiMetaV1.ObjectMeta) {
		return &r.TypeMeta, &r.ObjectMeta
	},
}

var accCronJob = accessor[
	clientBatchV1.CronJobInterface,
	apiBatchV1.CronJob,
	*applyBatchV1.CronJobApplyConfiguration,
]{
	typ: applyToAPITypeMeta(applyBatchV1.CronJob("", "").TypeMetaApplyConfiguration),
	getClient: func(c Interface, ns Namespace) clientBatchV1.CronJobInterface {
		return c.BatchV1().CronJobs(string(ns))
	},
	list: func(
		ctx context.Context,
		c clientBatchV1.CronJobInterface,
		opts apiMetaV1.ListOptions,
	) ([]apiBatchV1.CronJob, error) {
		l, err := c.List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return l.Items, nil
	},
	applyMeta: func(a *applyBatchV1.CronJobApplyConfiguration) (
		*applyMetaV1.TypeMetaApplyConfiguration,
		*applyMetaV1.ObjectMetaApplyConfiguration,
	) {
		// this will ensure the ObjectMeta... is populated
		a.GetName()
		return &a.TypeMetaApplyConfiguration, a.ObjectMetaApplyConfiguration
	},
	resourceMeta: func(r *apiBatchV1.CronJob) (*apiMetaV1.TypeMeta, *apiMetaV1.ObjectMeta) {
		return &r.TypeMeta, &r.ObjectMeta
	},
	podTemplate: func(a *applyBatchV1.CronJobApplyConfiguration) *applyCoreV1.PodSpecApplyConfiguration {
		return a.Spec.JobTemplate.Spec.Template.Spec
	},
	ready: func(ctx context.Context, _ Interface, r *apiBatchV1.CronJob) (bool, error) {
		// TODO: check if it has run, and if the last run was successful?
		return true, nil
	},
}

var accBatchJob = accessor[
	clientBatchV1.JobInterface,
	apiBatchV1.Job,
	*applyBatchV1.JobApplyConfiguration,
]{
	typ: applyToAPITypeMeta(applyBatchV1.Job("", "").TypeMetaApplyConfiguration),
	getClient: func(c Interface, ns Namespace) clientBatchV1.JobInterface {
		return c.BatchV1().Jobs(string(ns))
	},
	list: func(
		ctx context.Context,
		c clientBatchV1.JobInterface,
		opts apiMetaV1.ListOptions,
	) ([]apiBatchV1.Job, error) {
		l, err := c.List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return l.Items, nil
	},
	applyMeta: func(a *applyBatchV1.JobApplyConfiguration) (
		*applyMetaV1.TypeMetaApplyConfiguration,
		*applyMetaV1.ObjectMetaApplyConfiguration,
	) {
		// this will ensure the ObjectMeta... is populated
		a.GetName()
		return &a.TypeMetaApplyConfiguration, a.ObjectMetaApplyConfiguration
	},
	resourceMeta: func(r *apiBatchV1.Job) (*apiMetaV1.TypeMeta, *apiMetaV1.ObjectMeta) {
		return &r.TypeMeta, &r.ObjectMeta
	},
	podTemplate: func(a *applyBatchV1.JobApplyConfiguration) *applyCoreV1.PodSpecApplyConfiguration {
		return a.Spec.Template.Spec
	},
	ready: func(ctx context.Context, _ Interface, r *apiBatchV1.Job) (bool, error) {
		// batch jobs have a ready status for when they are running, but we want to
		// wait for them to finish given how this readiness gate is used.
		s := r.Status
		// TODO: is there an ObservedGeneration for jobs?
		// don't muck with trying to replicate all the counting, just check for a
		// success condition
		for _, c := range s.Conditions {
			if c.Type == apiBatchV1.JobComplete && c.Status == apiCoreV1.ConditionTrue {
				return true, nil
			} else if c.Type == apiBatchV1.JobSuccessCriteriaMet && c.Status == apiCoreV1.ConditionTrue {
				return true, nil
			}
		}
		return false, nil
	},
}

var accPod = accessor[
	clientCoreV1.PodInterface,
	apiCoreV1.Pod,
	*applyCoreV1.PodApplyConfiguration,
]{
	typ: applyToAPITypeMeta(applyCoreV1.Pod("", "").TypeMetaApplyConfiguration),
	getClient: func(c Interface, ns Namespace) clientCoreV1.PodInterface {
		return c.CoreV1().Pods(string(ns))
	},
	list: func(
		ctx context.Context,
		c clientCoreV1.PodInterface,
		opts apiMetaV1.ListOptions,
	) ([]apiCoreV1.Pod, error) {
		l, err := c.List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return l.Items, nil
	},
	applyMeta: func(a *applyCoreV1.PodApplyConfiguration) (
		*applyMetaV1.TypeMetaApplyConfiguration,
		*applyMetaV1.ObjectMetaApplyConfiguration,
	) {
		// this will ensure the ObjectMeta... is populated
		a.GetName()
		return &a.TypeMetaApplyConfiguration, a.ObjectMetaApplyConfiguration
	},
	resourceMeta: func(r *apiCoreV1.Pod) (*apiMetaV1.TypeMeta, *apiMetaV1.ObjectMeta) {
		return &r.TypeMeta, &r.ObjectMeta
	},
	ready: func(ctx context.Context, _ Interface, r *apiCoreV1.Pod) (bool, error) {
		// use the pod conditions to determine readiness
		for _, c := range r.Status.Conditions {
			if c.Type == apiCoreV1.PodReady {
				return c.Status == apiCoreV1.ConditionTrue, nil
			}
		}
		// TODO: check all the container statuses too?
		return false, nil
	},
}

var accSecret = accessor[
	clientCoreV1.SecretInterface,
	apiCoreV1.Secret,
	*applyCoreV1.SecretApplyConfiguration,
]{
	typ: applyToAPITypeMeta(applyCoreV1.Secret("", "").TypeMetaApplyConfiguration),
	getClient: func(c Interface, ns Namespace) clientCoreV1.SecretInterface {
		return c.CoreV1().Secrets(string(ns))
	},
	list: func(
		ctx context.Context,
		c clientCoreV1.SecretInterface,
		opts apiMetaV1.ListOptions,
	) ([]apiCoreV1.Secret, error) {
		l, err := c.List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return l.Items, nil
	},
	applyMeta: func(a *applyCoreV1.SecretApplyConfiguration) (
		*applyMetaV1.TypeMetaApplyConfiguration,
		*applyMetaV1.ObjectMetaApplyConfiguration,
	) {
		// this will ensure the ObjectMeta... is populated
		a.GetName()
		return &a.TypeMetaApplyConfiguration, a.ObjectMetaApplyConfiguration
	},
	resourceMeta: func(r *apiCoreV1.Secret) (*apiMetaV1.TypeMeta, *apiMetaV1.ObjectMeta) {
		return &r.TypeMeta, &r.ObjectMeta
	},
	ready: func(context.Context, Interface, *apiCoreV1.Secret) (bool, error) {
		// secrets have no readiness gates
		return true, nil
	},
}

var accNode = accessor[
	clientCoreV1.NodeInterface,
	apiCoreV1.Node,
	*applyCoreV1.NodeApplyConfiguration,
]{
	typ: applyToAPITypeMeta(applyCoreV1.Node("").TypeMetaApplyConfiguration),
	getClient: func(c Interface, _ Namespace) clientCoreV1.NodeInterface {
		return c.CoreV1().Nodes()
	},
	list: func(
		ctx context.Context,
		c clientCoreV1.NodeInterface,
		opts apiMetaV1.ListOptions,
	) ([]apiCoreV1.Node, error) {
		l, err := c.List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return l.Items, nil
	},
	applyMeta: func(a *applyCoreV1.NodeApplyConfiguration) (
		*applyMetaV1.TypeMetaApplyConfiguration,
		*applyMetaV1.ObjectMetaApplyConfiguration,
	) {
		// this will ensure the ObjectMeta... is populated
		a.GetName()
		return &a.TypeMetaApplyConfiguration, a.ObjectMetaApplyConfiguration
	},
	resourceMeta: func(r *apiCoreV1.Node) (*apiMetaV1.TypeMeta, *apiMetaV1.ObjectMeta) {
		return &r.TypeMeta, &r.ObjectMeta
	},
	ready: func(ctx context.Context, _ Interface, r *apiCoreV1.Node) (bool, error) {
		// use the node conditions to determine readiness
		for _, c := range r.Status.Conditions {
			if c.Type == apiCoreV1.NodeReady {
				return c.Status == apiCoreV1.ConditionTrue, nil
			}
		}
		return false, nil
	},
}

func getOpts(context.Context) apiMetaV1.GetOptions {
	return apiMetaV1.GetOptions{
		// nothing here for now
	}
}

func listOpts(context.Context) apiMetaV1.ListOptions {
	return apiMetaV1.ListOptions{
		// nothing here for now
	}
}

func applyOpts(context.Context) apiMetaV1.ApplyOptions {
	return apiMetaV1.ApplyOptions{
		Force:        true,
		FieldManager: instance.AppName(),
		// TODO: dry run
	}
}

func deleteOpts(context.Context) apiMetaV1.DeleteOptions {
	return apiMetaV1.DeleteOptions{
		PropagationPolicy: internal.Ptr(apiMetaV1.DeletePropagationBackground),
	}
}
