package k8s

import (
	"errors"
	"fmt"

	"fastcat.org/go/gdev/internal"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewClient() (kubernetes.Interface, error) {
	internal.CheckLockedDown()
	if config == nil {
		panic(errors.New("addon not configured"))
	}
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{
			CurrentContext: config.ContextName(),
		},
	)
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed loading k8s config for %s: %w", config.contextName, err)
	}
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed creating k8s client for %s: %w", config.contextName, err)
	}
	return client, nil
}
