package openstack

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/availabilityzones"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/bootfromvolume"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/floatingips"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/schedulerhints"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/startstop"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/tenantnetworks"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/volumeattach"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceComputeInstanceV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeInstanceV2Create,
		Read:   resourceComputeInstanceV2Read,
		Update: resourceComputeInstanceV2Update,
		Delete: resourceComputeInstanceV2Delete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_REGION_NAME", ""),
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"image_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"image_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"flavor_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
				Computed:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_FLAVOR_ID", nil),
			},
			"flavor_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
				Computed:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_FLAVOR_NAME", nil),
			},
			"floating_ip": &schema.Schema{
				Type:       schema.TypeString,
				Optional:   true,
				ForceNew:   false,
				Deprecated: "Use the openstack_compute_floatingip_associate_v2 resource instead",
			},
			"user_data": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				// just stash the hash for state & diff comparisons
				StateFunc: func(v interface{}) string {
					switch v.(type) {
					case string:
						hash := sha1.Sum([]byte(v.(string)))
						return hex.EncodeToString(hash[:])
					default:
						return ""
					}
				},
			},
			"security_groups": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"availability_zone": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"network": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"uuid": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Computed: true,
						},
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Computed: true,
						},
						"port": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Computed: true,
						},
						"fixed_ip_v4": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Computed: true,
						},
						"fixed_ip_v6": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Computed: true,
						},
						"floating_ip": &schema.Schema{
							Type:       schema.TypeString,
							Optional:   true,
							Computed:   true,
							Deprecated: "Use the openstack_compute_floatingip_associate_v2 resource instead",
						},
						"mac": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"access_network": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},
			"metadata": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: false,
			},
			"config_drive": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"admin_pass": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"access_ip_v4": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ForceNew: false,
			},
			"access_ip_v6": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ForceNew: false,
			},
			"key_pair": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"block_device": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"source_type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"uuid": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"volume_size": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"destination_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"boot_index": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"delete_on_termination": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
							ForceNew: true,
						},
						"guest_format": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},
			"volume": &schema.Schema{
				Type:       schema.TypeSet,
				Optional:   true,
				Deprecated: "Use block_device or openstack_compute_volume_attach_v2 instead",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"volume_id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"device": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
				Set: resourceComputeVolumeAttachmentHash,
			},
			"scheduler_hints": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"group": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"different_host": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"same_host": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"query": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"target_cell": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"build_near_host_ip": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
				Set: resourceComputeSchedulerHintsHash,
			},
			"personality": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"file": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"content": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: resourceComputeInstancePersonalityHash,
			},
			"stop_before_destroy": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"force_delete": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"all_metadata": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
			},
		},
	}
}

func resourceComputeInstanceV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	var createOpts servers.CreateOptsBuilder

	// Determines the Image ID using the following rules:
	// If a bootable block_device was specified, ignore the image altogether.
	// If an image_id was specified, use it.
	// If an image_name was specified, look up the image ID, report if error.
	imageId, err := getImageIDFromConfig(computeClient, d)
	if err != nil {
		return err
	}

	flavorId, err := getFlavorID(computeClient, d)
	if err != nil {
		return err
	}

	// determine if block_device configuration is correct
	// this includes valid combinations and required attributes
	if err := checkBlockDeviceConfig(d); err != nil {
		return err
	}

	// check if floating IP configuration is correct
	if err := checkInstanceFloatingIPs(d); err != nil {
		return err
	}

	// Build a list of networks with the information given upon creation.
	// Error out if an invalid network configuration was used.
	networkDetails, err := getInstanceNetworks(computeClient, d)
	if err != nil {
		return err
	}

	networks := make([]servers.Network, len(networkDetails))
	for i, net := range networkDetails {
		networks[i] = servers.Network{
			UUID:    net["uuid"].(string),
			Port:    net["port"].(string),
			FixedIP: net["fixed_ip_v4"].(string),
		}
	}

	configDrive := d.Get("config_drive").(bool)

	createOpts = &servers.CreateOpts{
		Name:             d.Get("name").(string),
		ImageRef:         imageId,
		FlavorRef:        flavorId,
		SecurityGroups:   resourceInstanceSecGroupsV2(d),
		AvailabilityZone: d.Get("availability_zone").(string),
		Networks:         networks,
		Metadata:         resourceInstanceMetadataV2(d),
		ConfigDrive:      &configDrive,
		AdminPass:        d.Get("admin_pass").(string),
		UserData:         []byte(d.Get("user_data").(string)),
		Personality:      resourceInstancePersonalityV2(d),
	}

	if keyName, ok := d.Get("key_pair").(string); ok && keyName != "" {
		createOpts = &keypairs.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			KeyName:           keyName,
		}
	}

	if vL, ok := d.GetOk("block_device"); ok {
		blockDevices, err := resourceInstanceBlockDevicesV2(d, vL.([]interface{}))
		if err != nil {
			return err
		}

		createOpts = &bootfromvolume.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			BlockDevice:       blockDevices,
		}
	}

	schedulerHintsRaw := d.Get("scheduler_hints").(*schema.Set).List()
	if len(schedulerHintsRaw) > 0 {
		log.Printf("[DEBUG] schedulerhints: %+v", schedulerHintsRaw)
		schedulerHints := resourceInstanceSchedulerHintsV2(d, schedulerHintsRaw[0].(map[string]interface{}))
		createOpts = &schedulerhints.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			SchedulerHints:    schedulerHints,
		}
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)

	// If a block_device is used, use the bootfromvolume.Create function as it allows an empty ImageRef.
	// Otherwise, use the normal servers.Create function.
	var server *servers.Server
	if _, ok := d.GetOk("block_device"); ok {
		server, err = bootfromvolume.Create(computeClient, createOpts).Extract()
	} else {
		server, err = servers.Create(computeClient, createOpts).Extract()
	}

	if err != nil {
		return fmt.Errorf("Error creating OpenStack server: %s", err)
	}
	log.Printf("[INFO] Instance ID: %s", server.ID)

	// Store the ID now
	d.SetId(server.ID)

	// Wait for the instance to become running so we can get some attributes
	// that aren't available until later.
	log.Printf(
		"[DEBUG] Waiting for instance (%s) to become running",
		server.ID)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"BUILD"},
		Target:     []string{"ACTIVE"},
		Refresh:    ServerV2StateRefreshFunc(computeClient, server.ID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for instance (%s) to become ready: %s",
			server.ID, err)
	}

	// Now that the instance has been created, we need to do an early read on the
	// networks in order to associate floating IPs
	_, err = getInstanceNetworksAndAddresses(computeClient, d)

	// If floating IPs were specified, associate them after the instance has launched.
	err = associateFloatingIPsToInstance(computeClient, d)
	if err != nil {
		return err
	}

	// if volumes were specified, attach them after the instance has launched.
	if v, ok := d.GetOk("volume"); ok {
		vols := v.(*schema.Set).List()
		if blockClient, err := config.blockStorageV1Client(GetRegion(d)); err != nil {
			return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
		} else {
			if err := attachVolumesToInstance(computeClient, blockClient, d.Id(), vols); err != nil {
				return err
			}
		}
	}

	return resourceComputeInstanceV2Read(d, meta)
}

func resourceComputeInstanceV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	server, err := servers.Get(computeClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "server")
	}

	log.Printf("[DEBUG] Retrieved Server %s: %+v", d.Id(), server)

	d.Set("name", server.Name)

	// Get the instance network and address information
	networks, err := getInstanceNetworksAndAddresses(computeClient, d)
	if err != nil {
		return err
	}

	// Determine the best IPv4 and IPv6 addresses to access the instance with
	hostv4, hostv6 := getInstanceAccessAddresses(d, networks)

	if server.AccessIPv4 != "" && hostv4 == "" {
		hostv4 = server.AccessIPv4
	}

	if server.AccessIPv6 != "" && hostv6 == "" {
		hostv6 = server.AccessIPv6
	}

	d.Set("network", networks)
	d.Set("access_ip_v4", hostv4)
	d.Set("access_ip_v6", hostv6)

	// Determine the best IP address to use for SSH connectivity.
	// Prefer IPv4 over IPv6.
	preferredSSHAddress := ""
	if hostv4 != "" {
		preferredSSHAddress = hostv4
	} else if hostv6 != "" {
		preferredSSHAddress = hostv6
	}

	if preferredSSHAddress != "" {
		// Initialize the connection info
		d.SetConnInfo(map[string]string{
			"type": "ssh",
			"host": preferredSSHAddress,
		})
	}

	d.Set("all_metadata", server.Metadata)

	secGrpNames := []string{}
	for _, sg := range server.SecurityGroups {
		secGrpNames = append(secGrpNames, sg["name"].(string))
	}
	d.Set("security_groups", secGrpNames)

	flavorId, ok := server.Flavor["id"].(string)
	if !ok {
		return fmt.Errorf("Error setting OpenStack server's flavor: %v", server.Flavor)
	}
	d.Set("flavor_id", flavorId)

	flavor, err := flavors.Get(computeClient, flavorId).Extract()
	if err != nil {
		return err
	}
	d.Set("flavor_name", flavor.Name)

	// Set the instance's image information appropriately
	if err := setImageInformation(computeClient, server, d); err != nil {
		return err
	}

	// volume attachments
	if err := getVolumeAttachments(computeClient, d); err != nil {
		return err
	}

	// Build a custom struct for the availability zone extension
	var serverWithAZ struct {
		servers.Server
		availabilityzones.ServerExt
	}

	// Do another Get so the above work is not disturbed.
	err = servers.Get(computeClient, d.Id()).ExtractInto(&serverWithAZ)
	if err != nil {
		return CheckDeleted(d, err, "server")
	}

	// Set the availability zone
	d.Set("availability_zone", serverWithAZ.AvailabilityZone)

	return nil
}

func resourceComputeInstanceV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	var updateOpts servers.UpdateOpts
	if d.HasChange("name") {
		updateOpts.Name = d.Get("name").(string)
	}

	if updateOpts != (servers.UpdateOpts{}) {
		_, err := servers.Update(computeClient, d.Id(), updateOpts).Extract()
		if err != nil {
			return fmt.Errorf("Error updating OpenStack server: %s", err)
		}
	}

	if d.HasChange("metadata") {
		oldMetadata, newMetadata := d.GetChange("metadata")
		var metadataToDelete []string

		// Determine if any metadata keys were removed from the configuration.
		// Then request those keys to be deleted.
		for oldKey, _ := range oldMetadata.(map[string]interface{}) {
			var found bool
			for newKey, _ := range newMetadata.(map[string]interface{}) {
				if oldKey == newKey {
					found = true
				}
			}

			if !found {
				metadataToDelete = append(metadataToDelete, oldKey)
			}
		}

		for _, key := range metadataToDelete {
			err := servers.DeleteMetadatum(computeClient, d.Id(), key).ExtractErr()
			if err != nil {
				return fmt.Errorf("Error deleting metadata (%s) from server (%s): %s", key, d.Id(), err)
			}
		}

		// Update existing metadata and add any new metadata.
		metadataOpts := make(servers.MetadataOpts)
		for k, v := range newMetadata.(map[string]interface{}) {
			metadataOpts[k] = v.(string)
		}

		_, err := servers.UpdateMetadata(computeClient, d.Id(), metadataOpts).Extract()
		if err != nil {
			return fmt.Errorf("Error updating OpenStack server (%s) metadata: %s", d.Id(), err)
		}
	}

	if d.HasChange("security_groups") {
		oldSGRaw, newSGRaw := d.GetChange("security_groups")
		oldSGSet := oldSGRaw.(*schema.Set)
		newSGSet := newSGRaw.(*schema.Set)
		secgroupsToAdd := newSGSet.Difference(oldSGSet)
		secgroupsToRemove := oldSGSet.Difference(newSGSet)

		log.Printf("[DEBUG] Security groups to add: %v", secgroupsToAdd)

		log.Printf("[DEBUG] Security groups to remove: %v", secgroupsToRemove)

		for _, g := range secgroupsToRemove.List() {
			err := secgroups.RemoveServer(computeClient, d.Id(), g.(string)).ExtractErr()
			if err != nil && err.Error() != "EOF" {
				if _, ok := err.(gophercloud.ErrDefault404); ok {
					continue
				}

				return fmt.Errorf("Error removing security group (%s) from OpenStack server (%s): %s", g, d.Id(), err)
			} else {
				log.Printf("[DEBUG] Removed security group (%s) from instance (%s)", g, d.Id())
			}
		}

		for _, g := range secgroupsToAdd.List() {
			err := secgroups.AddServer(computeClient, d.Id(), g.(string)).ExtractErr()
			if err != nil && err.Error() != "EOF" {
				return fmt.Errorf("Error adding security group (%s) to OpenStack server (%s): %s", g, d.Id(), err)
			}
			log.Printf("[DEBUG] Added security group (%s) to instance (%s)", g, d.Id())
		}
	}

	if d.HasChange("admin_pass") {
		if newPwd, ok := d.Get("admin_pass").(string); ok {
			err := servers.ChangeAdminPassword(computeClient, d.Id(), newPwd).ExtractErr()
			if err != nil {
				return fmt.Errorf("Error changing admin password of OpenStack server (%s): %s", d.Id(), err)
			}
		}
	}

	if d.HasChange("floating_ip") {
		oldFIP, newFIP := d.GetChange("floating_ip")
		log.Printf("[DEBUG] Old Floating IP: %v", oldFIP)
		log.Printf("[DEBUG] New Floating IP: %v", newFIP)
		if oldFIP.(string) != "" {
			log.Printf("[DEBUG] Attempting to disassociate %s from %s", oldFIP, d.Id())
			if err := disassociateFloatingIPFromInstance(computeClient, oldFIP.(string), d.Id(), ""); err != nil {
				return fmt.Errorf("Error disassociating Floating IP during update: %s", err)
			}
		}

		if newFIP.(string) != "" {
			log.Printf("[DEBUG] Attempting to associate %s to %s", newFIP, d.Id())
			if err := associateFloatingIPToInstance(computeClient, newFIP.(string), d.Id(), ""); err != nil {
				return fmt.Errorf("Error associating Floating IP during update: %s", err)
			}
		}
	}

	if d.HasChange("network") {
		oldNetworks, newNetworks := d.GetChange("network")
		oldNetworkList := oldNetworks.([]interface{})
		newNetworkList := newNetworks.([]interface{})
		for i, oldNet := range oldNetworkList {
			var oldFIP, newFIP string
			var oldFixedIP, newFixedIP string

			if oldNetRaw, ok := oldNet.(map[string]interface{}); ok {
				oldFIP = oldNetRaw["floating_ip"].(string)
				oldFixedIP = oldNetRaw["fixed_ip_v4"].(string)
			}

			if len(newNetworkList) > i {
				if newNetRaw, ok := newNetworkList[i].(map[string]interface{}); ok {
					newFIP = newNetRaw["floating_ip"].(string)
					newFixedIP = newNetRaw["fixed_ip_v4"].(string)
				}
			}

			// Only changes to the floating IP are supported
			if oldFIP != "" && oldFIP != newFIP {
				log.Printf("[DEBUG] Attempting to disassociate %s from %s", oldFIP, d.Id())
				if err := disassociateFloatingIPFromInstance(computeClient, oldFIP, d.Id(), oldFixedIP); err != nil {
					return fmt.Errorf("Error disassociating Floating IP during update: %s", err)
				}
			}

			if newFIP != "" && oldFIP != newFIP {
				log.Printf("[DEBUG] Attempting to associate %s to %s", newFIP, d.Id())
				if err := associateFloatingIPToInstance(computeClient, newFIP, d.Id(), newFixedIP); err != nil {
					return fmt.Errorf("Error associating Floating IP during update: %s", err)
				}
			}
		}
	}

	if d.HasChange("volume") {
		// old attachments and new attachments
		oldAttachments, newAttachments := d.GetChange("volume")
		// for each old attachment, detach the volume
		oldAttachmentSet := oldAttachments.(*schema.Set).List()

		log.Printf("[DEBUG] Attempting to detach the following volumes: %#v", oldAttachmentSet)
		if blockClient, err := config.blockStorageV1Client(GetRegion(d)); err != nil {
			return err
		} else {
			if err := detachVolumesFromInstance(computeClient, blockClient, d.Id(), oldAttachmentSet); err != nil {
				return err
			}
		}

		// for each new attachment, attach the volume
		newAttachmentSet := newAttachments.(*schema.Set).List()
		if blockClient, err := config.blockStorageV1Client(GetRegion(d)); err != nil {
			return err
		} else {
			if err := attachVolumesToInstance(computeClient, blockClient, d.Id(), newAttachmentSet); err != nil {
				return err
			}
		}

		d.SetPartial("volume")
	}

	if d.HasChange("flavor_id") || d.HasChange("flavor_name") {
		var newFlavorId string
		var err error
		if d.HasChange("flavor_id") {
			newFlavorId = d.Get("flavor_id").(string)
		} else {
			newFlavorName := d.Get("flavor_name").(string)
			newFlavorId, err = flavors.IDFromName(computeClient, newFlavorName)
			if err != nil {
				return err
			}
		}

		resizeOpts := &servers.ResizeOpts{
			FlavorRef: newFlavorId,
		}
		log.Printf("[DEBUG] Resize configuration: %#v", resizeOpts)
		err = servers.Resize(computeClient, d.Id(), resizeOpts).ExtractErr()
		if err != nil {
			return fmt.Errorf("Error resizing OpenStack server: %s", err)
		}

		// Wait for the instance to finish resizing.
		log.Printf("[DEBUG] Waiting for instance (%s) to finish resizing", d.Id())

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"RESIZE"},
			Target:     []string{"VERIFY_RESIZE"},
			Refresh:    ServerV2StateRefreshFunc(computeClient, d.Id()),
			Timeout:    d.Timeout(schema.TimeoutUpdate),
			Delay:      10 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf("Error waiting for instance (%s) to resize: %s", d.Id(), err)
		}

		// Confirm resize.
		log.Printf("[DEBUG] Confirming resize")
		err = servers.ConfirmResize(computeClient, d.Id()).ExtractErr()
		if err != nil {
			return fmt.Errorf("Error confirming resize of OpenStack server: %s", err)
		}

		stateConf = &resource.StateChangeConf{
			Pending:    []string{"VERIFY_RESIZE"},
			Target:     []string{"ACTIVE"},
			Refresh:    ServerV2StateRefreshFunc(computeClient, d.Id()),
			Timeout:    d.Timeout(schema.TimeoutUpdate),
			Delay:      10 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf("Error waiting for instance (%s) to confirm resize: %s", d.Id(), err)
		}
	}

	return resourceComputeInstanceV2Read(d, meta)
}

func resourceComputeInstanceV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	// Make sure all volumes are detached before deleting
	volumes := d.Get("volume")
	if volumeSet, ok := volumes.(*schema.Set); ok {
		volumeList := volumeSet.List()
		if len(volumeList) > 0 {
			log.Printf("[DEBUG] Attempting to detach the following volumes: %#v", volumeList)
			if blockClient, err := config.blockStorageV1Client(GetRegion(d)); err != nil {
				return err
			} else {
				if err := detachVolumesFromInstance(computeClient, blockClient, d.Id(), volumeList); err != nil {
					return err
				}
			}
		}
	}

	if d.Get("stop_before_destroy").(bool) {
		err = startstop.Stop(computeClient, d.Id()).ExtractErr()
		if err != nil {
			log.Printf("[WARN] Error stopping OpenStack instance: %s", err)
		} else {
			stopStateConf := &resource.StateChangeConf{
				Pending:    []string{"ACTIVE"},
				Target:     []string{"SHUTOFF"},
				Refresh:    ServerV2StateRefreshFunc(computeClient, d.Id()),
				Timeout:    3 * time.Minute,
				Delay:      10 * time.Second,
				MinTimeout: 3 * time.Second,
			}
			log.Printf("[DEBUG] Waiting for instance (%s) to stop", d.Id())
			_, err = stopStateConf.WaitForState()
			if err != nil {
				log.Printf("[WARN] Error waiting for instance (%s) to stop: %s, proceeding to delete", d.Id(), err)
			}
		}
	}

	if d.Get("force_delete").(bool) {
		log.Printf("[DEBUG] Force deleting OpenStack Instance %s", d.Id())
		err = servers.ForceDelete(computeClient, d.Id()).ExtractErr()
		if err != nil {
			return fmt.Errorf("Error deleting OpenStack server: %s", err)
		}
	} else {
		log.Printf("[DEBUG] Deleting OpenStack Instance %s", d.Id())
		err = servers.Delete(computeClient, d.Id()).ExtractErr()
		if err != nil {
			return fmt.Errorf("Error deleting OpenStack server: %s", err)
		}
	}

	// Wait for the instance to delete before moving on.
	log.Printf("[DEBUG] Waiting for instance (%s) to delete", d.Id())

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE", "SHUTOFF"},
		Target:     []string{"DELETED", "SOFT_DELETED"},
		Refresh:    ServerV2StateRefreshFunc(computeClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for instance (%s) to delete: %s",
			d.Id(), err)
	}

	d.SetId("")
	return nil
}

// ServerV2StateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// an OpenStack instance.
func ServerV2StateRefreshFunc(client *gophercloud.ServiceClient, instanceID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		s, err := servers.Get(client, instanceID).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				return s, "DELETED", nil
			}
			return nil, "", err
		}

		return s, s.Status, nil
	}
}

func resourceInstanceSecGroupsV2(d *schema.ResourceData) []string {
	rawSecGroups := d.Get("security_groups").(*schema.Set).List()
	secgroups := make([]string, len(rawSecGroups))
	for i, raw := range rawSecGroups {
		secgroups[i] = raw.(string)
	}
	return secgroups
}

// getInstanceNetworks collects instance network information from different sources
// and aggregates it all together.
func getInstanceNetworksAndAddresses(computeClient *gophercloud.ServiceClient, d *schema.ResourceData) ([]map[string]interface{}, error) {
	server, err := servers.Get(computeClient, d.Id()).Extract()

	if err != nil {
		return nil, CheckDeleted(d, err, "server")
	}

	networkDetails, err := getInstanceNetworks(computeClient, d)
	addresses := getInstanceAddresses(server.Addresses)
	if err != nil {
		return nil, err
	}

	// if there are no networkDetails, make networks at least a length of 1
	networkLength := 1
	if len(networkDetails) > 0 {
		networkLength = len(networkDetails)
	}
	networks := make([]map[string]interface{}, networkLength)

	// Loop through all networks and addresses,
	// merge relevant address details.
	if len(networkDetails) == 0 {
		for netName, n := range addresses {
			networks[0] = map[string]interface{}{
				"name":        netName,
				"fixed_ip_v4": n["fixed_ip_v4"],
				"fixed_ip_v6": n["fixed_ip_v6"],
				"floating_ip": n["floating_ip"],
				"mac":         n["mac"],
			}
		}
	} else {
		for i, net := range networkDetails {
			n := addresses[net["name"].(string)]

			networks[i] = map[string]interface{}{
				"uuid":           networkDetails[i]["uuid"],
				"name":           networkDetails[i]["name"],
				"port":           networkDetails[i]["port"],
				"fixed_ip_v4":    n["fixed_ip_v4"],
				"fixed_ip_v6":    n["fixed_ip_v6"],
				"floating_ip":    n["floating_ip"],
				"mac":            n["mac"],
				"access_network": networkDetails[i]["access_network"],
			}
		}
	}

	log.Printf("[DEBUG] networks: %+v", networks)

	return networks, nil
}

func getInstanceNetworks(computeClient *gophercloud.ServiceClient, d *schema.ResourceData) ([]map[string]interface{}, error) {
	rawNetworks := d.Get("network").([]interface{})
	newNetworks := make([]map[string]interface{}, 0, len(rawNetworks))
	var tenantnet tenantnetworks.Network

	tenantNetworkExt := true
	for _, raw := range rawNetworks {
		// Not sure what causes this, but it is a possibility (see GH-2323).
		// Since we call this function to reconcile what we'll save in the
		// state anyways, we just ignore it.
		if raw == nil {
			continue
		}

		rawMap := raw.(map[string]interface{})

		// Both a floating IP and a port cannot be specified
		if fip, ok := rawMap["floating_ip"].(string); ok {
			if port, ok := rawMap["port"].(string); ok {
				if fip != "" && port != "" {
					return nil, fmt.Errorf("Only one of a floating IP or port may be specified per network.")
				}
			}
		}

		allPages, err := tenantnetworks.List(computeClient).AllPages()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				log.Printf("[DEBUG] os-tenant-networks disabled")
				tenantNetworkExt = false
			}

			log.Printf("[DEBUG] Err looks like: %+v", err)
			if errCode, ok := err.(gophercloud.ErrUnexpectedResponseCode); ok {
				if errCode.Actual == 403 {
					log.Printf("[DEBUG] os-tenant-networks disabled.")
					tenantNetworkExt = false
				} else {
					log.Printf("[DEBUG] unexpected os-tenant-networks error: %s", err)
					tenantNetworkExt = false
				}
			}
		}

		// In some cases, a call to os-tenant-networks might work,
		// but the response is invalid. Catch this during extraction.
		networkList := []tenantnetworks.Network{}
		if tenantNetworkExt {
			networkList, err = tenantnetworks.ExtractNetworks(allPages)
			if err != nil {
				log.Printf("[DEBUG] error extracting os-tenant-networks results: %s", err)
				tenantNetworkExt = false
			}
		}

		networkID := ""
		networkName := ""
		if tenantNetworkExt {
			for _, network := range networkList {
				if network.Name == rawMap["name"] {
					tenantnet = network
				}
				if network.ID == rawMap["uuid"] {
					tenantnet = network
				}
			}

			networkID = tenantnet.ID
			networkName = tenantnet.Name
		} else {
			networkID = rawMap["uuid"].(string)
			networkName = rawMap["name"].(string)
		}

		newNetworks = append(newNetworks, map[string]interface{}{
			"uuid":           networkID,
			"name":           networkName,
			"port":           rawMap["port"].(string),
			"fixed_ip_v4":    rawMap["fixed_ip_v4"].(string),
			"access_network": rawMap["access_network"].(bool),
		})
	}

	log.Printf("[DEBUG] networks: %+v", newNetworks)
	return newNetworks, nil
}

func getInstanceAddresses(addresses map[string]interface{}) map[string]map[string]interface{} {
	addrs := make(map[string]map[string]interface{})
	for n, networkAddresses := range addresses {
		addrs[n] = make(map[string]interface{})
		for _, element := range networkAddresses.([]interface{}) {
			address := element.(map[string]interface{})
			if address["OS-EXT-IPS:type"] == "floating" {
				addrs[n]["floating_ip"] = address["addr"]
			} else {
				if address["version"].(float64) == 4 {
					addrs[n]["fixed_ip_v4"] = address["addr"].(string)
				} else {
					addrs[n]["fixed_ip_v6"] = fmt.Sprintf("[%s]", address["addr"].(string))
				}
			}
			if mac, ok := address["OS-EXT-IPS-MAC:mac_addr"]; ok {
				addrs[n]["mac"] = mac.(string)
			}
		}
	}

	log.Printf("[DEBUG] Addresses: %+v", addresses)

	return addrs
}

func getInstanceAccessAddresses(d *schema.ResourceData, networks []map[string]interface{}) (string, string) {
	var hostv4, hostv6 string

	// Start with a global floating IP
	floatingIP := d.Get("floating_ip").(string)
	if floatingIP != "" {
		hostv4 = floatingIP
	}

	// Loop through all networks
	// If the network has a valid floating, fixed v4, or fixed v6 address
	// and hostv4 or hostv6 is not set, set hostv4/hostv6.
	// If the network is an "access_network" overwrite hostv4/hostv6.
	for _, n := range networks {
		var accessNetwork bool

		if an, ok := n["access_network"].(bool); ok && an {
			accessNetwork = true
		}

		if fixedIPv4, ok := n["fixed_ip_v4"].(string); ok && fixedIPv4 != "" {
			if hostv4 == "" || accessNetwork {
				hostv4 = fixedIPv4
			}
		}

		if floatingIP, ok := n["floating_ip"].(string); ok && floatingIP != "" {
			if hostv4 == "" || accessNetwork {
				hostv4 = floatingIP
			}
		}

		if fixedIPv6, ok := n["fixed_ip_v6"].(string); ok && fixedIPv6 != "" {
			if hostv6 == "" || accessNetwork {
				hostv6 = fixedIPv6
			}
		}
	}

	log.Printf("[DEBUG] OpenStack Instance Network Access Addresses: %s, %s", hostv4, hostv6)

	return hostv4, hostv6
}

func checkInstanceFloatingIPs(d *schema.ResourceData) error {
	rawNetworks := d.Get("network").([]interface{})
	floatingIP := d.Get("floating_ip").(string)

	for _, raw := range rawNetworks {
		if raw == nil {
			continue
		}

		rawMap := raw.(map[string]interface{})

		// Error if a floating IP was specified both globally and in the network block.
		if floatingIP != "" && rawMap["floating_ip"] != "" {
			return fmt.Errorf("Cannot specify a floating IP both globally and in a network block.")
		}
	}
	return nil
}

func associateFloatingIPsToInstance(computeClient *gophercloud.ServiceClient, d *schema.ResourceData) error {
	floatingIP := d.Get("floating_ip").(string)
	rawNetworks := d.Get("network").([]interface{})
	instanceID := d.Id()

	if floatingIP != "" {
		if err := associateFloatingIPToInstance(computeClient, floatingIP, instanceID, ""); err != nil {
			return err
		}
	} else {
		for _, raw := range rawNetworks {
			if raw == nil {
				continue
			}

			rawMap := raw.(map[string]interface{})
			if rawMap["floating_ip"].(string) != "" {
				floatingIP := rawMap["floating_ip"].(string)
				fixedIP := rawMap["fixed_ip_v4"].(string)
				if err := associateFloatingIPToInstance(computeClient, floatingIP, instanceID, fixedIP); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func associateFloatingIPToInstance(computeClient *gophercloud.ServiceClient, floatingIP string, instanceID string, fixedIP string) error {
	associateOpts := floatingips.AssociateOpts{
		FloatingIP: floatingIP,
		FixedIP:    fixedIP,
	}

	if err := floatingips.AssociateInstance(computeClient, instanceID, associateOpts).ExtractErr(); err != nil {
		return fmt.Errorf("Error associating floating IP: %s", err)
	}

	return nil
}

func disassociateFloatingIPFromInstance(computeClient *gophercloud.ServiceClient, floatingIP string, instanceID string, fixedIP string) error {
	disassociateOpts := floatingips.DisassociateOpts{
		FloatingIP: floatingIP,
	}

	if err := floatingips.DisassociateInstance(computeClient, instanceID, disassociateOpts).ExtractErr(); err != nil {
		return fmt.Errorf("Error disassociating floating IP: %s", err)
	}

	return nil
}

func resourceInstanceMetadataV2(d *schema.ResourceData) map[string]string {
	m := make(map[string]string)
	for key, val := range d.Get("metadata").(map[string]interface{}) {
		m[key] = val.(string)
	}
	return m
}

func resourceInstanceBlockDevicesV2(d *schema.ResourceData, bds []interface{}) ([]bootfromvolume.BlockDevice, error) {
	blockDeviceOpts := make([]bootfromvolume.BlockDevice, len(bds))
	for i, bd := range bds {
		bdM := bd.(map[string]interface{})
		blockDeviceOpts[i] = bootfromvolume.BlockDevice{
			UUID:                bdM["uuid"].(string),
			VolumeSize:          bdM["volume_size"].(int),
			BootIndex:           bdM["boot_index"].(int),
			DeleteOnTermination: bdM["delete_on_termination"].(bool),
			GuestFormat:         bdM["guest_format"].(string),
		}

		sourceType := bdM["source_type"].(string)
		switch sourceType {
		case "blank":
			blockDeviceOpts[i].SourceType = bootfromvolume.SourceBlank
		case "image":
			blockDeviceOpts[i].SourceType = bootfromvolume.SourceImage
		case "snapshot":
			blockDeviceOpts[i].SourceType = bootfromvolume.SourceSnapshot
		case "volume":
			blockDeviceOpts[i].SourceType = bootfromvolume.SourceVolume
		default:
			return blockDeviceOpts, fmt.Errorf("unknown block device source type %s", sourceType)
		}

		destinationType := bdM["destination_type"].(string)
		switch destinationType {
		case "local":
			blockDeviceOpts[i].DestinationType = bootfromvolume.DestinationLocal
		case "volume":
			blockDeviceOpts[i].DestinationType = bootfromvolume.DestinationVolume
		default:
			return blockDeviceOpts, fmt.Errorf("unknown block device destination type %s", destinationType)
		}
	}

	log.Printf("[DEBUG] Block Device Options: %+v", blockDeviceOpts)
	return blockDeviceOpts, nil
}

func resourceInstanceSchedulerHintsV2(d *schema.ResourceData, schedulerHintsRaw map[string]interface{}) schedulerhints.SchedulerHints {
	differentHost := []string{}
	if len(schedulerHintsRaw["different_host"].([]interface{})) > 0 {
		for _, dh := range schedulerHintsRaw["different_host"].([]interface{}) {
			differentHost = append(differentHost, dh.(string))
		}
	}

	sameHost := []string{}
	if len(schedulerHintsRaw["same_host"].([]interface{})) > 0 {
		for _, sh := range schedulerHintsRaw["same_host"].([]interface{}) {
			sameHost = append(sameHost, sh.(string))
		}
	}

	query := make([]interface{}, len(schedulerHintsRaw["query"].([]interface{})))
	if len(schedulerHintsRaw["query"].([]interface{})) > 0 {
		for _, q := range schedulerHintsRaw["query"].([]interface{}) {
			query = append(query, q.(string))
		}
	}

	schedulerHints := schedulerhints.SchedulerHints{
		Group:           schedulerHintsRaw["group"].(string),
		DifferentHost:   differentHost,
		SameHost:        sameHost,
		Query:           query,
		TargetCell:      schedulerHintsRaw["target_cell"].(string),
		BuildNearHostIP: schedulerHintsRaw["build_near_host_ip"].(string),
	}

	return schedulerHints
}

func getImageIDFromConfig(computeClient *gophercloud.ServiceClient, d *schema.ResourceData) (string, error) {
	// If block_device was used, an Image does not need to be specified, unless an image/local
	// combination was used. This emulates normal boot behavior. Otherwise, ignore the image altogether.
	if vL, ok := d.GetOk("block_device"); ok {
		needImage := false
		for _, v := range vL.([]interface{}) {
			vM := v.(map[string]interface{})
			if vM["source_type"] == "image" && vM["destination_type"] == "local" {
				needImage = true
			}
		}
		if !needImage {
			return "", nil
		}
	}

	if imageId := d.Get("image_id").(string); imageId != "" {
		return imageId, nil
	} else {
		// try the OS_IMAGE_ID environment variable
		if v := os.Getenv("OS_IMAGE_ID"); v != "" {
			return v, nil
		}
	}

	imageName := d.Get("image_name").(string)
	if imageName == "" {
		// try the OS_IMAGE_NAME environment variable
		if v := os.Getenv("OS_IMAGE_NAME"); v != "" {
			imageName = v
		}
	}

	if imageName != "" {
		imageId, err := images.IDFromName(computeClient, imageName)
		if err != nil {
			return "", err
		}
		return imageId, nil
	}

	return "", fmt.Errorf("Neither a boot device, image ID, or image name were able to be determined.")
}

func setImageInformation(computeClient *gophercloud.ServiceClient, server *servers.Server, d *schema.ResourceData) error {
	// If block_device was used, an Image does not need to be specified, unless an image/local
	// combination was used. This emulates normal boot behavior. Otherwise, ignore the image altogether.
	if vL, ok := d.GetOk("block_device"); ok {
		needImage := false
		for _, v := range vL.([]interface{}) {
			vM := v.(map[string]interface{})
			if vM["source_type"] == "image" && vM["destination_type"] == "local" {
				needImage = true
			}
		}
		if !needImage {
			d.Set("image_id", "Attempt to boot from volume - no image supplied")
			return nil
		}
	}

	imageId := server.Image["id"].(string)
	if imageId != "" {
		d.Set("image_id", imageId)
		if image, err := images.Get(computeClient, imageId).Extract(); err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				// If the image name can't be found, set the value to "Image not found".
				// The most likely scenario is that the image no longer exists in the Image Service
				// but the instance still has a record from when it existed.
				d.Set("image_name", "Image not found")
				return nil
			}
			return err
		} else {
			d.Set("image_name", image.Name)
		}
	}

	return nil
}

func getFlavorID(client *gophercloud.ServiceClient, d *schema.ResourceData) (string, error) {
	flavorId := d.Get("flavor_id").(string)

	if flavorId != "" {
		return flavorId, nil
	}

	flavorName := d.Get("flavor_name").(string)
	return flavors.IDFromName(client, flavorName)
}

func resourceComputeVolumeAttachmentHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["volume_id"].(string)))

	return hashcode.String(buf.String())
}

func resourceComputeSchedulerHintsHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	if m["group"] != nil {
		buf.WriteString(fmt.Sprintf("%s-", m["group"].(string)))
	}

	if m["target_cell"] != nil {
		buf.WriteString(fmt.Sprintf("%s-", m["target_cell"].(string)))
	}

	if m["build_host_near_ip"] != nil {
		buf.WriteString(fmt.Sprintf("%s-", m["build_host_near_ip"].(string)))
	}

	buf.WriteString(fmt.Sprintf("%s-", m["different_host"].([]interface{})))
	buf.WriteString(fmt.Sprintf("%s-", m["same_host"].([]interface{})))
	buf.WriteString(fmt.Sprintf("%s-", m["query"].([]interface{})))

	return hashcode.String(buf.String())
}

func attachVolumesToInstance(computeClient *gophercloud.ServiceClient, blockClient *gophercloud.ServiceClient, serverId string, vols []interface{}) error {
	for _, v := range vols {
		va := v.(map[string]interface{})
		volumeId := va["volume_id"].(string)
		device := va["device"].(string)

		s := ""
		if serverId != "" {
			s = serverId
		} else if va["server_id"] != "" {
			s = va["server_id"].(string)
		} else {
			return fmt.Errorf("Unable to determine server ID to attach volume.")
		}

		vaOpts := &volumeattach.CreateOpts{
			Device:   device,
			VolumeID: volumeId,
		}

		if _, err := volumeattach.Create(computeClient, s, vaOpts).Extract(); err != nil {
			return err
		}

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"attaching", "available"},
			Target:     []string{"in-use"},
			Refresh:    VolumeV1StateRefreshFunc(blockClient, va["volume_id"].(string)),
			Timeout:    30 * time.Minute,
			Delay:      5 * time.Second,
			MinTimeout: 2 * time.Second,
		}

		if _, err := stateConf.WaitForState(); err != nil {
			return err
		}

		log.Printf("[INFO] Attached volume %s to instance %s", volumeId, serverId)
	}
	return nil
}

func detachVolumesFromInstance(computeClient *gophercloud.ServiceClient, blockClient *gophercloud.ServiceClient, serverId string, vols []interface{}) error {
	for _, v := range vols {
		va := v.(map[string]interface{})
		aId := va["id"].(string)

		log.Printf("[INFO] Attempting to detach volume %s", va["volume_id"])
		if err := volumeattach.Delete(computeClient, serverId, aId).ExtractErr(); err != nil {
			return err
		}

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"detaching", "in-use"},
			Target:     []string{"available"},
			Refresh:    VolumeV1StateRefreshFunc(blockClient, va["volume_id"].(string)),
			Timeout:    30 * time.Minute,
			Delay:      5 * time.Second,
			MinTimeout: 2 * time.Second,
		}

		if _, err := stateConf.WaitForState(); err != nil {
			return err
		}
		log.Printf("[INFO] Detached volume %s from instance %s", va["volume_id"], serverId)
	}

	return nil
}

func getVolumeAttachments(computeClient *gophercloud.ServiceClient, d *schema.ResourceData) error {
	var vols []map[string]interface{}

	allPages, err := volumeattach.List(computeClient, d.Id()).AllPages()
	if err != nil {
		if errCode, ok := err.(gophercloud.ErrUnexpectedResponseCode); ok {
			if errCode.Actual == 403 {
				log.Printf("[DEBUG] os-volume_attachments disabled.")
				return nil
			} else {
				return err
			}
		}
	}

	allVolumeAttachments, err := volumeattach.ExtractVolumeAttachments(allPages)
	if err != nil {
		return err
	}

	if v, ok := d.GetOk("volume"); ok {
		volumes := v.(*schema.Set).List()
		for _, volume := range volumes {
			if volumeMap, ok := volume.(map[string]interface{}); ok {
				if v, ok := volumeMap["volume_id"].(string); ok {
					for _, volumeAttachment := range allVolumeAttachments {
						if v == volumeAttachment.ID {
							vol := make(map[string]interface{})
							vol["id"] = volumeAttachment.ID
							vol["volume_id"] = volumeAttachment.VolumeID
							vol["device"] = volumeAttachment.Device
							vols = append(vols, vol)
						}
					}
				}
			}
		}
	}

	log.Printf("[INFO] Volume attachments: %v", vols)
	d.Set("volume", vols)

	return nil
}

func checkBlockDeviceConfig(d *schema.ResourceData) error {
	if vL, ok := d.GetOk("block_device"); ok {
		for _, v := range vL.([]interface{}) {
			vM := v.(map[string]interface{})

			if vM["source_type"] != "blank" && vM["uuid"] == "" {
				return fmt.Errorf("You must specify a uuid for %s block device types", vM["source_type"])
			}

			if vM["source_type"] == "image" && vM["destination_type"] == "volume" {
				if vM["volume_size"] == 0 {
					return fmt.Errorf("You must specify a volume_size when creating a volume from an image")
				}
			}

			if vM["source_type"] == "blank" && vM["destination_type"] == "local" {
				if vM["volume_size"] == 0 {
					return fmt.Errorf("You must specify a volume_size when creating a blank block device")
				}
			}
		}
	}

	return nil
}

func resourceComputeInstancePersonalityHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["file"].(string)))

	return hashcode.String(buf.String())
}

func resourceInstancePersonalityV2(d *schema.ResourceData) servers.Personality {
	var personalities servers.Personality

	if v := d.Get("personality"); v != nil {
		personalityList := v.(*schema.Set).List()
		if len(personalityList) > 0 {
			for _, p := range personalityList {
				rawPersonality := p.(map[string]interface{})
				file := servers.File{
					Path:     rawPersonality["file"].(string),
					Contents: []byte(rawPersonality["content"].(string)),
				}

				log.Printf("[DEBUG] OpenStack Compute Instance Personality: %+v", file)

				personalities = append(personalities, &file)
			}
		}
	}

	return personalities
}
