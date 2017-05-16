package google

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
)

func resourceComputeInstanceTemplate() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeInstanceTemplateCreate,
		Read:   resourceComputeInstanceTemplateRead,
		Delete: resourceComputeInstanceTemplateDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					// https://cloud.google.com/compute/docs/reference/latest/instanceTemplates#resource
					value := v.(string)
					if len(value) > 63 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 63 characters", k))
					}
					return
				},
			},

			"name_prefix": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					// https://cloud.google.com/compute/docs/reference/latest/instanceTemplates#resource
					// uuid is 26 characters, limit the prefix to 37.
					value := v.(string)
					if len(value) > 37 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 37 characters, name is limited to 63", k))
					}
					return
				},
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
							Computed: true,
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
							Computed: true,
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
							Computed: true,
						},

						"mode": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Computed: true,
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
							Computed: true,
						},
					},
				},
			},

			"machine_type": &schema.Schema{
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

			"metadata_startup_script": &schema.Schema{
				Type:     schema.TypeString,
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
							Computed: true,
						},

						"network_ip": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"subnetwork": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"subnetwork_project": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Computed: true,
						},

						"access_config": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
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
				Computed: true,
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"scheduling": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"preemptible": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
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
							Computed: true,
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
				MaxItems: 1,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"email": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
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

		var networkName, subnetworkName, subnetworkProject string
		if v, ok := d.GetOk(prefix + ".network"); ok {
			networkName = v.(string)
		}
		if v, ok := d.GetOk(prefix + ".subnetwork"); ok {
			subnetworkName = v.(string)
		}
		if v, ok := d.GetOk(prefix + ".subnetwork_project"); ok {
			subnetworkProject = v.(string)
		}
		if networkName == "" && subnetworkName == "" {
			return nil, fmt.Errorf("network or subnetwork must be provided")
		}
		if networkName != "" && subnetworkName != "" {
			return nil, fmt.Errorf("network or subnetwork must not both be provided")
		}

		var networkLink, subnetworkLink string
		if networkName != "" {
			networkLink, err = getNetworkLink(d, config, prefix+".network")
			if err != nil {
				return nil, fmt.Errorf("Error referencing network '%s': %s",
					networkName, err)
			}

		} else {
			// lookup subnetwork link using region and subnetwork name
			region, err := getRegion(d, config)
			if err != nil {
				return nil, err
			}
			if subnetworkProject == "" {
				subnetworkProject = project
			}
			subnetwork, err := config.clientCompute.Subnetworks.Get(
				subnetworkProject, region, subnetworkName).Do()
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
		if v, ok := d.GetOk(prefix + ".network_ip"); ok {
			iface.NetworkIP = v.(string)
		}
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

	// Depreciated fields
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

		email := "default"
		if v := d.Get(prefix + ".email"); v != nil {
			email = v.(string)
		}

		serviceAccount := &compute.ServiceAccount{
			Email:  email,
			Scopes: scopes,
		}

		serviceAccounts = append(serviceAccounts, serviceAccount)
	}
	instanceProperties.ServiceAccounts = serviceAccounts

	instanceProperties.Tags = resourceInstanceTags(d)

	var itName string
	if v, ok := d.GetOk("name"); ok {
		itName = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		itName = resource.PrefixedUniqueId(v.(string))
	} else {
		itName = resource.UniqueId()
	}
	instanceTemplate := compute.InstanceTemplate{
		Description: d.Get("description").(string),
		Properties:  instanceProperties,
		Name:        itName,
	}

	op, err := config.clientCompute.InstanceTemplates.Insert(
		project, &instanceTemplate).Do()
	if err != nil {
		return fmt.Errorf("Error creating instance: %s", err)
	}

	// Store the ID now
	d.SetId(instanceTemplate.Name)

	err = computeOperationWaitGlobal(config, op, project, "Creating Instance Template")
	if err != nil {
		return err
	}

	return resourceComputeInstanceTemplateRead(d, meta)
}

func flattenDisks(disks []*compute.AttachedDisk, d *schema.ResourceData) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(disks))
	for i, disk := range disks {
		diskMap := make(map[string]interface{})
		if disk.InitializeParams != nil {
			var source_img = fmt.Sprintf("disk.%d.source_image", i)
			if d.Get(source_img) == nil || d.Get(source_img) == "" {
				sourceImageUrl := strings.Split(disk.InitializeParams.SourceImage, "/")
				diskMap["source_image"] = sourceImageUrl[len(sourceImageUrl)-1]
			} else {
				diskMap["source_image"] = d.Get(source_img)
			}
			diskMap["disk_type"] = disk.InitializeParams.DiskType
			diskMap["disk_name"] = disk.InitializeParams.DiskName
			diskMap["disk_size_gb"] = disk.InitializeParams.DiskSizeGb
		}
		diskMap["auto_delete"] = disk.AutoDelete
		diskMap["boot"] = disk.Boot
		diskMap["device_name"] = disk.DeviceName
		diskMap["interface"] = disk.Interface
		diskMap["source"] = disk.Source
		diskMap["mode"] = disk.Mode
		diskMap["type"] = disk.Type
		result = append(result, diskMap)
	}
	return result
}

func flattenNetworkInterfaces(networkInterfaces []*compute.NetworkInterface) ([]map[string]interface{}, string) {
	result := make([]map[string]interface{}, 0, len(networkInterfaces))
	region := ""
	for _, networkInterface := range networkInterfaces {
		networkInterfaceMap := make(map[string]interface{})
		if networkInterface.Network != "" {
			networkUrl := strings.Split(networkInterface.Network, "/")
			networkInterfaceMap["network"] = networkUrl[len(networkUrl)-1]
		}
		if networkInterface.NetworkIP != "" {
			networkInterfaceMap["network_ip"] = networkInterface.NetworkIP
		}
		if networkInterface.Subnetwork != "" {
			subnetworkUrl := strings.Split(networkInterface.Subnetwork, "/")
			networkInterfaceMap["subnetwork"] = subnetworkUrl[len(subnetworkUrl)-1]
			region = subnetworkUrl[len(subnetworkUrl)-3]
			networkInterfaceMap["subnetwork_project"] = subnetworkUrl[len(subnetworkUrl)-5]
		}

		if networkInterface.AccessConfigs != nil {
			accessConfigsMap := make([]map[string]interface{}, 0, len(networkInterface.AccessConfigs))
			for _, accessConfig := range networkInterface.AccessConfigs {
				accessConfigMap := make(map[string]interface{})
				accessConfigMap["nat_ip"] = accessConfig.NatIP

				accessConfigsMap = append(accessConfigsMap, accessConfigMap)
			}
			networkInterfaceMap["access_config"] = accessConfigsMap
		}
		result = append(result, networkInterfaceMap)
	}
	return result, region
}

func flattenScheduling(scheduling *compute.Scheduling) ([]map[string]interface{}, bool) {
	result := make([]map[string]interface{}, 0, 1)
	schedulingMap := make(map[string]interface{})
	schedulingMap["automatic_restart"] = scheduling.AutomaticRestart
	schedulingMap["on_host_maintenance"] = scheduling.OnHostMaintenance
	schedulingMap["preemptible"] = scheduling.Preemptible
	result = append(result, schedulingMap)
	return result, scheduling.AutomaticRestart
}

func flattenServiceAccounts(serviceAccounts []*compute.ServiceAccount) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(serviceAccounts))
	for _, serviceAccount := range serviceAccounts {
		serviceAccountMap := make(map[string]interface{})
		serviceAccountMap["email"] = serviceAccount.Email
		serviceAccountMap["scopes"] = serviceAccount.Scopes

		result = append(result, serviceAccountMap)
	}
	return result
}

func flattenMetadata(metadata *compute.Metadata) map[string]string {
	metadataMap := make(map[string]string)
	for _, item := range metadata.Items {
		metadataMap[item.Key] = *item.Value
	}
	return metadataMap
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
		return handleNotFoundError(err, d, fmt.Sprintf("Instance Template %q", d.Get("name").(string)))
	}

	// Set the metadata fingerprint if there is one.
	if instanceTemplate.Properties.Metadata != nil {
		if err = d.Set("metadata_fingerprint", instanceTemplate.Properties.Metadata.Fingerprint); err != nil {
			return fmt.Errorf("Error setting metadata_fingerprint: %s", err)
		}

		md := instanceTemplate.Properties.Metadata

		_md := flattenMetadata(md)

		if script, scriptExists := d.GetOk("metadata_startup_script"); scriptExists {
			if err = d.Set("metadata_startup_script", script); err != nil {
				return fmt.Errorf("Error setting metadata_startup_script: %s", err)
			}
			delete(_md, "startup-script")
		}
		if err = d.Set("metadata", _md); err != nil {
			return fmt.Errorf("Error setting metadata: %s", err)
		}
	}

	// Set the tags fingerprint if there is one.
	if instanceTemplate.Properties.Tags != nil {
		if err = d.Set("tags_fingerprint", instanceTemplate.Properties.Tags.Fingerprint); err != nil {
			return fmt.Errorf("Error setting tags_fingerprint: %s", err)
		}
	}
	if err = d.Set("self_link", instanceTemplate.SelfLink); err != nil {
		return fmt.Errorf("Error setting self_link: %s", err)
	}
	if err = d.Set("name", instanceTemplate.Name); err != nil {
		return fmt.Errorf("Error setting name: %s", err)
	}
	if instanceTemplate.Properties.Disks != nil {
		if err = d.Set("disk", flattenDisks(instanceTemplate.Properties.Disks, d)); err != nil {
			return fmt.Errorf("Error setting disk: %s", err)
		}
	}
	if err = d.Set("description", instanceTemplate.Description); err != nil {
		return fmt.Errorf("Error setting description: %s", err)
	}
	if err = d.Set("machine_type", instanceTemplate.Properties.MachineType); err != nil {
		return fmt.Errorf("Error setting machine_type: %s", err)
	}

	if err = d.Set("can_ip_forward", instanceTemplate.Properties.CanIpForward); err != nil {
		return fmt.Errorf("Error setting can_ip_forward: %s", err)
	}

	if err = d.Set("instance_description", instanceTemplate.Properties.Description); err != nil {
		return fmt.Errorf("Error setting instance_description: %s", err)
	}
	if err = d.Set("project", project); err != nil {
		return fmt.Errorf("Error setting project: %s", err)
	}
	if instanceTemplate.Properties.NetworkInterfaces != nil {
		networkInterfaces, region := flattenNetworkInterfaces(instanceTemplate.Properties.NetworkInterfaces)
		if err = d.Set("network_interface", networkInterfaces); err != nil {
			return fmt.Errorf("Error setting network_interface: %s", err)
		}
		// region is where to look up the subnetwork if there is one attached to the instance template
		if region != "" {
			if err = d.Set("region", region); err != nil {
				return fmt.Errorf("Error setting region: %s", err)
			}
		}
	}
	if instanceTemplate.Properties.Scheduling != nil {
		scheduling, autoRestart := flattenScheduling(instanceTemplate.Properties.Scheduling)
		if err = d.Set("scheduling", scheduling); err != nil {
			return fmt.Errorf("Error setting scheduling: %s", err)
		}
		if err = d.Set("automatic_restart", autoRestart); err != nil {
			return fmt.Errorf("Error setting automatic_restart: %s", err)
		}
	}
	if instanceTemplate.Properties.Tags != nil {
		if err = d.Set("tags", instanceTemplate.Properties.Tags.Items); err != nil {
			return fmt.Errorf("Error setting tags: %s", err)
		}
	}
	if instanceTemplate.Properties.ServiceAccounts != nil {
		if err = d.Set("service_account", flattenServiceAccounts(instanceTemplate.Properties.ServiceAccounts)); err != nil {
			return fmt.Errorf("Error setting service_account: %s", err)
		}
	}
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

	err = computeOperationWaitGlobal(config, op, project, "Deleting Instance Template")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
