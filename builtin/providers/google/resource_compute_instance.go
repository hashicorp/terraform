package google

import (
	"fmt"
	"log"
	"time"

	"code.google.com/p/google-api-go-client/compute/v1"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceComputeInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeInstanceCreate,
		Read:   resourceComputeInstanceRead,
		Delete: resourceComputeInstanceDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"machine_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"disk": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"source": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"network": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"source": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceComputeInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Get the zone
	log.Printf("[DEBUG] Loading zone: %s", d.Get("zone").(string))
	zone, err := config.clientCompute.Zones.Get(
		config.Project, d.Get("zone").(string)).Do()
	if err != nil {
		return fmt.Errorf(
			"Error loading zone '%s': %s", d.Get("zone").(string), err)
	}

	// Get the machine type
	log.Printf("[DEBUG] Loading machine type: %s", d.Get("machine_type").(string))
	machineType, err := config.clientCompute.MachineTypes.Get(
		config.Project, zone.Name, d.Get("machine_type").(string)).Do()
	if err != nil {
		return fmt.Errorf(
			"Error loading machine type: %s",
			err)
	}

	// Build up the list of disks
	disksCount := d.Get("disk.#").(int)
	disks := make([]*compute.AttachedDisk, 0, disksCount)
	for i := 0; i < disksCount; i++ {
		// Load up the image for this disk
		imageName := d.Get(fmt.Sprintf("disk.%d.source", i)).(string)
		image, err := readImage(config, imageName)
		if err != nil {
			return fmt.Errorf(
				"Error loading image '%s': %s",
				imageName, err)
		}

		// Build the disk
		var disk compute.AttachedDisk
		disk.Type = "PERSISTENT"
		disk.Mode = "READ_WRITE"
		disk.Boot = i == 0
		disk.AutoDelete = true
		disk.InitializeParams = &compute.AttachedDiskInitializeParams{
			SourceImage: image.SelfLink,
		}

		disks = append(disks, &disk)
	}

	// Build up the list of networks
	networksCount := d.Get("network.#").(int)
	networks := make([]*compute.NetworkInterface, 0, networksCount)
	for i := 0; i < networksCount; i++ {
		// Load up the name of this network
		networkName := d.Get(fmt.Sprintf("network.%d.source", i)).(string)
		network, err := config.clientCompute.Networks.Get(
			config.Project, networkName).Do()
		if err != nil {
			return fmt.Errorf(
				"Error loading network '%s': %s",
				networkName, err)
		}

		// Build the disk
		var iface compute.NetworkInterface
		iface.AccessConfigs = []*compute.AccessConfig{
			&compute.AccessConfig{
				Type: "ONE_TO_ONE_NAT",
			},
		}
		iface.Network = network.SelfLink

		networks = append(networks, &iface)
	}

	// Create the instance information
	instance := compute.Instance{
		Description: d.Get("description").(string),
		Disks:       disks,
		MachineType: machineType.SelfLink,
		/*
			Metadata: &compute.Metadata{
				Items: metadata,
			},
		*/
		Name:              d.Get("name").(string),
		NetworkInterfaces: networks,
		/*
			ServiceAccounts: []*compute.ServiceAccount{
				&compute.ServiceAccount{
					Email: "default",
					Scopes: []string{
						"https://www.googleapis.com/auth/userinfo.email",
						"https://www.googleapis.com/auth/compute",
						"https://www.googleapis.com/auth/devstorage.full_control",
					},
				},
			},
			Tags: &compute.Tags{
				Items: c.Tags,
			},
		*/
	}

	log.Printf("[INFO] Requesting instance creation")
	op, err := config.clientCompute.Instances.Insert(
		config.Project, zone.Name, &instance).Do()
	if err != nil {
		return fmt.Errorf("Error creating instance: %s", err)
	}

	// Store the ID now
	d.SetId(instance.Name)

	// Wait for the operation to complete
	w := &OperationWaiter{
		Service: config.clientCompute,
		Op:      op,
		Project: config.Project,
		Zone:    zone.Name,
		Type:    OperationWaitZone,
	}
	state := w.Conf()
	state.Delay = 10 * time.Second
	state.Timeout = 10 * time.Minute
	state.MinTimeout = 2 * time.Second
	if _, err := state.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for instance to create: %s", err)
	}

	return resourceComputeInstanceRead(d, meta)
}

func resourceComputeInstanceRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	_, err := config.clientCompute.Instances.Get(
		config.Project, d.Get("zone").(string), d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error reading instance: %s", err)
	}

	return nil
}

func resourceComputeInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	op, err := config.clientCompute.Instances.Delete(
		config.Project, d.Get("zone").(string), d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting instance: %s", err)
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
	state.Delay = 5 * time.Second
	state.Timeout = 5 * time.Minute
	state.MinTimeout = 2 * time.Second
	if _, err := state.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for instance to create: %s", err)
	}

	d.SetId("")
	return nil
}
