package k8s

import (
	"context"
	"fmt"
	"net/http"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	appsV1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	batchV1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	coreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	discoveryV1 "k8s.io/client-go/kubernetes/typed/discovery/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
)

// Interface is a subset of kubernetes.Interface that reduces the amount of k8s
// packages that get pulled in and avoid binary bloat.
type Interface interface {
	AppsV1() appsV1.AppsV1Interface
	BatchV1() batchV1.BatchV1Interface
	CoreV1() coreV1.CoreV1Interface
	DiscoveryV1() discoveryV1.DiscoveryV1Interface
	Health() HealthInterface
}

// k8s doesn't provide this interface, so we build it up using the rest client
type HealthInterface interface {
	// Calls the healthz endpoint
	//
	// Deprecated: kubernetes recommends using Ready(z) or Live(z) endpoints instead.
	// See https://kubernetes.io/docs/reference/using-api/health-checks/
	Healthy(context.Context) error
	// Calls the readyz endpoint
	Ready(context.Context) error
	// Calls the livez endpoint
	Live(context.Context) error
}

type clientset struct {
	apps      appsV1.AppsV1Interface
	batch     batchV1.BatchV1Interface
	core      coreV1.CoreV1Interface
	discovery discoveryV1.DiscoveryV1Interface
	health    HealthInterface
}

// AppsV1 implements Interface.
func (c *clientset) AppsV1() appsV1.AppsV1Interface {
	return c.apps
}

// BatchV1 implements Interface.
func (c *clientset) BatchV1() batchV1.BatchV1Interface {
	return c.batch
}

// CoreV1 implements Interface.
func (c *clientset) CoreV1() coreV1.CoreV1Interface {
	return c.core
}

// DiscoveryV1 implements Interface.
func (c *clientset) DiscoveryV1() discoveryV1.DiscoveryV1Interface {
	return c.discovery
}

// Health implements Interface.
func (c *clientset) Health() HealthInterface {
	return c.health
}

func NewForConfig(config *rest.Config) (Interface, error) {
	// replicating kubernetes.NewForConfig, but only our subset
	configShallowCopy := *config

	if configShallowCopy.UserAgent == "" {
		configShallowCopy.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	// share the transport between all clients
	httpClient, err := rest.HTTPClientFor(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	return NewForConfigAndClient(&configShallowCopy, httpClient)
}

func NewForConfigAndClient(c *rest.Config, httpClient *http.Client) (*clientset, error) {
	configShallowCopy := *c
	if configShallowCopy.RateLimiter == nil && configShallowCopy.QPS > 0 {
		if configShallowCopy.Burst <= 0 {
			return nil, fmt.Errorf(
				"burst is required to be greater than 0 when RateLimiter is not set and QPS is set to greater than 0",
			)
		}
		configShallowCopy.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(
			configShallowCopy.QPS,
			configShallowCopy.Burst,
		)
	}

	var cs clientset
	var err error
	if cs.apps, err = appsV1.NewForConfigAndClient(&configShallowCopy, httpClient); err != nil {
		return nil, err
	}
	if cs.batch, err = batchV1.NewForConfigAndClient(&configShallowCopy, httpClient); err != nil {
		return nil, err
	}
	if cs.core, err = coreV1.NewForConfigAndClient(&configShallowCopy, httpClient); err != nil {
		return nil, err
	}
	if cs.discovery, err = discoveryV1.NewForConfigAndClient(&configShallowCopy, httpClient); err != nil {
		return nil, err
	}
	if cs.health, err = NewHealthClient(&configShallowCopy, httpClient); err != nil {
		return nil, err
	}

	return &cs, nil
}

type healthClient struct {
	c *rest.RESTClient
}

func NewHealthClient(config *rest.Config, httpClient *http.Client) (*healthClient, error) {
	// replicate enough of k8s' `setConfigDefaults` so things work
	config.GroupVersion = &schema.GroupVersion{Group: "", Version: ""}
	config.NegotiatedSerializer = rest.CodecFactoryForGeneratedClient(scheme.Scheme, scheme.Codecs).
		WithoutConversion()
	restClient, err := rest.RESTClientForConfigAndClient(config, httpClient)
	if err != nil {
		return nil, err
	}
	return &healthClient{restClient}, nil
}

// Healthy implements HealthInterface.
func (h *healthClient) Healthy(ctx context.Context) error {
	res := h.c.Get().AbsPath("/healthz").Do(ctx)
	return res.Error()
}

// Ready implements HealthInterface.
func (h *healthClient) Ready(ctx context.Context) error {
	res := h.c.Get().AbsPath("/readyz").Do(ctx)
	return res.Error()
}

// Live implements HealthInterface.
func (h *healthClient) Live(ctx context.Context) error {
	res := h.c.Get().AbsPath("/livez").Do(ctx)
	return res.Error()
}
