package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func resourceComputeDisk() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeDiskCreate,
		Read:   resourceComputeDiskRead,
		Delete: resourceComputeDiskDelete,

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

			"image": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"size": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"snapshot": &schema.Schema{
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

func resourceComputeDiskCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Get the zone
	log.Printf("[DEBUG] Loading zone: %s", d.Get("zone").(string))
	zone, err := config.clientCompute.Zones.Get(
		config.Project, d.Get("zone").(string)).Do()
	if err != nil {
		return fmt.Errorf(
			"Error loading zone '%s': %s", d.Get("zone").(string), err)
	}

	// Build the disk parameter
	disk := &compute.Disk{
		Name:   d.Get("name").(string),
		SizeGb: int64(d.Get("size").(int)),
	}

	// If we were given a source image, load that.
	if v, ok := d.GetOk("image"); ok {
		log.Printf("[DEBUG] Resolving image name: %s", v.(string))
		imageUrl, err := resolveImage(config, v.(string))
		if err != nil {
			return fmt.Errorf(
				"Error resolving image name '%s': %s",
				v.(string), err)
		}

		disk.SourceImage = imageUrl
	}

	if v, ok := d.GetOk("type"); ok {
		log.Printf("[DEBUG] Loading disk type: %s", v.(string))
		diskType, err := readDiskType(config, zone, v.(string))
		if err != nil {
			return fmt.Errorf(
				"Error loading disk type '%s': %s",
				v.(string), err)
		}

		disk.Type = diskType.SelfLink
	}

	if v, ok := d.GetOk("snapshot"); ok {
		snapshotName := v.(string)
		log.Printf("[DEBUG] Loading snapshot: %s", snapshotName)
		snapshotData, err := config.clientCompute.Snapshots.Get(
			config.Project, snapshotName).Do()

		if err != nil {
			return fmt.Errorf(
				"Error loading snapshot '%s': %s",
				snapshotName, err)
		}

		disk.SourceSnapshot = snapshotData.SelfLink
	}

	op, err := config.clientCompute.Disks.Insert(
		config.Project, d.Get("zone").(string), disk).Do()
	if err != nil {
		return fmt.Errorf("Error creating disk: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(disk.Name)

	err = computeOperationWaitZone(config, op, d.Get("zone").(string), "Creating Disk")
	if err != nil {
		return err
	}
	return resourceComputeDiskRead(d, meta)
}

func resourceComputeDiskRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	disk, err := config.clientCompute.Disks.Get(
		config.Project, d.Get("zone").(string), d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing Disk %q because it's gone", d.Get("name").(string))
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading disk: %s", err)
	}

	d.Set("self_link", disk.SelfLink)

	return nil
}

func resourceComputeDiskDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Delete the disk
	op, err := config.clientCompute.Disks.Delete(
		config.Project, d.Get("zone").(string), d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting disk: %s", err)
	}

	zone := d.Get("zone").(string)
	err = computeOperationWaitZone(config, op, zone, "Creating Disk")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
