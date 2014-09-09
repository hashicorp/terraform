package google

import (
	"fmt"
	"log"
	"time"

	"code.google.com/p/google-api-go-client/compute/v1"
	"code.google.com/p/google-api-go-client/googleapi"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceComputeInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeInstanceCreate,
		Read:   resourceComputeInstanceRead,
		Update: resourceComputeInstanceUpdate,
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
						// TODO(mitchellh): one of image or disk is required

						"disk": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"image": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"auto_delete": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
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

						"address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"name": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},

						"internal_address": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"metadata": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeMap,
				},
			},

			"tags": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			"metadata_fingerprint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags_fingerprint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
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
		prefix := fmt.Sprintf("disk.%d", i)

		// var sourceLink string

		// Build the disk
		var disk compute.AttachedDisk
		disk.Type = "PERSISTENT"
		disk.Mode = "READ_WRITE"
		disk.Boot = i == 0
		disk.AutoDelete = true

		if v, ok := d.GetOk(prefix + ".auto_delete"); ok {
			disk.AutoDelete = v.(bool)
		}

		// Load up the disk for this disk if specified
		if v, ok := d.GetOk(prefix + ".disk"); ok {
			diskName := v.(string)
			diskData, err := config.clientCompute.Disks.Get(
				config.Project, zone.Name, diskName).Do()
			if err != nil {
				return fmt.Errorf(
					"Error loading disk '%s': %s",
					diskName, err)
			}

			disk.Source = diskData.SelfLink
		}

		// Load up the image for this disk if specified
		if v, ok := d.GetOk(prefix + ".image"); ok {
			imageName := v.(string)
			image, err := readImage(config, imageName)
			if err != nil {
				return fmt.Errorf(
					"Error loading image '%s': %s",
					imageName, err)
			}

			disk.InitializeParams = &compute.AttachedDiskInitializeParams{
				SourceImage: image.SelfLink,
			}
		}

		disks = append(disks, &disk)
	}

	// Build up the list of networks
	networksCount := d.Get("network.#").(int)
	networks := make([]*compute.NetworkInterface, 0, networksCount)
	for i := 0; i < networksCount; i++ {
		prefix := fmt.Sprintf("network.%d", i)
		// Load up the name of this network
		networkName := d.Get(prefix + ".source").(string)
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
				Type:  "ONE_TO_ONE_NAT",
				NatIP: d.Get(prefix + ".address").(string),
			},
		}
		iface.Network = network.SelfLink

		networks = append(networks, &iface)
	}

	// Create the instance information
	instance := compute.Instance{
		Description:       d.Get("description").(string),
		Disks:             disks,
		MachineType:       machineType.SelfLink,
		Metadata:          resourceInstanceMetadata(d),
		Name:              d.Get("name").(string),
		NetworkInterfaces: networks,
		Tags:              resourceInstanceTags(d),
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
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for instance to create: %s", err)
	}
	op = opRaw.(*compute.Operation)
	if op.Error != nil {
		// The resource didn't actually create
		d.SetId("")

		// Return the error
		return OperationError(*op.Error)
	}

	return resourceComputeInstanceRead(d, meta)
}

func resourceComputeInstanceRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	instance, err := config.clientCompute.Instances.Get(
		config.Project, d.Get("zone").(string), d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading instance: %s", err)
	}

	// Set the networks
	for i, iface := range instance.NetworkInterfaces {
		prefix := fmt.Sprintf("network.%d", i)
		d.Set(prefix+".name", iface.Name)
		d.Set(prefix+".internal_address", iface.NetworkIP)
	}

	// Set the metadata fingerprint if there is one.
	if instance.Metadata != nil {
		d.Set("metadata_fingerprint", instance.Metadata.Fingerprint)
	}

	// Set the tags fingerprint if there is one.
	if instance.Tags != nil {
		d.Set("tags_fingerprint", instance.Tags.Fingerprint)
	}

	return nil
}

func resourceComputeInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Enable partial mode for the resource since it is possible
	d.Partial(true)

	// If the Metadata has changed, then update that.
	if d.HasChange("metadata") {
		metadata := resourceInstanceMetadata(d)
		op, err := config.clientCompute.Instances.SetMetadata(
			config.Project, d.Get("zone").(string), d.Id(), metadata).Do()
		if err != nil {
			return fmt.Errorf("Error updating metadata: %s", err)
		}

		w := &OperationWaiter{
			Service: config.clientCompute,
			Op:      op,
			Project: config.Project,
			Zone:    d.Get("zone").(string),
			Type:    OperationWaitZone,
		}
		state := w.Conf()
		state.Delay = 1 * time.Second
		state.Timeout = 5 * time.Minute
		state.MinTimeout = 2 * time.Second
		opRaw, err := state.WaitForState()
		if err != nil {
			return fmt.Errorf("Error waiting for metadata to update: %s", err)
		}
		op = opRaw.(*compute.Operation)
		if op.Error != nil {
			// Return the error
			return OperationError(*op.Error)
		}

		d.SetPartial("metadata")
	}

	if d.HasChange("tags") {
		tags := resourceInstanceTags(d)
		op, err := config.clientCompute.Instances.SetTags(
			config.Project, d.Get("zone").(string), d.Id(), tags).Do()
		if err != nil {
			return fmt.Errorf("Error updating tags: %s", err)
		}

		w := &OperationWaiter{
			Service: config.clientCompute,
			Op:      op,
			Project: config.Project,
			Zone:    d.Get("zone").(string),
			Type:    OperationWaitZone,
		}
		state := w.Conf()
		state.Delay = 1 * time.Second
		state.Timeout = 5 * time.Minute
		state.MinTimeout = 2 * time.Second
		opRaw, err := state.WaitForState()
		if err != nil {
			return fmt.Errorf("Error waiting for tags to update: %s", err)
		}
		op = opRaw.(*compute.Operation)
		if op.Error != nil {
			// Return the error
			return OperationError(*op.Error)
		}

		d.SetPartial("tags")
	}

	// We made it, disable partial mode
	d.Partial(false)

	return resourceComputeInstanceRead(d, meta)
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
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for instance to delete: %s", err)
	}
	op = opRaw.(*compute.Operation)
	if op.Error != nil {
		// Return the error
		return OperationError(*op.Error)
	}

	d.SetId("")
	return nil
}

func resourceInstanceMetadata(d *schema.ResourceData) *compute.Metadata {
	var metadata *compute.Metadata
	if v := d.Get("metadata").([]interface{}); len(v) > 0 {
		m := new(compute.Metadata)
		m.Items = make([]*compute.MetadataItems, 0, len(v))
		for _, v := range v {
			for k, v := range v.(map[string]interface{}) {
				m.Items = append(m.Items, &compute.MetadataItems{
					Key:   k,
					Value: v.(string),
				})
			}
		}

		// Set the fingerprint. If the metadata has never been set before
		// then this will just be blank.
		m.Fingerprint = d.Get("metadata_fingerprint").(string)

		metadata = m
	}

	return metadata
}

func resourceInstanceTags(d *schema.ResourceData) *compute.Tags {
	// Calculate the tags
	var tags *compute.Tags
	if v := d.Get("tags"); v != nil {
		vs := v.(*schema.Set).List()
		tags = new(compute.Tags)
		tags.Items = make([]string, len(vs))
		for i, v := range v.(*schema.Set).List() {
			tags.Items[i] = v.(string)
		}

		tags.Fingerprint = d.Get("tags_fingerprint").(string)
	}

	return tags
}
