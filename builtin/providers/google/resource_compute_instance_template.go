package google

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func resourceComputeInstanceTemplate() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeInstanceTemplateCreate,
		Read:   resourceComputeInstanceTemplateRead,
		Delete: resourceComputeInstanceTemplateDelete,

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

			"can_ip_forward": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"instance_description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"machine_type": &schema.Schema{
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
						"auto_delete": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
							ForceNew: true,
						},

						"boot": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},

						"device_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"disk_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"disk_size_gb": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},

						"disk_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"source_image": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"interface": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"mode": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"source": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},

			"metadata": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"network_interface": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"network": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"access_config": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"nat_ip": &schema.Schema{
										Type:     schema.TypeString,
										Computed: true,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},

			"automatic_restart": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				ForceNew: true,
			},

			"on_host_maintenance": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"service_account": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"email": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
							ForceNew: true,
						},

						"scopes": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							ForceNew: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								StateFunc: func(v interface{}) string {
									return canonicalizeServiceScope(v.(string))
								},
							},
						},
					},
				},
			},

			"tags": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
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

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func buildDisks(d *schema.ResourceData, meta interface{}) []*compute.AttachedDisk {
	disksCount := d.Get("disk.#").(int)

	disks := make([]*compute.AttachedDisk, 0, disksCount)
	for i := 0; i < disksCount; i++ {
		prefix := fmt.Sprintf("disk.%d", i)

		// Build the disk
		var disk compute.AttachedDisk
		disk.Type = "PERSISTENT"
		disk.Mode = "READ_WRITE"
		disk.Interface = "SCSI"
		disk.Boot = i == 0
		disk.AutoDelete = d.Get(prefix + ".auto_delete").(bool)

		if v, ok := d.GetOk(prefix + ".boot"); ok {
			disk.Boot = v.(bool)
		}

		if v, ok := d.GetOk(prefix + ".device_name"); ok {
			disk.DeviceName = v.(string)
		}

		if v, ok := d.GetOk(prefix + ".source"); ok {
			disk.Source = v.(string)
		} else {
			disk.InitializeParams = &compute.AttachedDiskInitializeParams{}

			if v, ok := d.GetOk(prefix + ".disk_name"); ok {
				disk.InitializeParams.DiskName = v.(string)
			}
			if v, ok := d.GetOk(prefix + ".disk_size_gb"); ok {
				disk.InitializeParams.DiskSizeGb = v.(int64)
			}
			disk.InitializeParams.DiskType = "pd-standard"
			if v, ok := d.GetOk(prefix + ".disk_type"); ok {
				disk.InitializeParams.DiskType = v.(string)
			}

			if v, ok := d.GetOk(prefix + ".source_image"); ok {
				disk.InitializeParams.SourceImage = v.(string)
			}
		}

		if v, ok := d.GetOk(prefix + ".interface"); ok {
			disk.Interface = v.(string)
		}

		if v, ok := d.GetOk(prefix + ".mode"); ok {
			disk.Mode = v.(string)
		}

		if v, ok := d.GetOk(prefix + ".type"); ok {
			disk.Type = v.(string)
		}

		disks = append(disks, &disk)
	}

	return disks
}

func buildNetworks(d *schema.ResourceData, meta interface{}) (error, []*compute.NetworkInterface) {
	// Build up the list of networks
	networksCount := d.Get("network_interface.#").(int)
	networkInterfaces := make([]*compute.NetworkInterface, 0, networksCount)
	for i := 0; i < networksCount; i++ {
		prefix := fmt.Sprintf("network_interface.%d", i)

		source := "global/networks/default"
		if v, ok := d.GetOk(prefix + ".network"); ok {
			if v.(string) != "default" {
				source = v.(string)
			}
		}

		// Build the networkInterface
		var iface compute.NetworkInterface
		iface.Network = source

		accessConfigsCount := d.Get(prefix + ".access_config.#").(int)
		iface.AccessConfigs = make([]*compute.AccessConfig, accessConfigsCount)
		for j := 0; j < accessConfigsCount; j++ {
			acPrefix := fmt.Sprintf("%s.access_config.%d", prefix, j)
			iface.AccessConfigs[j] = &compute.AccessConfig{
				Type:  "ONE_TO_ONE_NAT",
				NatIP: d.Get(acPrefix + ".nat_ip").(string),
			}
		}

		networkInterfaces = append(networkInterfaces, &iface)
	}
	return nil, networkInterfaces
}

func resourceComputeInstanceTemplateCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	instanceProperties := &compute.InstanceProperties{}

	instanceProperties.CanIpForward = d.Get("can_ip_forward").(bool)
	instanceProperties.Description = d.Get("instance_description").(string)
	instanceProperties.MachineType = d.Get("machine_type").(string)
	instanceProperties.Disks = buildDisks(d, meta)
	metadata, err := resourceInstanceMetadata(d)
	if err != nil {
		return err
	}
	instanceProperties.Metadata = metadata
	err, networks := buildNetworks(d, meta)
	if err != nil {
		return err
	}
	instanceProperties.NetworkInterfaces = networks

	instanceProperties.Scheduling = &compute.Scheduling{
		AutomaticRestart: d.Get("automatic_restart").(bool),
	}
	instanceProperties.Scheduling.OnHostMaintenance = "MIGRATE"
	if v, ok := d.GetOk("on_host_maintenance"); ok {
		instanceProperties.Scheduling.OnHostMaintenance = v.(string)
	}

	serviceAccountsCount := d.Get("service_account.#").(int)
	serviceAccounts := make([]*compute.ServiceAccount, 0, serviceAccountsCount)
	for i := 0; i < serviceAccountsCount; i++ {
		prefix := fmt.Sprintf("service_account.%d", i)

		scopesCount := d.Get(prefix + ".scopes.#").(int)
		scopes := make([]string, 0, scopesCount)
		for j := 0; j < scopesCount; j++ {
			scope := d.Get(fmt.Sprintf(prefix+".scopes.%d", j)).(string)
			scopes = append(scopes, canonicalizeServiceScope(scope))
		}

		serviceAccount := &compute.ServiceAccount{
			Email:  "default",
			Scopes: scopes,
		}

		serviceAccounts = append(serviceAccounts, serviceAccount)
	}
	instanceProperties.ServiceAccounts = serviceAccounts

	instanceProperties.Tags = resourceInstanceTags(d)

	instanceTemplate := compute.InstanceTemplate{
		Description: d.Get("description").(string),
		Properties:  instanceProperties,
		Name:        d.Get("name").(string),
	}

	op, err := config.clientCompute.InstanceTemplates.Insert(
		config.Project, &instanceTemplate).Do()
	if err != nil {
		return fmt.Errorf("Error creating instance: %s", err)
	}

	// Store the ID now
	d.SetId(instanceTemplate.Name)

	// Wait for the operation to complete
	w := &OperationWaiter{
		Service: config.clientCompute,
		Op:      op,
		Project: config.Project,
		Type:    OperationWaitGlobal,
	}
	state := w.Conf()
	state.Delay = 10 * time.Second
	state.Timeout = 10 * time.Minute
	state.MinTimeout = 2 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for instance template to create: %s", err)
	}
	op = opRaw.(*compute.Operation)
	if op.Error != nil {
		// The resource didn't actually create
		d.SetId("")

		// Return the error
		return OperationError(*op.Error)
	}

	return resourceComputeInstanceTemplateRead(d, meta)
}

func resourceComputeInstanceTemplateRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	instanceTemplate, err := config.clientCompute.InstanceTemplates.Get(
		config.Project, d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading instance template: %s", err)
	}

	// Set the metadata fingerprint if there is one.
	if instanceTemplate.Properties.Metadata != nil {
		d.Set("metadata_fingerprint", instanceTemplate.Properties.Metadata.Fingerprint)
	}

	// Set the tags fingerprint if there is one.
	if instanceTemplate.Properties.Tags != nil {
		d.Set("tags_fingerprint", instanceTemplate.Properties.Tags.Fingerprint)
	}
	d.Set("self_link", instanceTemplate.SelfLink)

	return nil
}

func resourceComputeInstanceTemplateDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	op, err := config.clientCompute.InstanceTemplates.Delete(
		config.Project, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting instance template: %s", err)
	}

	// Wait for the operation to complete
	w := &OperationWaiter{
		Service: config.clientCompute,
		Op:      op,
		Project: config.Project,
		Type:    OperationWaitGlobal,
	}
	state := w.Conf()
	state.Delay = 5 * time.Second
	state.Timeout = 5 * time.Minute
	state.MinTimeout = 2 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for instance template to delete: %s", err)
	}
	op = opRaw.(*compute.Operation)
	if op.Error != nil {
		// Return the error
		return OperationError(*op.Error)
	}

	d.SetId("")
	return nil
}
