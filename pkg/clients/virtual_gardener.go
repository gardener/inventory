package clients

import (
	"log/slog"
	"os"

	gardenerversioned "github.com/gardener/gardener/pkg/client/core/clientset/versioned"
	"k8s.io/client-go/tools/clientcmd"
)

var VirtualGardenerClient *gardenerversioned.Clientset

// TODO: Add k8s in-cluster config, to allow running the application in a k8s cluster
func init() {
	kubeconfig := os.Getenv("KUBECONFIG")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		slog.Error("Error creating config", "err", err, "kubeconfig", kubeconfig)
		os.Exit(1)
	}

	// Create a Gardener client
	VirtualGardenerClient, err = gardenerversioned.NewForConfig(config)
	if err != nil {
		slog.Error("Error creating Gardener client", "err", err, "kubeconfig", kubeconfig)
		os.Exit(1)
	}

}
