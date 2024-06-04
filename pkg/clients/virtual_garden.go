package clients

import (
	"log/slog"

	gardenerversioned "github.com/gardener/gardener/pkg/client/core/clientset/versioned"
	machineversioned "github.com/gardener/machine-controller-manager/pkg/client/clientset/versioned"
	"k8s.io/client-go/rest"
)

const VIRTUAL_GARDEN = "virtual-garden"

var GardenConfigs map[string]*rest.Config

func VirtualGardenClient() *gardenerversioned.Clientset {
	config, found := GardenConfigs[VIRTUAL_GARDEN]
	if !found {
		slog.Error("VirtualGardenClient not found", "name", "virtual-garden")
		return nil
	}
	client, err := gardenerversioned.NewForConfig(config)
	if err != nil {
		slog.Error("Failed to create VirtualGardenClient", "error", err)
		return nil
	}
	return client
}

func SetGardenConfigs(clients map[string]*rest.Config) {
	GardenConfigs = clients
}

func GardenClient(name string) *machineversioned.Clientset {
	config, found := GardenConfigs[name]
	if !found {
		slog.Error("GardenClient not found", "name", name)
		return nil
	}
	client, err := machineversioned.NewForConfig(config)
	if err != nil {
		slog.Error("Failed to create GardenClient", "error", err)
		return nil
	}
	return client

}
