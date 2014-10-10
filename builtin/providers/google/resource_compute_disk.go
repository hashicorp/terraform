package google

import (
	"fmt"
	"log"
	"time"

	"code.google.com/p/google-api-go-client/compute/v1"
	"code.google.com/p/google-api-go-client/googleapi"
	"github.com/hashicorp/terraform/helper/schema"
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
		log.Printf("[DEBUG] Loading image: %s", v.(string))
		image, err := readImage(config, v.(string))
		if err != nil {
			return fmt.Errorf(
				"Error loading image '%s': %s",
				v.(string), err)
		}

		disk.SourceImage = image.SelfLink
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

	op, err := config.clientCompute.Disks.Insert(
		config.Project, d.Get("zone").(string), disk).Do()
	if err != nil {
		return fmt.Errorf("Error creating disk: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(disk.Name)

	// Wait for the operation to complete
	w := &OperationWaiter{
		Service: config.clientCompute,
		Op:      op,
		Project: config.Project,
		Zone:    d.Get("zone").(string),
		Type:    OperationWaitZone,
	}
	state := w.Conf()
	state.Timeout = 2 * time.Minute
	state.MinTimeout = 1 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for disk to create: %s", err)
	}
	op = opRaw.(*compute.Operation)
	if op.Error != nil {
		// The resource didn't actually create
		d.SetId("")

		// Return the error
		return OperationError(*op.Error)
	}

	return resourceComputeDiskRead(d, meta)
}

func resourceComputeDiskRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	_, err := config.clientCompute.Disks.Get(
		config.Project, d.Get("zone").(string), d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading disk: %s", err)
	}

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

	// Wait for the operation to complete
	w := &OperationWaiter{
		Service: config.clientCompute,
		Op:      op,
		Project: config.Project,
		Zone:    d.Get("zone").(string),
		Type:    OperationWaitZone,
	}
	state := w.Conf()
	state.Timeout = 2 * time.Minute
	state.MinTimeout = 1 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for disk to delete: %s", err)
	}
	op = opRaw.(*compute.Operation)
	if op.Error != nil {
		// Return the error
		return OperationError(*op.Error)
	}

	d.SetId("")
	return nil
}
