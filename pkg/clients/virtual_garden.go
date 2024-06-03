package clients

import (
	gardenerversioned "github.com/gardener/gardener/pkg/client/core/clientset/versioned"
)

var VirtualGardenClient *gardenerversioned.Clientset

// SetVirtualGardenClient sets the Gardener clientset used by the tasks.
func SetVirtualGardenClient(clientset *gardenerversioned.Clientset) {
	VirtualGardenClient = clientset
}
