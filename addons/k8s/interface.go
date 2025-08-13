package k8s

import (
	"fmt"
	"net/http"

	appsV1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	batchV1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	coreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
)

// Interface is a subset of kubernetes.Interface that reduces the amount of k8s
// packages that get pulled in and avoid binary bloat.
type Interface interface {
	AppsV1() appsV1.AppsV1Interface
	CoreV1() coreV1.CoreV1Interface
	BatchV1() batchV1.BatchV1Interface
}

type clientset struct {
	apps  appsV1.AppsV1Interface
	core  coreV1.CoreV1Interface
	batch batchV1.BatchV1Interface
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
	cs.apps, err = appsV1.NewForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}
	cs.core, err = coreV1.NewForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}
	cs.batch, err = batchV1.NewForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}
	return &cs, nil
}
