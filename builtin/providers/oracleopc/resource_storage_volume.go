package opc

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/oracle/terraform-provider-compute/sdk/compute"
	"log"
)

func resourceStorageVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceStorageVolumeCreate,
		Read:   resourceStorageVolumeRead,
		Update: resourceStorageVolumeUpdate,
		Delete: resourceStorageVolumeDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"size": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"sizeInBytes": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"storage": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "/oracle/public/storage/default",
			},

			"tags": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"bootableImage": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"bootableImageVersion": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Default:  -1,
			},

			"snapshot": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"account": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},

			"snapshotId": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceStorageVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource data: %#v", d)

	sv := meta.(*OPCClient).StorageVolumes()
	name := d.Get("name").(string)
	properties := []string{d.Get("storage").(string)}

	spec := sv.NewStorageVolumeSpec(
		d.Get("size").(string),
		properties,
		name)

	if d.Get("description").(string) != "" {
		spec.SetDescription(d.Get("description").(string))
	}

	spec.SetTags(getTags(d))

	if d.Get("bootableImage") != "" {
		spec.SetBootableImage(d.Get("bootableImage").(string), d.Get("bootableImageVersion").(int))
	}

	if len(d.Get("snapshot").(*schema.Set).List()) > 0 {
		snapshotDetails := d.Get("snapshot").(*schema.Set).List()[0].(map[string]interface{})
		spec.SetSnapshot(
			snapshotDetails["name"].(string),
			snapshotDetails["account"].(string),
		)
	}

	if d.Get("snapshotId") != "" {
		spec.SetSnapshotID(d.Get("snapshotId").(string))
	}

	log.Printf("[DEBUG] Creating storage volume %s with spec %#v", name, spec)
	err := sv.CreateStorageVolume(spec)
	if err != nil {
		return fmt.Errorf("Error creating storage volume %s: %s", name, err)
	}

	log.Printf("[DEBUG] Waiting for storage volume %s to come online", name)
	info, err := sv.WaitForStorageVolumeOnline(name, meta.(*OPCClient).MaxRetryTimeout)
	if err != nil {
		return fmt.Errorf("Error waiting for storage volume %s to come online: %s", name, err)
	}

	log.Printf("[DEBUG] Created storage volume %s: %#v", name, info)

	cachedAttachments, attachmentsFound := meta.(*OPCClient).storageAttachmentsByVolumeCache[name]
	if attachmentsFound {
		log.Printf("[DEBUG] Rebuilding storage attachments for volume %s", name)
		for _, cachedAttachment := range cachedAttachments {
			log.Printf("[DEBUG] Rebuilding storage attachments between volume %s and instance %s",
				name,
				cachedAttachment.instanceName)

			attachmentInfo, err := meta.(*OPCClient).StorageAttachments().CreateStorageAttachment(
				cachedAttachment.index,
				cachedAttachment.instanceName,
				name,
			)

			if err != nil {
				return fmt.Errorf(
					"Error recreating storage attachment between volume %s and instance %s: %s",
					name,
					*cachedAttachment.instanceName,
					err)
			}
			err = meta.(*OPCClient).StorageAttachments().WaitForStorageAttachmentCreated(
				attachmentInfo.Name,
				meta.(*OPCClient).MaxRetryTimeout)
			if err != nil {
				return fmt.Errorf(
					"Error recreating storage attachment between volume %s and instance %s: %s",
					name,
					*cachedAttachment.instanceName,
					err)
			}
		}
		meta.(*OPCClient).storageAttachmentsByVolumeCache[name] = nil
	}

	d.SetId(name)
	updateResourceData(d, info)
	return nil
}

func getTags(d *schema.ResourceData) []string {
	tags := []string{}
	for _, i := range d.Get("tags").([]interface{}) {
		tags = append(tags, i.(string))
	}
	return tags
}

func updateResourceData(d *schema.ResourceData, info *compute.StorageVolumeInfo) error {
	d.Set("name", info.Name)
	d.Set("description", info.Description)
	d.Set("storage", info.Properties[0])
	d.Set("sizeInBytes", info.Size)
	d.Set("tags", info.Tags)
	d.Set("bootableImage", info.ImageList)
	d.Set("bootableImageVersion", info.ImageListEntry)
	if info.Snapshot != "" {
		d.Set("snapshot", map[string]interface{}{
			"name":    info.Snapshot,
			"account": info.SnapshotAccount,
		})
	}
	d.Set("snapshotId", info.SnapshotID)

	return nil
}

func resourceStorageVolumeRead(d *schema.ResourceData, meta interface{}) error {
	sv := meta.(*OPCClient).StorageVolumes()
	name := d.Get("name").(string)

	log.Printf("[DEBUG] Reading state of storage volume %s", name)
	result, err := sv.GetStorageVolume(name)
	if err != nil {
		// Volume doesn't exist
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading storage volume %s: %s", name, err)
	}

	if len(result.Result) == 0 {
		// Volume doesn't exist
		d.SetId("")
		return nil
	}

	log.Printf("[DEBUG] Read state of storage volume %s: %#v", name, &result.Result[0])
	updateResourceData(d, &result.Result[0])

	return nil
}

func resourceStorageVolumeUpdate(d *schema.ResourceData, meta interface{}) error {
	sv := meta.(*OPCClient).StorageVolumes()
	name := d.Get("name").(string)
	description := d.Get("description").(string)
	size := d.Get("size").(string)
	tags := getTags(d)

	log.Printf("[DEBUG] Updating storage volume %s with size %s, description %s, tags %#v", name, size, description, tags)
	err := sv.UpdateStorageVolume(name, size, description, tags)

	if err != nil {
		return fmt.Errorf("Error updating storage volume %s: %s", name, err)
	}

	log.Printf("[DEBUG] Waiting for updated storage volume %s to come online", name)
	info, err := sv.WaitForStorageVolumeOnline(name, meta.(*OPCClient).MaxRetryTimeout)
	if err != nil {
		return fmt.Errorf("Error waiting for updated storage volume %s to come online: %s", name, err)
	}

	log.Printf("[DEBUG] Updated storage volume %s: %#v", name, info)
	updateResourceData(d, info)
	return nil
}

func resourceStorageVolumeDelete(d *schema.ResourceData, meta interface{}) error {
	sv := meta.(*OPCClient).StorageVolumes()
	name := d.Get("name").(string)

	sva := meta.(*OPCClient).StorageAttachments()
	attachments, err := sva.GetStorageAttachmentsForVolume(name)
	if err != nil {
		return fmt.Errorf("Error retrieving storage attachments for volume %s: %s", name, err)
	}

	attachmentsToCache := make([]storageAttachment, len(*attachments))
	for index, attachment := range *attachments {
		log.Printf("[DEBUG] Deleting storage attachment %s for volume %s", attachment.Name, name)
		sva.DeleteStorageAttachment(attachment.Name)
		sva.WaitForStorageAttachmentDeleted(attachment.Name, meta.(*OPCClient).MaxRetryTimeout)
		attachmentsToCache[index] = storageAttachment{
			index:        attachment.Index,
			instanceName: compute.InstanceNameFromString(attachment.InstanceName),
		}
	}
	meta.(*OPCClient).storageAttachmentsByVolumeCache[name] = attachmentsToCache

	log.Printf("[DEBUG] Deleting storage volume %s", name)
	err = sv.DeleteStorageVolume(name)
	if err != nil {
		return fmt.Errorf("Error deleting storage volume %s: %s", name, err)
	}

	log.Printf("[DEBUG] Waiting for storage volume %s to finish deleting", name)
	err = sv.WaitForStorageVolumeDeleted(name, meta.(*OPCClient).MaxRetryTimeout)
	if err != nil {
		return fmt.Errorf("Error waiting for storage volume %s to finish deleting: %s", name, err)
	}

	log.Printf("[DEBUG] Deleted storage volume %s", name)
	return nil
}
