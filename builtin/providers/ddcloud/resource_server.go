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
	resourceKeyServerAdditionalDisk  = "additional_disk"
	resourceKeyServerDiskID          = "disk_id"
	resourceKeyServerDiskSizeGB      = "size_gb"
	resourceKeyServerDiskUnitID      = "scsi_unit_id"
	resourceKeyServerDiskSpeed       = "speed"
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
			resourceKeyServerAdditionalDisk: &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Default:  nil,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						resourceKeyServerDiskID: &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						resourceKeyServerDiskSizeGB: &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						resourceKeyServerDiskUnitID: &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						resourceKeyServerDiskSpeed: &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "STANDARD",
						},
					},
				},
				Set: hashDiskUnitID,
			},
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

	data.Partial(true)

	resource, err := apiClient.WaitForDeploy(compute.ResourceTypeServer, serverID, resourceCreateTimeoutServer)
	if err != nil {
		return err
	}

	// Capture additional properties that may only be available after deployment.
	server := resource.(*compute.Server)

	serverIPv4Address := server.Network.PrimaryAdapter.PrivateIPv4Address
	data.Set(resourceKeyServerPrimaryIPv4, serverIPv4Address)
	data.SetPartial(resourceKeyServerPrimaryIPv4)

	serverIPv6Address := *server.Network.PrimaryAdapter.PrivateIPv6Address
	data.Set(resourceKeyServerPrimaryIPv6, serverIPv6Address)
	data.SetPartial(resourceKeyServerPrimaryIPv6)

	// Additional disks
	additionalDisks := propertyHelper.GetServerAdditionalDisks()
	if len(additionalDisks) == 0 {
		data.Partial(false)

		return nil
	}

	addedDisks := additionalDisks[:0]
	for index := range additionalDisks {
		disk := &additionalDisks[index]

		log.Printf("Adding disk with SCSI unit ID %d to server '%s'...", disk.SCSIUnitID, serverID)

		var diskID string
		diskID, err = apiClient.AddDiskToServer(serverID, disk.SCSIUnitID, disk.SizeGB, disk.Speed)
		if err != nil {
			return err
		}

		log.Printf("Disk with SCSI unit ID %d in server '%s' will have Id '%s'.", disk.SCSIUnitID, serverID, diskID)

		disk.ID = &diskID

		addedDisks = append(addedDisks, *disk)
		propertyHelper.SetServerAdditionalDisks(addedDisks)
		data.SetPartial(resourceKeyServerAdditionalDisk)

		_, err = apiClient.WaitForChange(
			compute.ResourceTypeServer,
			serverID,
			"Add disk",
			resourceUpdateTimeoutServer,
		)
		if err != nil {
			return err
		}

		log.Printf("Added disk with SCSI unit ID %d to server '%s' as disk '%s'.", disk.SCSIUnitID, serverID, diskID)
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

	// TODO: Update disks once we store both image and additional disks (until then we can't tell which disks are actually additional disks).

	data.Set(resourceKeyServerPrimaryVLAN, *server.Network.PrimaryAdapter.VLANID)
	data.Set(resourceKeyServerPrimaryIPv4, *server.Network.PrimaryAdapter.PrivateIPv4Address)
	data.Set(resourceKeyServerPrimaryIPv6, *server.Network.PrimaryAdapter.PrivateIPv6Address)
	data.Set(resourceKeyServerNetworkDomainID, server.Network.NetworkDomainID)

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

	// TODO: Handle disk changes.

	var primaryIPv4, primaryIPv6 *string
	if data.HasChange(resourceKeyServerPrimaryIPv4) {
		primaryIPv4 = propertyHelper.GetOptionalString(resourceKeyServerPrimaryIPv4, false)
	}
	if data.HasChange(resourceKeyServerPrimaryIPv6) {
		primaryIPv6 = propertyHelper.GetOptionalString(resourceKeyServerPrimaryIPv6, false)
	}

	if primaryIPv4 != nil || primaryIPv6 != nil {
		log.Printf("Server network configuration change detected.")

		err = updateServerIPAddress(apiClient, server, primaryIPv4, primaryIPv6)
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
		log.Printf("Server '%s' is currently running. The server will be shut down.", id)

		err = apiClient.ShutdownServer(id)
		if err != nil {
			return err
		}

		_, err = apiClient.WaitForChange(compute.ResourceTypeServer, id, "Shut down server", serverShutdownTimeout)
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

// updateServerIPAddress notifies the compute infrastructure that a server's IP address has changed.
func updateServerIPAddress(apiClient *compute.Client, server *compute.Server, primaryIPv4 *string, primaryIPv6 *string) error {
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

// Parse and append additional disks to those specified by the image being deployed.
func mergeDisks(imageDisks []compute.VirtualMachineDisk, additionalDisks []compute.VirtualMachineDisk) []compute.VirtualMachineDisk {
	diskSet := schema.NewSet(hashDiskUnitID, []interface{}{})

	for _, disk := range imageDisks {
		log.Printf("Merge image disk with SCSI unit Id %d.", disk.SCSIUnitID)
		diskSet.Add(disk)
	}

	for _, disk := range additionalDisks {
		log.Printf("Merge additional disk with SCSI unit Id %d.", disk.SCSIUnitID)
		diskSet.Add(disk)
	}

	mergedDisks := make([]compute.VirtualMachineDisk, diskSet.Len())
	for index, disk := range diskSet.List() {
		mergedDisks[index] = disk.(compute.VirtualMachineDisk)
	}

	return mergedDisks
}

func hashDiskUnitID(item interface{}) int {
	disk, ok := item.(compute.VirtualMachineDisk)
	if ok {
		return disk.SCSIUnitID
	}

	diskData := item.(map[string]interface{})

	return diskData[resourceKeyServerDiskUnitID].(int)
}
