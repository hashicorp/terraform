package azure

import (
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/management"
	"github.com/Azure/azure-sdk-for-go/management/virtualnetwork"
	"github.com/hashicorp/terraform/helper/schema"
)

// resourceAzureDnsServer returns the *schema.Resource associated
// to an Azure hosted service.
func resourceAzureDnsServer() *schema.Resource {
	return &schema.Resource{
		Create: resourceAzureDnsServerCreate,
		Read:   resourceAzureDnsServerRead,
		Update: resourceAzureDnsServerUpdate,
		Exists: resourceAzureDnsServerExists,
		Delete: resourceAzureDnsServerDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: parameterDescriptions["name"],
			},
			"dns_address": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: parameterDescriptions["dns_address"],
			},
		},
	}
}

// resourceAzureDnsServerCreate does all the necessary API calls
// to create a new DNS server definition on Azure.
func resourceAzureDnsServerCreate(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*Client)
	mgmtClient := azureClient.mgmtClient
	vnetClient := azureClient.vnetClient

	log.Println("[INFO] Fetching current network configuration from Azure.")
	azureClient.vnetMutex.Lock()
	defer azureClient.vnetMutex.Unlock()
	netConf, err := vnetClient.GetVirtualNetworkConfiguration()
	if err != nil {
		if management.IsResourceNotFoundError(err) {
			// if no network configuration exists yet; create one now:
			netConf = virtualnetwork.NetworkConfiguration{}
		} else {
			return fmt.Errorf("Failed to get the current network configuration from Azure: %s", err)
		}
	}

	log.Println("[DEBUG] Adding new DNS server definition to Azure.")
	name := d.Get("name").(string)
	address := d.Get("dns_address").(string)
	netConf.Configuration.DNS.DNSServers = append(
		netConf.Configuration.DNS.DNSServers,
		virtualnetwork.DNSServer{
			Name:      name,
			IPAddress: address,
		})

	// send the configuration back to Azure:
	log.Println("[INFO] Sending updated network configuration back to Azure.")
	reqID, err := vnetClient.SetVirtualNetworkConfiguration(netConf)
	if err != nil {
		return fmt.Errorf("Failed issuing update to network configuration: %s", err)
	}
	err = mgmtClient.WaitForOperation(reqID, nil)
	if err != nil {
		return fmt.Errorf("Error setting network configuration: %s", err)
	}

	d.SetId(name)
	return nil
}

// resourceAzureDnsServerRead does all the necessary API calls to read
// the state of the DNS server off Azure.
func resourceAzureDnsServerRead(d *schema.ResourceData, meta interface{}) error {
	vnetClient := meta.(*Client).vnetClient

	log.Println("[INFO] Fetching current network configuration from Azure.")
	netConf, err := vnetClient.GetVirtualNetworkConfiguration()
	if err != nil {
		return fmt.Errorf("Failed to get the current network configuration from Azure: %s", err)
	}

	var found bool
	name := d.Get("name").(string)

	// search for our DNS and update it if the IP has been changed:
	for _, dns := range netConf.Configuration.DNS.DNSServers {
		if dns.Name == name {
			found = true
			d.Set("dns_address", dns.IPAddress)
			break
		}
	}

	// remove the resource from the state if it has been deleted in the meantime:
	if !found {
		d.SetId("")
	}

	return nil
}

// resourceAzureDnsServerUpdate does all the necessary API calls
// to update the DNS definition on Azure.
func resourceAzureDnsServerUpdate(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*Client)
	mgmtClient := azureClient.mgmtClient
	vnetClient := azureClient.vnetClient

	var found bool
	name := d.Get("name").(string)

	if d.HasChange("dns_address") {
		log.Println("[DEBUG] DNS server address has changes; updating it on Azure.")
		log.Println("[INFO] Fetching current network configuration from Azure.")
		azureClient.vnetMutex.Lock()
		defer azureClient.vnetMutex.Unlock()
		netConf, err := vnetClient.GetVirtualNetworkConfiguration()
		if err != nil {
			return fmt.Errorf("Failed to get the current network configuration from Azure: %s", err)
		}

		// search for our DNS and update its address value:
		for i, dns := range netConf.Configuration.DNS.DNSServers {
			if dns.Name == name {
				found = true
				netConf.Configuration.DNS.DNSServers[i].IPAddress = d.Get("dns_address").(string)
				break
			}
		}

		// if the config has changes, send the configuration back to Azure:
		if found {
			log.Println("[INFO] Sending updated network configuration back to Azure.")
			reqID, err := vnetClient.SetVirtualNetworkConfiguration(netConf)
			if err != nil {
				return fmt.Errorf("Failed issuing update to network configuration: %s", err)
			}
			err = mgmtClient.WaitForOperation(reqID, nil)
			if err != nil {
				return fmt.Errorf("Error setting network configuration: %s", err)
			}

			return nil
		}
	}

	// remove the resource from the state if it has been deleted in the meantime:
	if !found {
		d.SetId("")
	}

	return nil
}

// resourceAzureDnsServerExists does all the necessary API calls to
// check if the DNS server definition already exists on Azure.
func resourceAzureDnsServerExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	azureClient := meta.(*Client)
	vnetClient := azureClient.vnetClient

	log.Println("[INFO] Fetching current network configuration from Azure.")
	netConf, err := vnetClient.GetVirtualNetworkConfiguration()
	if err != nil {
		return false, fmt.Errorf("Failed to get the current network configuration from Azure: %s", err)
	}

	name := d.Get("name").(string)

	// search for the DNS server's definition:
	for _, dns := range netConf.Configuration.DNS.DNSServers {
		if dns.Name == name {
			return true, nil
		}
	}

	// if we reached this point; the resource must have been deleted; and we must untrack it:
	d.SetId("")
	return false, nil
}

// resourceAzureDnsServerDelete does all the necessary API calls
// to delete the DNS server definition from Azure.
func resourceAzureDnsServerDelete(d *schema.ResourceData, meta interface{}) error {
	azureClient := meta.(*Client)
	mgmtClient := azureClient.mgmtClient
	vnetClient := azureClient.vnetClient

	log.Println("[INFO] Fetching current network configuration from Azure.")
	azureClient.vnetMutex.Lock()
	defer azureClient.vnetMutex.Unlock()
	netConf, err := vnetClient.GetVirtualNetworkConfiguration()
	if err != nil {
		return fmt.Errorf("Failed to get the current network configuration from Azure: %s", err)
	}

	name := d.Get("name").(string)

	// search for the DNS server's definition and remove it:
	var found bool
	for i, dns := range netConf.Configuration.DNS.DNSServers {
		if dns.Name == name {
			found = true
			netConf.Configuration.DNS.DNSServers = append(
				netConf.Configuration.DNS.DNSServers[:i],
				netConf.Configuration.DNS.DNSServers[i+1:]...,
			)
			break
		}
	}

	// if not found; don't bother re-sending the natwork config:
	if !found {
		return nil
	}

	// send the configuration back to Azure:
	log.Println("[INFO] Sending updated network configuration back to Azure.")
	reqID, err := vnetClient.SetVirtualNetworkConfiguration(netConf)
	if err != nil {
		return fmt.Errorf("Failed issuing update to network configuration: %s", err)
	}
	err = mgmtClient.WaitForOperation(reqID, nil)
	if err != nil {
		return fmt.Errorf("Error setting network configuration: %s", err)
	}

	return nil
}
