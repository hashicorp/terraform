package ddcloud

import (
	"fmt"
	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"time"
)

const (
	resourceKeyServerName            = "name"
	resourceKeyServerDescription     = "description"
	resourceKeyServerAdminPassword   = "admin_password"
	resourceKeyServerNetworkDomainID = "networkdomain"
	resourceKeyServerMemoryGB        = "memory_gb"
	resourceKeyServerCPUCount        = "cpu_count"
	resourceKeyServerOSImageID       = "osimage_id"
	resourceKeyServerOSImageName     = "osimage_name"
	resourceKeyServerPrimaryVLAN     = "primary_adapter_vlan"
	resourceKeyServerPrimaryIPv4     = "primary_adapter_ipv4"
	resourceKeyServerPrimaryIPv6     = "primary_adapter_ipv6"
	resourceKeyServerPrimaryDNS      = "dns_primary"
	resourceKeyServerSecondaryDNS    = "dns_secondary"
	resourceKeyServerAutoStart       = "auto_start"
	resourceCreateTimeoutServer      = 30 * time.Minute
	resourceUpdateTimeoutServer      = 10 * time.Minute
	resourceDeleteTimeoutServer      = 15 * time.Minute
	serverShutdownTimeout            = 5 * time.Minute
)

func resourceServer() *schema.Resource {
	return &schema.Resource{
		Create: resourceServerCreate,
		Read:   resourceServerRead,
		Update: resourceServerUpdate,
		Delete: resourceServerDelete,

		Schema: map[string]*schema.Schema{
			resourceKeyServerName: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			resourceKeyServerDescription: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			resourceKeyServerAdminPassword: &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			resourceKeyServerMemoryGB: &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				Default:  nil,
			},
			resourceKeyServerCPUCount: &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				Default:  nil,
			},
			resourceKeyServerDisk: schemaServerDisk(),
			resourceKeyServerNetworkDomainID: &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			resourceKeyServerPrimaryVLAN: &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Computed: true,
				Default:  nil,
			},
			resourceKeyServerPrimaryIPv4: &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Computed: true,
				Default:  nil,
			},
			resourceKeyServerPrimaryIPv6: &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			resourceKeyServerPrimaryDNS: &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
			},
			resourceKeyServerSecondaryDNS: &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Default:  "",
			},
			resourceKeyServerOSImageID: &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Computed: true,
				Default:  nil,
			},
			resourceKeyServerOSImageName: &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Computed: true,
				Default:  nil,
			},
			resourceKeyServerAutoStart: &schema.Schema{
				Type:     schema.TypeBool,
				ForceNew: true,
				Optional: true,
				Default:  false,
			},
			resourceKeyServerTag: schemaServerTag(),
		},
	}
}

// Create a server resource.
func resourceServerCreate(data *schema.ResourceData, provider interface{}) error {
	name := data.Get(resourceKeyServerName).(string)
	description := data.Get(resourceKeyServerDescription).(string)
	adminPassword := data.Get(resourceKeyServerAdminPassword).(string)
	networkDomainID := data.Get(resourceKeyServerNetworkDomainID).(string)
	primaryDNS := data.Get(resourceKeyServerPrimaryDNS).(string)
	secondaryDNS := data.Get(resourceKeyServerSecondaryDNS).(string)
	autoStart := data.Get(resourceKeyServerAutoStart).(bool)

	log.Printf("Create server '%s' in network domain '%s' (description = '%s').", name, networkDomainID, description)

	apiClient := provider.(*providerState).Client()

	networkDomain, err := apiClient.GetNetworkDomain(networkDomainID)
	if err != nil {
		return err
	}

	if networkDomain == nil {
		return fmt.Errorf("No network domain was found with Id '%s'.", networkDomainID)
	}

	dataCenterID := networkDomain.DatacenterID
	log.Printf("Server will be deployed in data centre '%s'.", dataCenterID)

	propertyHelper := propertyHelper(data)

	// Retrieve image details.
	osImageID := propertyHelper.GetOptionalString(resourceKeyServerOSImageID, false)
	osImageName := propertyHelper.GetOptionalString(resourceKeyServerOSImageName, false)

	var osImage *compute.OSImage
	if osImageID != nil {
		// TODO: Look up OS image by Id (first, implement in compute API client).

		return fmt.Errorf("Specifying osimage_id is not supported yet.")
	} else if osImageName != nil {
		log.Printf("Looking up OS image '%s' by name...", *osImageName)

		osImage, err = apiClient.FindOSImage(*osImageName, dataCenterID)
		if err != nil {
			return err
		}

		if osImage == nil {
			log.Printf("Warning - unable to find an OS image named '%s' in data centre '%s' (which is where the target network domain, '%s', is located).", *osImageName, dataCenterID, networkDomainID)

			return fmt.Errorf("Unable to find an OS image named '%s' in data centre '%s' (which is where the target network domain, '%s', is located).", *osImageName, dataCenterID, networkDomainID)
		}

		log.Printf("Server will be deployed from OS image with Id '%s'.", osImage.ID)
		data.Set(resourceKeyServerOSImageID, osImage.ID)
	} else {
		return fmt.Errorf("Must specify either osimage_id or osimage_name.")
	}

	deploymentConfiguration := compute.ServerDeploymentConfiguration{
		Name:                  name,
		Description:           description,
		AdministratorPassword: adminPassword,
		Start: autoStart,
	}
	err = deploymentConfiguration.ApplyImage(osImage)
	if err != nil {
		return err
	}

	// Memory and CPU
	memoryGB := propertyHelper.GetOptionalInt(resourceKeyServerMemoryGB, false)
	if memoryGB != nil {
		deploymentConfiguration.MemoryGB = *memoryGB
	} else {
		data.Set(resourceKeyServerMemoryGB, deploymentConfiguration.MemoryGB)
	}

	cpuCount := propertyHelper.GetOptionalInt(resourceKeyServerCPUCount, false)
	if cpuCount != nil {
		deploymentConfiguration.CPU.Count = *cpuCount
	} else {
		data.Set(resourceKeyServerCPUCount, deploymentConfiguration.CPU.Count)
	}

	// Network
	primaryVLANID := propertyHelper.GetOptionalString(resourceKeyServerPrimaryVLAN, false)
	primaryIPv4Address := propertyHelper.GetOptionalString(resourceKeyServerPrimaryIPv4, false)

	deploymentConfiguration.Network = compute.VirtualMachineNetwork{
		NetworkDomainID: networkDomainID,
		PrimaryAdapter: compute.VirtualMachineNetworkAdapter{
			VLANID:             primaryVLANID,
			PrivateIPv4Address: primaryIPv4Address,
		},
	}
	deploymentConfiguration.PrimaryDNS = primaryDNS
	deploymentConfiguration.SecondaryDNS = secondaryDNS

	log.Printf("Server deployment configuration: %+v", deploymentConfiguration)
	log.Printf("Server CPU deployment configuration: %+v", deploymentConfiguration.CPU)

	serverID, err := apiClient.DeployServer(deploymentConfiguration)
	if err != nil {
		return err
	}

	data.SetId(serverID)

	log.Printf("Server '%s' is being provisioned...", name)

	resource, err := apiClient.WaitForDeploy(compute.ResourceTypeServer, serverID, resourceCreateTimeoutServer)
	if err != nil {
		return err
	}

	// Capture additional properties that may only be available after deployment.
	server := resource.(*compute.Server)
	captureServerNetworkConfiguration(server, data, false)

	data.Partial(true)

	err = applyServerTags(data, apiClient)
	if err != nil {
		return err
	}
	data.SetPartial(resourceKeyServerTag)

	err = createDisks(server.Disks, data, apiClient)
	if err != nil {
		return err
	}

	data.Partial(false)

	return nil
}

// Read a server resource.
func resourceServerRead(data *schema.ResourceData, provider interface{}) error {
	id := data.Id()
	name := data.Get(resourceKeyServerName).(string)
	description := data.Get(resourceKeyServerDescription).(string)
	networkDomainID := data.Get(resourceKeyServerNetworkDomainID).(string)

	log.Printf("Read server '%s' (Id = '%s') in network domain '%s' (description = '%s').", name, id, networkDomainID, description)

	apiClient := provider.(*providerState).Client()
	server, err := apiClient.GetServer(id)
	if err != nil {
		return err
	}

	if server == nil {
		log.Printf("Server '%s' has been deleted.", id)

		// Mark as deleted.
		data.SetId("")

		return nil
	}

	data.Set(resourceKeyServerName, server.Name)
	data.Set(resourceKeyServerDescription, server.Description)
	data.Set(resourceKeyServerOSImageID, server.SourceImageID)
	data.Set(resourceKeyServerMemoryGB, server.MemoryGB)
	data.Set(resourceKeyServerCPUCount, server.CPU.Count)

	captureServerNetworkConfiguration(server, data, false)

	err = readServerTags(data, apiClient)
	if err != nil {
		return err
	}

	propertyHelper := propertyHelper(data)
	propertyHelper.SetServerDisks(server.Disks)

	return nil
}

// Update a server resource.
func resourceServerUpdate(data *schema.ResourceData, provider interface{}) error {
	serverID := data.Id()

	// These changes can only be made through the V1 API (we're mostly using V2).
	// Later, we can come back and implement the required functionality in the compute API client.
	if data.HasChange(resourceKeyServerName) {
		return fmt.Errorf("Changing the 'name' property of a 'ddcloud_server' resource type is not yet implemented.")
	}

	if data.HasChange(resourceKeyServerDescription) {
		return fmt.Errorf("Changing the 'description' property of a 'ddcloud_server' resource type is not yet implemented.")
	}

	log.Printf("Update server '%s'.", serverID)

	apiClient := provider.(*providerState).Client()
	server, err := apiClient.GetServer(serverID)
	if err != nil {
		return err
	}

	data.Partial(true)

	propertyHelper := propertyHelper(data)

	var memoryGB, cpuCount *int
	if data.HasChange(resourceKeyServerMemoryGB) {
		memoryGB = propertyHelper.GetOptionalInt(resourceKeyServerMemoryGB, false)
	}
	if data.HasChange(resourceKeyServerCPUCount) {
		cpuCount = propertyHelper.GetOptionalInt(resourceKeyServerCPUCount, false)
	}

	if memoryGB != nil || cpuCount != nil {
		log.Printf("Server CPU / memory configuration change detected.")

		err = updateServerConfiguration(apiClient, server, memoryGB, cpuCount)
		if err != nil {
			return err
		}

		if data.HasChange(resourceKeyServerMemoryGB) {
			data.SetPartial(resourceKeyServerMemoryGB)
		}

		if data.HasChange(resourceKeyServerCPUCount) {
			data.SetPartial(resourceKeyServerCPUCount)
		}
	}

	var primaryIPv4, primaryIPv6 *string
	if data.HasChange(resourceKeyServerPrimaryIPv4) {
		primaryIPv4 = propertyHelper.GetOptionalString(resourceKeyServerPrimaryIPv4, false)
	}
	if data.HasChange(resourceKeyServerPrimaryIPv6) {
		primaryIPv6 = propertyHelper.GetOptionalString(resourceKeyServerPrimaryIPv6, false)
	}

	if primaryIPv4 != nil || primaryIPv6 != nil {
		log.Printf("Server network configuration change detected.")

		err = updateServerIPAddresses(apiClient, server, primaryIPv4, primaryIPv6)
		if err != nil {
			return err
		}

		if data.HasChange(resourceKeyServerPrimaryIPv4) {
			data.SetPartial(resourceKeyServerPrimaryIPv4)
		}

		if data.HasChange(resourceKeyServerPrimaryIPv6) {
			data.SetPartial(resourceKeyServerPrimaryIPv6)
		}
	}

	if data.HasChange(resourceKeyServerTag) {
		err = applyServerTags(data, apiClient)
		if err != nil {
			return err
		}

		data.SetPartial(resourceKeyServerTag)
	}

	if data.HasChange(resourceKeyServerDisk) {
		err = updateDisks(data, apiClient)
		if err != nil {
			return err
		}
	}

	data.Partial(false)

	return nil
}

// Delete a server resource.
func resourceServerDelete(data *schema.ResourceData, provider interface{}) error {
	var id, name, networkDomainID string

	id = data.Id()
	name = data.Get(resourceKeyServerName).(string)
	networkDomainID = data.Get(resourceKeyServerNetworkDomainID).(string)

	log.Printf("Delete server '%s' ('%s') in network domain '%s'.", id, name, networkDomainID)

	apiClient := provider.(*providerState).Client()
	server, err := apiClient.GetServer(id)
	if err != nil {
		return err
	}

	if server == nil {
		log.Printf("Server '%s' not found; will treat the server as having already been deleted.", id)

		return nil
	}

	if server.Started {
		log.Printf("Server '%s' is currently running. The server will be powered off.", id)

		err = apiClient.PowerOffServer(id)
		if err != nil {
			return err
		}

		_, err = apiClient.WaitForChange(compute.ResourceTypeServer, id, "Power off server", serverShutdownTimeout)
		if err != nil {
			return err
		}
	}

	log.Printf("Server '%s' is being deleted...", id)

	err = apiClient.DeleteServer(id)
	if err != nil {
		return err
	}

	return apiClient.WaitForDelete(compute.ResourceTypeServer, id, resourceDeleteTimeoutServer)
}
