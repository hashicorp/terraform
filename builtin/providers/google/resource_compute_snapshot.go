package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func resourceComputeSnapshot() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeSnapshotCreate,
		Read:   resourceComputeSnapshotRead,
		Delete: resourceComputeSnapshotDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"snapshot_encryption_key_raw": &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				ForceNew:  true,
				Sensitive: true,
			},

			"snapshot_encryption_key_sha256": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"sourcedisk_encryption_key_raw": &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				ForceNew:  true,
				Sensitive: true,
			},

			"sourcedisk_encryption_key_sha256": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"sourcedisk_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"sourcedisk": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"disk": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceComputeSnapshotCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Build the snapshot parameter
	snapshot := &compute.Snapshot{
		Name: d.Get("name").(string),
	}

	disk := d.Get("disk").(string)

	if v, ok := d.GetOk("snapshot_encryption_key_raw"); ok {
		snapshot.SnapshotEncryptionKey = &compute.CustomerEncryptionKey{}
		snapshot.SnapshotEncryptionKey.RawKey = v.(string)
	}

	if v, ok := d.GetOk("sourcedisk_encryption_key_raw"); ok {
		snapshot.SourceDiskEncryptionKey = &compute.CustomerEncryptionKey{}
		snapshot.SourceDiskEncryptionKey.RawKey = v.(string)
	}

	op, err := config.clientCompute.Disks.CreateSnapshot(
		project, d.Get("zone").(string), disk, snapshot).Do()
	if err != nil {
		return fmt.Errorf("Error creating snapshot: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(snapshot.Name)

	err = computeOperationWaitZone(config, op, project, d.Get("zone").(string), "Creating Snapshot")
	if err != nil {
		return err
	}
	return resourceComputeSnapshotRead(d, meta)
}

func resourceComputeSnapshotRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	snapshot, err := config.clientCompute.Snapshots.Get(
		project, d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing Snapshot %q because it's gone", d.Get("name").(string))
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading snapshot: %s", err)
	}

	d.Set("self_link", snapshot.SelfLink)
	if snapshot.SnapshotEncryptionKey != nil && snapshot.SnapshotEncryptionKey.Sha256 != "" {
		d.Set("snapshot_encryption_key_sha256", snapshot.SnapshotEncryptionKey.Sha256)
	}

	return nil
}

func resourceComputeSnapshotDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Delete the snapshot
	op, err := config.clientCompute.Snapshots.Delete(
		project, d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing Snapshot %q because it's gone", d.Get("name").(string))
			// The resource doesn't exist anymore
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error deleting snapshot: %s", err)
	}

	zone := d.Get("zone").(string)
	err = computeOperationWaitZone(config, op, project, zone, "Deleting Snapshot")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
