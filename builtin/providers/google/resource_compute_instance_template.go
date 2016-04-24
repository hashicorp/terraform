package google

import (
	"fmt"
	"log"

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

			"machine_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"automatic_restart": &schema.Schema{
				Type:       schema.TypeBool,
				Optional:   true,
				Default:    true,
				ForceNew:   true,
				Deprecated: "Please use `scheduling.automatic_restart` instead",
			},

			"can_ip_forward": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"instance_description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"metadata": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"metadata_fingerprint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"network_interface": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"network": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"subnetwork": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
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

			"on_host_maintenance": &schema.Schema{
				Type:       schema.TypeString,
				Optional:   true,
				ForceNew:   true,
				Deprecated: "Please use `scheduling.on_host_maintenance` instead",
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"scheduling": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"preemptible": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
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
					},
				},
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
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
				Set:      schema.HashString,
			},

			"tags_fingerprint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func buildDisks(d *schema.ResourceData, meta interface{}) ([]*compute.AttachedDisk, error) {
	config := meta.(*Config)

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
				disk.InitializeParams.DiskSizeGb = int64(v.(int))
			}
			disk.InitializeParams.DiskType = "pd-standard"
			if v, ok := d.GetOk(prefix + ".disk_type"); ok {
				disk.InitializeParams.DiskType = v.(string)
			}

			if v, ok := d.GetOk(prefix + ".source_image"); ok {
				imageName := v.(string)
				imageUrl, err := resolveImage(config, imageName)
				if err != nil {
					return nil, fmt.Errorf(
						"Error resolving image name '%s': %s",
						imageName, err)
				}
				disk.InitializeParams.SourceImage = imageUrl
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

	return disks, nil
}

func buildNetworks(d *schema.ResourceData, meta interface{}) ([]*compute.NetworkInterface, error) {
	// Build up the list of networks
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return nil, err
	}

	networksCount := d.Get("network_interface.#").(int)
	networkInterfaces := make([]*compute.NetworkInterface, 0, networksCount)
	for i := 0; i < networksCount; i++ {
		prefix := fmt.Sprintf("network_interface.%d", i)

		var networkName, subnetworkName string
		if v, ok := d.GetOk(prefix + ".network"); ok {
			networkName = v.(string)
		}
		if v, ok := d.GetOk(prefix + ".subnetwork"); ok {
			subnetworkName = v.(string)
		}

		if networkName == "" && subnetworkName == "" {
			return nil, fmt.Errorf("network or subnetwork must be provided")
		}
		if networkName != "" && subnetworkName != "" {
			return nil, fmt.Errorf("network or subnetwork must not both be provided")
		}

		var networkLink, subnetworkLink string
		if networkName != "" {
			network, err := config.clientCompute.Networks.Get(
				project, networkName).Do()
			if err != nil {
				return nil, fmt.Errorf("Error referencing network '%s': %s",
					networkName, err)
			}
			networkLink = network.SelfLink
		} else {
			// lookup subnetwork link using region and subnetwork name
			region, err := getRegion(d, config)
			if err != nil {
				return nil, err
			}
			subnetwork, err := config.clientCompute.Subnetworks.Get(
				project, region, subnetworkName).Do()
			if err != nil {
				return nil, fmt.Errorf(
					"Error referencing subnetwork '%s' in region '%s': %s",
					subnetworkName, region, err)
			}
			subnetworkLink = subnetwork.SelfLink
		}

		// Build the networkInterface
		var iface compute.NetworkInterface
		iface.Network = networkLink
		iface.Subnetwork = subnetworkLink

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
	return networkInterfaces, nil
}

func resourceComputeInstanceTemplateCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	instanceProperties := &compute.InstanceProperties{}

	instanceProperties.CanIpForward = d.Get("can_ip_forward").(bool)
	instanceProperties.Description = d.Get("instance_description").(string)
	instanceProperties.MachineType = d.Get("machine_type").(string)
	disks, err := buildDisks(d, meta)
	if err != nil {
		return err
	}
	instanceProperties.Disks = disks
	metadata, err := resourceInstanceMetadata(d)
	if err != nil {
		return err
	}
	instanceProperties.Metadata = metadata
	networks, err := buildNetworks(d, meta)
	if err != nil {
		return err
	}
	instanceProperties.NetworkInterfaces = networks

	instanceProperties.Scheduling = &compute.Scheduling{}
	instanceProperties.Scheduling.OnHostMaintenance = "MIGRATE"

	if v, ok := d.GetOk("automatic_restart"); ok {
		instanceProperties.Scheduling.AutomaticRestart = v.(bool)
	}

	if v, ok := d.GetOk("on_host_maintenance"); ok {
		instanceProperties.Scheduling.OnHostMaintenance = v.(string)
	}

	forceSendFieldsScheduling := make([]string, 0, 3)
	var hasSendMaintenance bool
	hasSendMaintenance = false
	if v, ok := d.GetOk("scheduling"); ok {
		_schedulings := v.([]interface{})
		if len(_schedulings) > 1 {
			return fmt.Errorf("Error, at most one `scheduling` block can be defined")
		}
		_scheduling := _schedulings[0].(map[string]interface{})

		if vp, okp := _scheduling["automatic_restart"]; okp {
			instanceProperties.Scheduling.AutomaticRestart = vp.(bool)
			forceSendFieldsScheduling = append(forceSendFieldsScheduling, "AutomaticRestart")
		}

		if vp, okp := _scheduling["on_host_maintenance"]; okp {
			instanceProperties.Scheduling.OnHostMaintenance = vp.(string)
			forceSendFieldsScheduling = append(forceSendFieldsScheduling, "OnHostMaintenance")
			hasSendMaintenance = true
		}

		if vp, okp := _scheduling["preemptible"]; okp {
			instanceProperties.Scheduling.Preemptible = vp.(bool)
			forceSendFieldsScheduling = append(forceSendFieldsScheduling, "Preemptible")
			if vp.(bool) && !hasSendMaintenance {
				instanceProperties.Scheduling.OnHostMaintenance = "TERMINATE"
				forceSendFieldsScheduling = append(forceSendFieldsScheduling, "OnHostMaintenance")
			}
		}
	}
	instanceProperties.Scheduling.ForceSendFields = forceSendFieldsScheduling

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
		project, &instanceTemplate).Do()
	if err != nil {
		return fmt.Errorf("Error creating instance: %s", err)
	}

	// Store the ID now
	d.SetId(instanceTemplate.Name)

	err = computeOperationWaitGlobal(config, op, "Creating Instance Template")
	if err != nil {
		return err
	}

	return resourceComputeInstanceTemplateRead(d, meta)
}

func resourceComputeInstanceTemplateRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	instanceTemplate, err := config.clientCompute.InstanceTemplates.Get(
		project, d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing Instance Template %q because it's gone", d.Get("name").(string))
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

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	op, err := config.clientCompute.InstanceTemplates.Delete(
		project, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting instance template: %s", err)
	}

	err = computeOperationWaitGlobal(config, op, "Deleting Instance Template")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
