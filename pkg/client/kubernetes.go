package client

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type K8sClient struct {
	Client    client.Client
	Clientset *kubernetes.Clientset
	Config    *rest.Config
}

func NewK8sClient() (*K8sClient, error) {
	var config *rest.Config
	var err error

	// Try in-cluster config first
	config, err = rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		var kubeconfig string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
		if envvar := os.Getenv("KUBECONFIG"); envvar != "" {
			kubeconfig = envvar
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create kubernetes config: %v", err)
		}
	}

	// Create controller-runtime client
	runtimeClient, err := client.New(config, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create controller-runtime client: %v", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %v", err)
	}

	return &K8sClient{
		Client:    runtimeClient,
		Clientset: clientset,
		Config:    config,
	}, nil
}

func (k *K8sClient) GetClusterInfo(ctx context.Context) (string, error) {
	version, err := k.Clientset.Discovery().ServerVersion()
	if err != nil {
		return "", fmt.Errorf("failed to get server version: %v", err)
	}
	return fmt.Sprintf("Kubernetes %s", version.String()), nil
}