package ddcloud

import (
	"fmt"
	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
)

// updateServerConfiguration reconfigures a server, changing the allocated RAM and / or CPU count.
func updateServerConfiguration(apiClient *compute.Client, server *compute.Server, memoryGB *int, cpuCount *int) error {
	memoryDescription := "no change"
	if memoryGB != nil {
		memoryDescription = fmt.Sprintf("will change to %dGB", *memoryGB)
	}

	cpuCountDescription := "no change"
	if memoryGB != nil {
		memoryDescription = fmt.Sprintf("will change to %d", *cpuCount)
	}

	log.Printf("Update configuration for server '%s' (memory: %s, CPU: %s)...", server.ID, memoryDescription, cpuCountDescription)

	err := apiClient.ReconfigureServer(server.ID, memoryGB, cpuCount)
	if err != nil {
		return err
	}

	_, err = apiClient.WaitForChange(compute.ResourceTypeServer, server.ID, "Reconfigure server", resourceUpdateTimeoutServer)

	return err
}

func captureServerNetworkConfiguration(server *compute.Server, data *schema.ResourceData, isPartial bool) {
	data.Set(resourceKeyServerPrimaryVLAN, *server.Network.PrimaryAdapter.VLANID)
	if isPartial {
		data.SetPartial(resourceKeyServerPrimaryVLAN)
	}

	data.Set(resourceKeyServerPrimaryIPv4, *server.Network.PrimaryAdapter.PrivateIPv4Address)
	if isPartial {
		data.SetPartial(resourceKeyServerPrimaryIPv4)
	}

	data.Set(resourceKeyServerPrimaryIPv6, *server.Network.PrimaryAdapter.PrivateIPv6Address)
	if isPartial {
		data.SetPartial(resourceKeyServerPrimaryIPv6)
	}

	data.Set(resourceKeyServerNetworkDomainID, server.Network.NetworkDomainID)
	if isPartial {
		data.SetPartial(resourceKeyServerNetworkDomainID)
	}
}

// updateServerIPAddress notifies the compute infrastructure that a server's IP address has changed.
func updateServerIPAddresses(apiClient *compute.Client, server *compute.Server, primaryIPv4 *string, primaryIPv6 *string) error {
	log.Printf("Update primary IP address(es) for server '%s'...", server.ID)

	primaryNetworkAdapterID := *server.Network.PrimaryAdapter.ID
	err := apiClient.NotifyServerIPAddressChange(primaryNetworkAdapterID, primaryIPv4, primaryIPv6)
	if err != nil {
		return err
	}

	compositeNetworkAdapterID := fmt.Sprintf("%s/%s", server.ID, primaryNetworkAdapterID)
	_, err = apiClient.WaitForChange(compute.ResourceTypeNetworkAdapter, compositeNetworkAdapterID, "Update adapter IP address", resourceUpdateTimeoutServer)

	return err
}
