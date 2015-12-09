package azure

import (
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/management"
	"github.com/Azure/azure-sdk-for-go/management/virtualnetwork"
	"github.com/hashicorp/terraform/helper/schema"
)

// sourceAzureLocalNetworkConnectionCreate issues all the neccessary API calls
// to create a local network connection on Azure.
func resourceAsmLocalNetworkConnectionCreate(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*AzureClient)
	mgmtClient := azureClient.asmClient.mgmtClient
	vnetClient := azureClient.asmClient.vnetClient

	log.Println("[INFO] Fetching current network configuration from Azure.")
	azureClient.asmClient.vnetMutex.Lock()
	defer azureClient.asmClient.vnetMutex.Unlock()
	netConf, err := vnetClient.GetVirtualNetworkConfiguration()
	if err != nil {
		if management.IsResourceNotFoundError(err) {
			// if no network config exists yet; create a new one now:
			netConf = virtualnetwork.NetworkConfiguration{}
		} else {
			return fmt.Errorf("Failed to get the current network configuration from Azure: %s", err)
		}
	}

	// get provided configuration:
	name := d.Get("name").(string)
	vpnGateway := d.Get("vpn_gateway_address").(string)
	var prefixes []string
	for _, prefix := range d.Get("address_space_prefixes").([]interface{}) {
		prefixes = append(prefixes, prefix.(string))
	}

	// add configuration to network config:
	netConf.Configuration.LocalNetworkSites = append(netConf.Configuration.LocalNetworkSites,
		virtualnetwork.LocalNetworkSite{
			Name:              name,
			VPNGatewayAddress: vpnGateway,
			AddressSpace: virtualnetwork.AddressSpace{
				AddressPrefix: prefixes,
			},
		})

	// send the configuration back to Azure:
	log.Println("[INFO] Sending updated network configuration back to Azure.")
	reqID, err := vnetClient.SetVirtualNetworkConfiguration(netConf)
	if err != nil {
		return fmt.Errorf("Failed setting updated network configuration: %s", err)
	}
	err = mgmtClient.WaitForOperation(reqID, nil)
	if err != nil {
		return fmt.Errorf("Failed updating the network configuration: %s", err)
	}

	d.SetId(name)
	return nil
}

// resourceAsmLocalNetworkConnectionRead does all the necessary API calls to
// read the state of our local natwork from Azure.
func resourceAsmLocalNetworkConnectionRead(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*AzureClient)
	vnetClient := azureClient.asmClient.vnetClient

	log.Println("[INFO] Fetching current network configuration from Azure.")
	netConf, err := vnetClient.GetVirtualNetworkConfiguration()
	if err != nil {
		return fmt.Errorf("Failed to get the current network configuration from Azure: %s", err)
	}

	var found bool
	name := d.Get("name").(string)

	// browsing for our network config:
	for _, lnet := range netConf.Configuration.LocalNetworkSites {
		if lnet.Name == name {
			found = true
			d.Set("vpn_gateway_address", lnet.VPNGatewayAddress)
			d.Set("address_space_prefixes", lnet.AddressSpace.AddressPrefix)
			break
		}
	}

	// remove the resource from the state of it has been deleted in the meantime:
	if !found {
		log.Println(fmt.Printf("[INFO] Azure local network '%s' has been deleted remotely. Removimg from Terraform.", name))
		d.SetId("")
	}

	return nil
}

// resourceAsmLocalNetworkConnectionUpdate does all the necessary API calls
// update the settings of our Local Network on Azure.
func resourceAsmLocalNetworkConnectionUpdate(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*AzureClient)
	mgmtClient := azureClient.asmClient.mgmtClient
	vnetClient := azureClient.asmClient.vnetClient

	log.Println("[INFO] Fetching current network configuration from Azure.")
	azureClient.asmClient.vnetMutex.Lock()
	defer azureClient.asmClient.vnetMutex.Unlock()
	netConf, err := vnetClient.GetVirtualNetworkConfiguration()
	if err != nil {
		return fmt.Errorf("Failed to get the current network configuration from Azure: %s", err)
	}

	name := d.Get("name").(string)
	cvpn := d.HasChange("vpn_gateway_address")
	cprefixes := d.HasChange("address_space_prefixes")

	var found bool
	for i, lnet := range netConf.Configuration.LocalNetworkSites {
		if lnet.Name == name {
			found = true
			if cvpn {
				netConf.Configuration.LocalNetworkSites[i].VPNGatewayAddress = d.Get("vpn_gateway_address").(string)
			}
			if cprefixes {
				var prefixes []string
				for _, prefix := range d.Get("address_space_prefixes").([]interface{}) {
					prefixes = append(prefixes, prefix.(string))
				}
				netConf.Configuration.LocalNetworkSites[i].AddressSpace.AddressPrefix = prefixes
			}
			break
		}
	}

	// remove the resource from the state of it has been deleted in the meantime:
	if !found {
		log.Println(fmt.Printf("[INFO] Azure local network '%s' has been deleted remotely. Removimg from Terraform.", name))
		d.SetId("")
	} else if cvpn || cprefixes {
		// else, send the configuration back to Azure:
		log.Println("[INFO] Sending updated network configuration back to Azure.")
		reqID, err := vnetClient.SetVirtualNetworkConfiguration(netConf)
		if err != nil {
			return fmt.Errorf("Failed setting updated network configuration: %s", err)
		}
		err = mgmtClient.WaitForOperation(reqID, nil)
		if err != nil {
			return fmt.Errorf("Failed updating the network configuration: %s", err)
		}
	}

	return nil
}

// resourceAsmLocalNetworkConnectionExists does all the necessary API calls
// to check if the local network already exists on Azure.
func resourceAsmLocalNetworkConnectionExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	vnetClient := meta.(*AzureClient).asmClient.vnetClient

	log.Println("[INFO] Fetching current network configuration from Azure.")
	netConf, err := vnetClient.GetVirtualNetworkConfiguration()
	if err != nil {
		return false, fmt.Errorf("Failed to get the current network configuration from Azure: %s", err)
	}

	name := d.Get("name")

	for _, lnet := range netConf.Configuration.LocalNetworkSites {
		if lnet.Name == name {
			return true, nil
		}
	}

	return false, nil
}

// resourceAsmLocalNetworkConnectionDelete does all the necessary API calls
// to delete a local network off Azure.
func resourceAsmLocalNetworkConnectionDelete(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*AzureClient)
	mgmtClient := azureClient.asmClient.mgmtClient
	vnetClient := azureClient.asmClient.vnetClient

	log.Println("[INFO] Fetching current network configuration from Azure.")
	azureClient.asmClient.vnetMutex.Lock()
	defer azureClient.asmClient.vnetMutex.Unlock()
	netConf, err := vnetClient.GetVirtualNetworkConfiguration()
	if err != nil {
		return fmt.Errorf("Failed to get the current network configuration from Azure: %s", err)
	}

	name := d.Get("name").(string)

	// search for our local network and remove it if found:
	for i, lnet := range netConf.Configuration.LocalNetworkSites {
		if lnet.Name == name {
			netConf.Configuration.LocalNetworkSites = append(
				netConf.Configuration.LocalNetworkSites[:i],
				netConf.Configuration.LocalNetworkSites[i+1:]...,
			)
			break
		}
	}

	// send the configuration back to Azure:
	log.Println("[INFO] Sending updated network configuration back to Azure.")
	reqID, err := vnetClient.SetVirtualNetworkConfiguration(netConf)
	if err != nil {
		return fmt.Errorf("Failed setting updated network configuration: %s", err)
	}
	err = mgmtClient.WaitForOperation(reqID, nil)
	if err != nil {
		return fmt.Errorf("Failed updating the network configuration: %s", err)
	}

	d.SetId("")
	return nil
}
