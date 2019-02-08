package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/sharedfilesystems/v2/errors"
	"github.com/gophercloud/gophercloud/openstack/sharedfilesystems/v2/messages"
	"github.com/gophercloud/gophercloud/openstack/sharedfilesystems/v2/shares"
)

const (
	// major share functionality appeared in 2.14
	minManilaShareMicroversion = "2.14"
)

func resourceSharedFilesystemShareV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceSharedFilesystemShareV2Create,
		Read:   resourceSharedFilesystemShareV2Read,
		Update: resourceSharedFilesystemShareV2Update,
		Delete: resourceSharedFilesystemShareV2Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"project_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"share_proto": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					"NFS", "CIFS", "CEPHFS", "GLUSTERFS", "HDFS", "MAPRFS",
				}, true),
			},

			"size": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntAtLeast(1),
			},

			"share_type": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"snapshot_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"is_public": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"metadata": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"share_network_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"availability_zone": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"export_locations": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"path": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"preferred": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"has_replicas": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"host": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"replication_type": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"share_server_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceSharedFilesystemShareV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	sfsClient, err := config.sharedfilesystemV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack sharedfilesystem client: %s", err)
	}

	sfsClient.Microversion = minManilaShareMicroversion

	isPublic := d.Get("is_public").(bool)

	metadataRaw := d.Get("metadata").(map[string]interface{})
	metadata := make(map[string]string, len(metadataRaw))
	for k, v := range metadataRaw {
		if stringVal, ok := v.(string); ok {
			metadata[k] = stringVal
		}
	}

	createOpts := shares.CreateOpts{
		Name:             d.Get("name").(string),
		Description:      d.Get("description").(string),
		ShareProto:       d.Get("share_proto").(string),
		Size:             d.Get("size").(int),
		SnapshotID:       d.Get("snapshot_id").(string),
		IsPublic:         &isPublic,
		Metadata:         metadata,
		ShareNetworkID:   d.Get("share_network_id").(string),
		AvailabilityZone: d.Get("availability_zone").(string),
	}

	if v, ok := d.GetOkExists("share_type"); ok {
		createOpts.ShareType = v.(string)
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)

	timeout := d.Timeout(schema.TimeoutCreate)

	log.Printf("[DEBUG] Attempting to create share")
	var share *shares.Share
	err = resource.Retry(timeout, func() *resource.RetryError {
		share, err = shares.Create(sfsClient, createOpts).Extract()
		if err != nil {
			return checkForRetryableError(err)
		}
		return nil
	})

	if err != nil {
		detailedErr := errors.ErrorDetails{}
		e := errors.ExtractErrorInto(err, &detailedErr)
		if e != nil {
			return fmt.Errorf("Error creating share: %s: %s", err, e)
		}
		for k, msg := range detailedErr {
			return fmt.Errorf("Error creating share: %s (%d): %s", k, msg.Code, msg.Message)
		}
	}

	d.SetId(share.ID)

	// Wait for share to become active before continuing
	err = waitForSFV2Share(sfsClient, share.ID, "available", []string{"creating", "manage_starting"}, timeout)
	if err != nil {
		return err
	}

	return resourceSharedFilesystemShareV2Read(d, meta)
}

func resourceSharedFilesystemShareV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	sfsClient, err := config.sharedfilesystemV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack sharedfilesystem client: %s", err)
	}

	sfsClient.Microversion = minManilaShareMicroversion

	share, err := shares.Get(sfsClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "share")
	}

	log.Printf("[DEBUG] Retrieved share %s: %#v", d.Id(), share)

	exportLocationsRaw, err := shares.GetExportLocations(sfsClient, d.Id()).Extract()
	if err != nil {
		return fmt.Errorf("Failed to retrieve share's export_locations %s: %s", d.Id(), err)
	}

	log.Printf("[DEBUG] Retrieved share's export_locations %s: %#v", d.Id(), exportLocationsRaw)

	var exportLocations []map[string]string
	for _, v := range exportLocationsRaw {
		exportLocations = append(exportLocations, map[string]string{
			"path":      v.Path,
			"preferred": fmt.Sprint(v.Preferred),
		})
	}
	if err = d.Set("export_locations", exportLocations); err != nil {
		log.Printf("[DEBUG] Unable to set export_locations: %s", err)
	}

	d.Set("name", share.Name)
	d.Set("description", share.Description)
	d.Set("share_proto", share.ShareProto)
	d.Set("size", share.Size)
	d.Set("share_type", share.ShareTypeName)
	d.Set("snapshot_id", share.SnapshotID)
	d.Set("is_public", share.IsPublic)
	d.Set("metadata", share.Metadata)
	d.Set("share_network_id", share.ShareNetworkID)
	d.Set("availability_zone", share.AvailabilityZone)
	// Computed
	d.Set("region", GetRegion(d, config))
	d.Set("project_id", share.ProjectID)
	d.Set("has_replicas", share.HasReplicas)
	d.Set("host", share.Host)
	d.Set("replication_type", share.ReplicationType)
	d.Set("share_server_id", share.ShareServerID)

	return nil
}

func resourceSharedFilesystemShareV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	sfsClient, err := config.sharedfilesystemV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack sharedfilesystem client: %s", err)
	}

	sfsClient.Microversion = minManilaShareMicroversion

	timeout := d.Timeout(schema.TimeoutUpdate)

	var updateOpts shares.UpdateOpts

	if d.HasChange("name") {
		name := d.Get("name").(string)
		updateOpts.DisplayName = &name
	}
	if d.HasChange("description") {
		description := d.Get("description").(string)
		updateOpts.DisplayDescription = &description
	}
	if d.HasChange("is_public") {
		isPublic := d.Get("is_public").(bool)
		updateOpts.IsPublic = &isPublic
	}

	if updateOpts != (shares.UpdateOpts{}) {
		// Wait for share to become active before continuing
		err = waitForSFV2Share(sfsClient, d.Id(), "available", []string{"creating", "manage_starting", "extending", "shrinking"}, timeout)
		if err != nil {
			return err
		}

		log.Printf("[DEBUG] Attempting to update share")
		err = resource.Retry(timeout, func() *resource.RetryError {
			_, err := shares.Update(sfsClient, d.Id(), updateOpts).Extract()
			if err != nil {
				return checkForRetryableError(err)
			}
			return nil
		})

		if err != nil {
			detailedErr := errors.ErrorDetails{}
			e := errors.ExtractErrorInto(err, &detailedErr)
			if e != nil {
				return fmt.Errorf("Error updating %s share: %s: %s", d.Id(), err, e)
			}
			for k, msg := range detailedErr {
				return fmt.Errorf("Error updating %s share: %s (%d): %s", d.Id(), k, msg.Code, msg.Message)
			}
		}

		// Wait for share to become active before continuing
		err = waitForSFV2Share(sfsClient, d.Id(), "available", []string{"creating", "manage_starting", "extending", "shrinking"}, timeout)
		if err != nil {
			return err
		}
	}

	if d.HasChange("size") {
		var pending []string
		oldSize, newSize := d.GetChange("size")

		if newSize.(int) > oldSize.(int) {
			pending = append(pending, "extending")
			resizeOpts := shares.ExtendOpts{NewSize: newSize.(int)}
			log.Printf("[DEBUG] Resizing share %s with options: %#v", d.Id(), resizeOpts)
			err = resource.Retry(timeout, func() *resource.RetryError {
				err := shares.Extend(sfsClient, d.Id(), resizeOpts).Err
				log.Printf("[DEBUG] Resizing share %s with options: %#v", d.Id(), resizeOpts)
				if err != nil {
					return checkForRetryableError(err)
				}
				return nil
			})
		} else if newSize.(int) < oldSize.(int) {
			pending = append(pending, "shrinking")
			resizeOpts := shares.ShrinkOpts{NewSize: newSize.(int)}
			log.Printf("[DEBUG] Resizing share %s with options: %#v", d.Id(), resizeOpts)
			err = resource.Retry(timeout, func() *resource.RetryError {
				err := shares.Shrink(sfsClient, d.Id(), resizeOpts).Err
				log.Printf("[DEBUG] Resizing share %s with options: %#v", d.Id(), resizeOpts)
				if err != nil {
					return checkForRetryableError(err)
				}
				return nil
			})
		}

		if err != nil {
			detailedErr := errors.ErrorDetails{}
			e := errors.ExtractErrorInto(err, &detailedErr)
			if e != nil {
				return fmt.Errorf("Unable to resize %s share: %s: %s", d.Id(), err, e)
			}
			for k, msg := range detailedErr {
				return fmt.Errorf("Unable to resize %s share: %s (%d): %s", d.Id(), k, msg.Code, msg.Message)
			}
		}

		// Wait for share to become active before continuing
		err = waitForSFV2Share(sfsClient, d.Id(), "available", pending, timeout)
		if err != nil {
			return err
		}
	}

	return resourceSharedFilesystemShareV2Read(d, meta)
}

func resourceSharedFilesystemShareV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	sfsClient, err := config.sharedfilesystemV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack sharedfilesystem client: %s", err)
	}

	timeout := d.Timeout(schema.TimeoutDelete)

	log.Printf("[DEBUG] Attempting to delete share %s", d.Id())
	err = resource.Retry(timeout, func() *resource.RetryError {
		err = shares.Delete(sfsClient, d.Id()).ExtractErr()
		if err != nil {
			return checkForRetryableError(err)
		}
		return nil
	})

	if err != nil {
		e := CheckDeleted(d, err, "")
		if e == nil {
			return nil
		}
		detailedErr := errors.ErrorDetails{}
		e = errors.ExtractErrorInto(err, &detailedErr)
		if e != nil {
			return fmt.Errorf("Unable to delete %s share: %s: %s", d.Id(), err, e)
		}
		for k, msg := range detailedErr {
			return fmt.Errorf("Unable to delete %s share: %s (%d): %s", d.Id(), k, msg.Code, msg.Message)
		}
	}

	// Wait for share to become deleted before continuing
	pending := []string{"", "deleting", "available"}
	err = waitForSFV2Share(sfsClient, d.Id(), "deleted", pending, timeout)
	if err != nil {
		return err
	}

	return nil
}

// Full list of the share statuses: https://developer.openstack.org/api-ref/shared-file-system/#shares
func waitForSFV2Share(sfsClient *gophercloud.ServiceClient, id string, target string, pending []string, timeout time.Duration) error {
	log.Printf("[DEBUG] Waiting for share %s to become %s.", id, target)

	stateConf := &resource.StateChangeConf{
		Target:     []string{target},
		Pending:    pending,
		Refresh:    resourceSFV2ShareRefreshFunc(sfsClient, id),
		Timeout:    timeout,
		Delay:      1 * time.Second,
		MinTimeout: 1 * time.Second,
	}

	_, err := stateConf.WaitForState()
	if err != nil {
		if _, ok := err.(gophercloud.ErrDefault404); ok {
			switch target {
			case "deleted":
				return nil
			default:
				return fmt.Errorf("Error: share %s not found: %s", id, err)
			}
		}
		errorMessage := fmt.Sprintf("Error waiting for share %s to become %s", id, target)
		msg := resourceSFSV2ShareManilaMessage(sfsClient, id)
		if msg == nil {
			return fmt.Errorf("%s: %s", errorMessage, err)
		}
		return fmt.Errorf("%s: %s: the latest manila message (%s): %s", errorMessage, err, msg.CreatedAt, msg.UserMessage)
	}

	return nil
}

func resourceSFV2ShareRefreshFunc(sfsClient *gophercloud.ServiceClient, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		share, err := shares.Get(sfsClient, id).Extract()
		if err != nil {
			return nil, "", err
		}
		return share, share.Status, nil
	}
}

func resourceSFSV2ShareManilaMessage(sfsClient *gophercloud.ServiceClient, id string) *messages.Message {
	// we can simply set this, because this function is called after the error occurred
	sfsClient.Microversion = "2.37"

	listOpts := messages.ListOpts{
		ResourceID: id,
		SortKey:    "created_at",
		SortDir:    "desc",
		Limit:      1,
	}
	allPages, err := messages.List(sfsClient, listOpts).AllPages()
	if err != nil {
		log.Printf("[DEBUG] Unable to retrieve messages: %v", err)
		return nil
	}

	allMessages, err := messages.ExtractMessages(allPages)
	if err != nil {
		log.Printf("[DEBUG] Unable to extract messages: %v", err)
		return nil
	}

	if len(allMessages) == 0 {
		log.Printf("[DEBUG] No messages found")
		return nil
	}

	return &allMessages[0]
}
