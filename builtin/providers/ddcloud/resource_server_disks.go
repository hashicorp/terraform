package ddcloud

import (
	"fmt"
	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
)

const (
	resourceKeyServerDisk       = "disk"
	resourceKeyServerDiskID     = "disk_id"
	resourceKeyServerDiskSizeGB = "size_gb"
	resourceKeyServerDiskUnitID = "scsi_unit_id"
	resourceKeyServerDiskSpeed  = "speed"
)

func schemaServerDisk() *schema.Schema {
	return &schema.Schema{
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
	}
}

// When creating a server resource, synchronise the server's disks with its resource data.
// imageDisks refers to the newly-deployed server's collection of disks (i.e. image disks).
func createDisks(imageDisks []compute.VirtualMachineDisk, data *schema.ResourceData, apiClient *compute.Client) (err error) {
	propertyHelper := propertyHelper(data)

	serverID := data.Id()

	log.Printf("Configuring image disks for server '%s'...", serverID)

	configuredDisks := propertyHelper.GetServerDisks()

	if len(configuredDisks) == 0 {
		// Since this is the first time, populate image disks.
		var serverDisks []compute.VirtualMachineDisk
		for _, disk := range configuredDisks {
			serverDisks = append(serverDisks, disk)
		}

		propertyHelper.SetServerDisks(serverDisks)
		propertyHelper.SetPartial(resourceKeyServerDisk)

		return
	}

	// First, handle disks that were part of the original server image.
	log.Printf("Configure image disks for server '%s'...", serverID)

	disksByUnitID := getDisksByUnitID(configuredDisks)
	for _, imageDisk := range imageDisks {
		serverImageDisk, ok := disksByUnitID[imageDisk.SCSIUnitID]
		if !ok {
			// This is not an image disk.
			log.Printf("No existing disk was found with SCSI unit Id %d for server '%s'; this disk will be treated as an additional disk.", imageDisk.SCSIUnitID, serverID)

			continue
		}

		// This is an image disk, so we don't want to see it when we're configuring additional disks
		delete(disksByUnitID, serverImageDisk.SCSIUnitID)

		imageDiskID := *serverImageDisk.ID

		if imageDisk.SizeGB == serverImageDisk.SizeGB {
			continue // Nothing to do.
		}

		if imageDisk.SizeGB < serverImageDisk.SizeGB {

			// Can't shrink disk, only grow it.
			err = fmt.Errorf(
				"Cannot resize disk '%s' for server '%s' from %d to GB to %d (for now, disks can only be expanded).",
				imageDiskID,
				serverID,
				serverImageDisk.SizeGB,
				imageDisk.SizeGB,
			)

			return
		}

		// Do we need to expand the disk?
		if imageDisk.SizeGB > serverImageDisk.SizeGB {
			log.Printf(
				"Expanding disk '%s' for server '%s' (from %d GB to %d GB)...",
				imageDiskID,
				serverID,
				serverImageDisk.SizeGB,
				imageDisk.SizeGB,
			)

			response, err := apiClient.ResizeServerDisk(serverID, imageDiskID, imageDisk.SizeGB)
			if err != nil {
				return err
			}
			if response.Result != compute.ResultSuccess {
				return response.ToError(
					"Unexpected result '%s' when resizing server disk '%s' for server '%s'.",
					response.Result,
					imageDiskID,
					serverID,
				)
			}

			resource, err := apiClient.WaitForChange(
				compute.ResourceTypeServer,
				serverID,
				"Resize disk",
				resourceUpdateTimeoutServer,
			)
			if err != nil {
				return err
			}

			server := resource.(*compute.Server)
			propertyHelper.SetServerDisks(server.Disks)
			propertyHelper.SetPartial(resourceKeyServerDisk)

			log.Printf(
				"Resized disk '%s' for server '%s' (from %d to GB to %d).",
				imageDiskID,
				serverID,
				serverImageDisk.SizeGB,
				imageDisk.SizeGB,
			)
		}
	}

	// By process of elimination, any remaining disks must be additional disks.
	log.Printf("Configure additional disks for server '%s'...", serverID)

	for additionalDiskUnitID := range disksByUnitID {
		log.Printf("Add disk with SCSI unit ID %d to server '%s'...", additionalDiskUnitID, serverID)

		additionalDisk := disksByUnitID[additionalDiskUnitID]

		var additionalDiskID string
		additionalDiskID, err = apiClient.AddDiskToServer(
			serverID,
			additionalDisk.SCSIUnitID,
			additionalDisk.SizeGB,
			additionalDisk.Speed,
		)
		if err != nil {
			return
		}

		log.Printf("Adding disk '%s' with SCSI unit ID %d to server '%s'...", additionalDiskID, additionalDisk.SCSIUnitID, serverID)

		additionalDisk.ID = &additionalDiskID

		var resource compute.Resource
		resource, err = apiClient.WaitForChange(
			compute.ResourceTypeServer,
			serverID,
			"Add disk",
			resourceUpdateTimeoutServer,
		)
		if err != nil {
			return
		}

		server := resource.(*compute.Server)
		propertyHelper.SetServerDisks(server.Disks)
		propertyHelper.SetPartial(resourceKeyServerDisk)

		log.Printf(
			"Added disk '%s' with SCSI unit ID %d to server '%s'.",
			additionalDiskID,
			additionalDisk.SCSIUnitID,
			serverID,
		)
	}

	return nil
}

// When updating a server resource, synchronise the server's image disk attributes with its resource data
// Removes image disks from existingDisksByUnitID as they are processed, leaving only additional disks.
func updateDisks(data *schema.ResourceData, apiClient *compute.Client) error {
	propertyHelper := propertyHelper(data)

	serverID := data.Id()

	log.Printf("Configure image disks for server '%s'...", serverID)

	server, err := apiClient.GetServer(serverID)
	if err != nil {
		return err
	}
	if server == nil {
		data.SetId("")

		return fmt.Errorf("Server '%s' has been deleted.", serverID)
	}

	configuredDisks := propertyHelper.GetServerDisks()
	if len(configuredDisks) == 0 {
		// No explicitly-configured disks.
		propertyHelper.SetServerDisks(server.Disks)
		propertyHelper.SetPartial(resourceKeyServerDisk)

		return nil
	}

	disksByUnitID := getDisksByUnitID(server.Disks)
	for _, configuredDisk := range configuredDisks {
		existingDisk, ok := disksByUnitID[configuredDisk.SCSIUnitID]
		if ok {
			diskID := *existingDisk.ID

			// Existing disk.
			log.Printf("Examining existing disk '%s' with SCSI unit Id %d in server '%s'...", diskID, existingDisk.SCSIUnitID, serverID)

			if configuredDisk.SizeGB == existingDisk.SizeGB {
				log.Printf("Disk '%s' with SCSI unit Id %d in server '%s' is up-to-date; nothing to do.", diskID, existingDisk.SCSIUnitID, serverID)

				continue // Nothing to do.
			}

			// Currently we can't shrink a disk, only grow it.
			if configuredDisk.SizeGB < existingDisk.SizeGB {
				log.Printf("Disk '%s' with SCSI unit Id %d in server '%s' is larger than the size specified in the server configuration; this is currently unsupported.", diskID, existingDisk.SCSIUnitID, serverID)

				return fmt.Errorf(
					"Cannot shrink disk '%s' for server '%s' from %d to GB to %d (for now, disks can only be expanded).",
					diskID,
					serverID,
					existingDisk.SizeGB,
					configuredDisk.SizeGB,
				)
			}

			// We need to expand the disk.
			log.Printf(
				"Expanding disk '%s' for server '%s' (from %d GB to %d GB)...",
				diskID,
				serverID,
				existingDisk.SizeGB,
				configuredDisk.SizeGB,
			)

			response, err := apiClient.ResizeServerDisk(serverID, diskID, configuredDisk.SizeGB)
			if err != nil {
				return err
			}
			if response.Result != compute.ResultSuccess {
				return response.ToError("Unexpected result '%s' when resizing server disk '%s' for server '%s'.", response.Result, diskID, serverID)
			}

			resource, err := apiClient.WaitForChange(
				compute.ResourceTypeServer,
				serverID,
				"Resize disk",
				resourceUpdateTimeoutServer,
			)
			if err != nil {
				return err
			}

			server := resource.(*compute.Server)

			propertyHelper.SetServerDisks(server.Disks)
			propertyHelper.SetPartial(resourceKeyServerDisk)

			log.Printf(
				"Resized disk '%s' for server '%s' (from %d to GB to %d).",
				diskID,
				serverID,
				existingDisk.SizeGB,
				configuredDisk.SizeGB,
			)
		} else {
			// New disk.
			log.Printf("Adding disk with SCSI unit ID %d to server '%s'...", configuredDisk.SCSIUnitID, serverID)

			var diskID string
			diskID, err = apiClient.AddDiskToServer(
				serverID,
				configuredDisk.SCSIUnitID,
				configuredDisk.SizeGB,
				configuredDisk.Speed,
			)
			if err != nil {
				return err
			}

			log.Printf("New disk '%s' has SCSI unit ID %d in server '%s'...", diskID, configuredDisk.SCSIUnitID, serverID)

			var resource compute.Resource
			resource, err = apiClient.WaitForChange(
				compute.ResourceTypeServer,
				serverID,
				"Add disk",
				resourceUpdateTimeoutServer,
			)
			if err != nil {
				return err
			}

			server := resource.(*compute.Server)
			propertyHelper.SetServerDisks(server.Disks)
			propertyHelper.SetPartial(resourceKeyServerDisk)

			log.Printf(
				"Added disk '%s' with SCSI unit ID %d to server '%s'.",
				diskID,
				configuredDisk.SCSIUnitID,
				serverID,
			)
		}
	}

	return nil
}

func getDisksByUnitID(disks []compute.VirtualMachineDisk) map[int]*compute.VirtualMachineDisk {
	disksByUnitID := make(map[int]*compute.VirtualMachineDisk)
	for index := range disks {
		disk := disks[index]
		disksByUnitID[disk.SCSIUnitID] = &disk
	}

	return disksByUnitID
}

func mergeAdditionalDisks(disksByUnitID map[int]*compute.VirtualMachineDisk, additionalDisks []compute.VirtualMachineDisk) {
	for _, disk := range additionalDisks {
		disksByUnitID[disk.SCSIUnitID] = &disk
	}
}

func hashDiskUnitID(item interface{}) int {
	disk, ok := item.(compute.VirtualMachineDisk)
	if ok {
		return disk.SCSIUnitID
	}

	diskData := item.(map[string]interface{})

	return diskData[resourceKeyServerDiskUnitID].(int)
}
