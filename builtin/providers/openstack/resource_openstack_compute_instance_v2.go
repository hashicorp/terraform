package openstack

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/bootfromvolume"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/floatingip"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/schedulerhints"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/tenantnetworks"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/volumeattach"
	"github.com/rackspace/gophercloud/openstack/compute/v2/flavors"
	"github.com/rackspace/gophercloud/openstack/compute/v2/images"
	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"
	"github.com/rackspace/gophercloud/pagination"
)

func resourceComputeInstanceV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeInstanceV2Create,
		Read:   resourceComputeInstanceV2Read,
		Update: resourceComputeInstanceV2Update,
		Delete: resourceComputeInstanceV2Delete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: envDefaultFuncAllowMissing("OS_REGION_NAME"),
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"image_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
				DefaultFunc: envDefaultFunc("OS_IMAGE_ID"),
			},
			"image_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
				DefaultFunc: envDefaultFunc("OS_IMAGE_NAME"),
			},
			"flavor_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
				Computed:    true,
				DefaultFunc: envDefaultFunc("OS_FLAVOR_ID"),
			},
			"flavor_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
				Computed:    true,
				DefaultFunc: envDefaultFunc("OS_FLAVOR_NAME"),
			},
			"floating_ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
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
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"availability_zone": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
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
							Computed: true,
						},
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"port": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"fixed_ip_v4": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"fixed_ip_v6": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"mac": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
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
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"uuid": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"source_type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"volume_size": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"destination_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"boot_index": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
			},
			"volume": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": &schema.Schema{
							Type:     schema.TypeString,
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
		},
	}
}

func resourceComputeInstanceV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	var createOpts servers.CreateOptsBuilder

	imageId, err := getImageID(computeClient, d)
	if err != nil {
		return err
	}

	flavorId, err := getFlavorID(computeClient, d)
	if err != nil {
		return err
	}

	networkDetails, err := resourceInstanceNetworks(computeClient, d)
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

	createOpts = &servers.CreateOpts{
		Name:             d.Get("name").(string),
		ImageRef:         imageId,
		FlavorRef:        flavorId,
		SecurityGroups:   resourceInstanceSecGroupsV2(d),
		AvailabilityZone: d.Get("availability_zone").(string),
		Networks:         networks,
		Metadata:         resourceInstanceMetadataV2(d),
		ConfigDrive:      d.Get("config_drive").(bool),
		AdminPass:        d.Get("admin_pass").(string),
		UserData:         []byte(d.Get("user_data").(string)),
	}

	if keyName, ok := d.Get("key_pair").(string); ok && keyName != "" {
		createOpts = &keypairs.CreateOptsExt{
			createOpts,
			keyName,
		}
	}

	if blockDeviceRaw, ok := d.Get("block_device").(map[string]interface{}); ok && blockDeviceRaw != nil {
		blockDevice := resourceInstanceBlockDeviceV2(d, blockDeviceRaw)
		createOpts = &bootfromvolume.CreateOptsExt{
			createOpts,
			blockDevice,
		}
	}

	schedulerHintsRaw := d.Get("scheduler_hints").(*schema.Set).List()
	if len(schedulerHintsRaw) > 0 {
		log.Printf("[DEBUG] schedulerhints: %+v", schedulerHintsRaw)
		schedulerHints := resourceInstanceSchedulerHintsV2(d, schedulerHintsRaw[0].(map[string]interface{}))
		createOpts = &schedulerhints.CreateOptsExt{
			createOpts,
			schedulerHints,
		}
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	server, err := servers.Create(computeClient, createOpts).Extract()
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
		Target:     "ACTIVE",
		Refresh:    ServerV2StateRefreshFunc(computeClient, server.ID),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for instance (%s) to become ready: %s",
			server.ID, err)
	}
	floatingIP := d.Get("floating_ip").(string)
	if floatingIP != "" {
		if err := floatingip.Associate(computeClient, server.ID, floatingIP).ExtractErr(); err != nil {
			return fmt.Errorf("Error associating floating IP: %s", err)
		}
	}

	// were volume attachments specified?
	if v := d.Get("volume"); v != nil {
		vols := v.(*schema.Set).List()
		if len(vols) > 0 {
			if blockClient, err := config.blockStorageV1Client(d.Get("region").(string)); err != nil {
				return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
			} else {
				if err := attachVolumesToInstance(computeClient, blockClient, d.Id(), vols); err != nil {
					return err
				}
			}
		}
	}

	return resourceComputeInstanceV2Read(d, meta)
}

func resourceComputeInstanceV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	server, err := servers.Get(computeClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "server")
	}

	log.Printf("[DEBUG] Retreived Server %s: %+v", d.Id(), server)

	d.Set("name", server.Name)

	// begin reading the network configuration
	d.Set("access_ip_v4", server.AccessIPv4)
	d.Set("access_ip_v6", server.AccessIPv6)
	hostv4 := server.AccessIPv4
	hostv6 := server.AccessIPv6

	networkDetails, err := resourceInstanceNetworks(computeClient, d)
	addresses := resourceInstanceAddresses(server.Addresses)
	if err != nil {
		return err
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
			if floatingIP, ok := n["floating_ip"]; ok {
				hostv4 = floatingIP.(string)
			} else {
				if hostv4 == "" && n["fixed_ip_v4"] != nil {
					hostv4 = n["fixed_ip_v4"].(string)
				}
			}

			if hostv6 == "" && n["fixed_ip_v6"] != nil {
				hostv6 = n["fixed_ip_v6"].(string)
			}

			networks[0] = map[string]interface{}{
				"name":        netName,
				"fixed_ip_v4": n["fixed_ip_v4"],
				"fixed_ip_v6": n["fixed_ip_v6"],
				"mac":         n["mac"],
			}
		}
	} else {
		for i, net := range networkDetails {
			n := addresses[net["name"].(string)]

			if floatingIP, ok := n["floating_ip"]; ok {
				hostv4 = floatingIP.(string)
			} else {
				if hostv4 == "" && n["fixed_ip_v4"] != nil {
					hostv4 = n["fixed_ip_v4"].(string)
				}
			}

			if hostv6 == "" && n["fixed_ip_v6"] != nil {
				hostv6 = n["fixed_ip_v6"].(string)
			}

			networks[i] = map[string]interface{}{
				"uuid":        networkDetails[i]["uuid"],
				"name":        networkDetails[i]["name"],
				"port":        networkDetails[i]["port"],
				"fixed_ip_v4": n["fixed_ip_v4"],
				"fixed_ip_v6": n["fixed_ip_v6"],
				"mac":         n["mac"],
			}
		}
	}

	log.Printf("[DEBUG] new networks: %+v", networks)

	d.Set("network", networks)
	d.Set("access_ip_v4", hostv4)
	d.Set("access_ip_v6", hostv6)
	log.Printf("hostv4: %s", hostv4)
	log.Printf("hostv6: %s", hostv6)

	// prefer the v6 address if no v4 address exists.
	preferredv := ""
	if hostv4 != "" {
		preferredv = hostv4
	} else if hostv6 != "" {
		preferredv = hostv6
	}

	if preferredv != "" {
		// Initialize the connection info
		d.SetConnInfo(map[string]string{
			"type": "ssh",
			"host": preferredv,
		})
	}
	// end network configuration

	d.Set("metadata", server.Metadata)

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

	imageId, ok := server.Image["id"].(string)
	if !ok {
		return fmt.Errorf("Error setting OpenStack server's image: %v", server.Image)
	}
	d.Set("image_id", imageId)

	image, err := images.Get(computeClient, imageId).Extract()
	if err != nil {
		return err
	}
	d.Set("image_name", image.Name)

	// volume attachments
	vas, err := getVolumeAttachments(computeClient, d.Id())
	if err != nil {
		return err
	}
	if len(vas) > 0 {
		attachments := make([]map[string]interface{}, len(vas))
		for i, attachment := range vas {
			attachments[i] = make(map[string]interface{})
			attachments[i]["id"] = attachment.ID
			attachments[i]["volume_id"] = attachment.VolumeID
			attachments[i]["device"] = attachment.Device
		}
		log.Printf("[INFO] Volume attachments: %v", attachments)
		d.Set("volume", attachments)
	}

	return nil
}

func resourceComputeInstanceV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	var updateOpts servers.UpdateOpts
	if d.HasChange("name") {
		updateOpts.Name = d.Get("name").(string)
	}
	if d.HasChange("access_ip_v4") {
		updateOpts.AccessIPv4 = d.Get("access_ip_v4").(string)
	}
	if d.HasChange("access_ip_v6") {
		updateOpts.AccessIPv4 = d.Get("access_ip_v6").(string)
	}

	if updateOpts != (servers.UpdateOpts{}) {
		_, err := servers.Update(computeClient, d.Id(), updateOpts).Extract()
		if err != nil {
			return fmt.Errorf("Error updating OpenStack server: %s", err)
		}
	}

	if d.HasChange("metadata") {
		var metadataOpts servers.MetadataOpts
		metadataOpts = make(servers.MetadataOpts)
		newMetadata := d.Get("metadata").(map[string]interface{})
		for k, v := range newMetadata {
			metadataOpts[k] = v.(string)
		}

		_, err := servers.UpdateMetadata(computeClient, d.Id(), metadataOpts).Extract()
		if err != nil {
			return fmt.Errorf("Error updating OpenStack server (%s) metadata: %s", d.Id(), err)
		}
	}

	if d.HasChange("security_groups") {
		oldSGRaw, newSGRaw := d.GetChange("security_groups")
		oldSGSlice, newSGSlice := oldSGRaw.([]interface{}), newSGRaw.([]interface{})
		oldSGSet := schema.NewSet(func(v interface{}) int { return hashcode.String(v.(string)) }, oldSGSlice)
		newSGSet := schema.NewSet(func(v interface{}) int { return hashcode.String(v.(string)) }, newSGSlice)
		secgroupsToAdd := newSGSet.Difference(oldSGSet)
		secgroupsToRemove := oldSGSet.Difference(newSGSet)

		log.Printf("[DEBUG] Security groups to add: %v", secgroupsToAdd)

		log.Printf("[DEBUG] Security groups to remove: %v", secgroupsToRemove)

		for _, g := range secgroupsToRemove.List() {
			err := secgroups.RemoveServerFromGroup(computeClient, d.Id(), g.(string)).ExtractErr()
			if err != nil {
				errCode, ok := err.(*gophercloud.UnexpectedResponseCodeError)
				if !ok {
					return fmt.Errorf("Error removing security group from OpenStack server (%s): %s", d.Id(), err)
				}
				if errCode.Actual == 404 {
					continue
				} else {
					return fmt.Errorf("Error removing security group from OpenStack server (%s): %s", d.Id(), err)
				}
			} else {
				log.Printf("[DEBUG] Removed security group (%s) from instance (%s)", g.(string), d.Id())
			}
		}
		for _, g := range secgroupsToAdd.List() {
			err := secgroups.AddServerToGroup(computeClient, d.Id(), g.(string)).ExtractErr()
			if err != nil {
				return fmt.Errorf("Error adding security group to OpenStack server (%s): %s", d.Id(), err)
			}
			log.Printf("[DEBUG] Added security group (%s) to instance (%s)", g.(string), d.Id())
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
			log.Printf("[DEBUG] Attemping to disassociate %s from %s", oldFIP, d.Id())
			if err := floatingip.Disassociate(computeClient, d.Id(), oldFIP.(string)).ExtractErr(); err != nil {
				return fmt.Errorf("Error disassociating Floating IP during update: %s", err)
			}
		}

		if newFIP.(string) != "" {
			log.Printf("[DEBUG] Attemping to associate %s to %s", newFIP, d.Id())
			if err := floatingip.Associate(computeClient, d.Id(), newFIP.(string)).ExtractErr(); err != nil {
				return fmt.Errorf("Error associating Floating IP during update: %s", err)
			}
		}
	}

	if d.HasChange("volume") {
		// old attachments and new attachments
		oldAttachments, newAttachments := d.GetChange("volume")

		// for each old attachment, detach the volume
		oldAttachmentSet := oldAttachments.(*schema.Set).List()
		if len(oldAttachmentSet) > 0 {
			if blockClient, err := config.blockStorageV1Client(d.Get("region").(string)); err != nil {
				return err
			} else {
				if err := detachVolumesFromInstance(computeClient, blockClient, d.Id(), oldAttachmentSet); err != nil {
					return err
				}
			}
		}

		// for each new attachment, attach the volume
		newAttachmentSet := newAttachments.(*schema.Set).List()
		if len(newAttachmentSet) > 0 {
			if blockClient, err := config.blockStorageV1Client(d.Get("region").(string)); err != nil {
				return err
			} else {
				if err := attachVolumesToInstance(computeClient, blockClient, d.Id(), newAttachmentSet); err != nil {
					return err
				}
			}
		}

		d.SetPartial("volume")
	}

	if d.HasChange("flavor_id") || d.HasChange("flavor_name") {
		flavorId, err := getFlavorID(computeClient, d)
		if err != nil {
			return err
		}
		resizeOpts := &servers.ResizeOpts{
			FlavorRef: flavorId,
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
			Target:     "VERIFY_RESIZE",
			Refresh:    ServerV2StateRefreshFunc(computeClient, d.Id()),
			Timeout:    3 * time.Minute,
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
			Target:     "ACTIVE",
			Refresh:    ServerV2StateRefreshFunc(computeClient, d.Id()),
			Timeout:    3 * time.Minute,
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
	computeClient, err := config.computeV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	err = servers.Delete(computeClient, d.Id()).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack server: %s", err)
	}

	// Wait for the instance to delete before moving on.
	log.Printf("[DEBUG] Waiting for instance (%s) to delete", d.Id())

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE"},
		Target:     "DELETED",
		Refresh:    ServerV2StateRefreshFunc(computeClient, d.Id()),
		Timeout:    10 * time.Minute,
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
			errCode, ok := err.(*gophercloud.UnexpectedResponseCodeError)
			if !ok {
				return nil, "", err
			}
			if errCode.Actual == 404 {
				return s, "DELETED", nil
			}
			return nil, "", err
		}

		return s, s.Status, nil
	}
}

func resourceInstanceSecGroupsV2(d *schema.ResourceData) []string {
	rawSecGroups := d.Get("security_groups").([]interface{})
	secgroups := make([]string, len(rawSecGroups))
	for i, raw := range rawSecGroups {
		secgroups[i] = raw.(string)
	}
	return secgroups
}

func resourceInstanceNetworks(computeClient *gophercloud.ServiceClient, d *schema.ResourceData) ([]map[string]interface{}, error) {
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
		allPages, err := tenantnetworks.List(computeClient).AllPages()
		if err != nil {
			errCode, ok := err.(*gophercloud.UnexpectedResponseCodeError)
			if !ok {
				return nil, err
			}

			if errCode.Actual == 404 {
				tenantNetworkExt = false
			} else {
				return nil, err
			}
		}

		networkID := ""
		networkName := ""
		if tenantNetworkExt {
			networkList, err := tenantnetworks.ExtractNetworks(allPages)
			if err != nil {
				return nil, err
			}

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
			"uuid":        networkID,
			"name":        networkName,
			"port":        rawMap["port"].(string),
			"fixed_ip_v4": rawMap["fixed_ip_v4"].(string),
		})
	}

	log.Printf("[DEBUG] networks: %+v", newNetworks)
	return newNetworks, nil
}

func resourceInstanceAddresses(addresses map[string]interface{}) map[string]map[string]interface{} {

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

func resourceInstanceMetadataV2(d *schema.ResourceData) map[string]string {
	m := make(map[string]string)
	for key, val := range d.Get("metadata").(map[string]interface{}) {
		m[key] = val.(string)
	}
	return m
}

func resourceInstanceBlockDeviceV2(d *schema.ResourceData, bd map[string]interface{}) []bootfromvolume.BlockDevice {
	sourceType := bootfromvolume.SourceType(bd["source_type"].(string))
	bfvOpts := []bootfromvolume.BlockDevice{
		bootfromvolume.BlockDevice{
			UUID:            bd["uuid"].(string),
			SourceType:      sourceType,
			VolumeSize:      bd["volume_size"].(int),
			DestinationType: bd["destination_type"].(string),
			BootIndex:       bd["boot_index"].(int),
		},
	}

	return bfvOpts
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

func getImageID(client *gophercloud.ServiceClient, d *schema.ResourceData) (string, error) {
	imageId := d.Get("image_id").(string)

	if imageId != "" {
		return imageId, nil
	}

	imageCount := 0
	imageName := d.Get("image_name").(string)
	if imageName != "" {
		pager := images.ListDetail(client, &images.ListOpts{
			Name: imageName,
		})
		pager.EachPage(func(page pagination.Page) (bool, error) {
			imageList, err := images.ExtractImages(page)
			if err != nil {
				return false, err
			}

			for _, i := range imageList {
				if i.Name == imageName {
					imageCount++
					imageId = i.ID
				}
			}
			return true, nil
		})

		switch imageCount {
		case 0:
			return "", fmt.Errorf("Unable to find image: %s", imageName)
		case 1:
			return imageId, nil
		default:
			return "", fmt.Errorf("Found %d images matching %s", imageCount, imageName)
		}
	}
	return "", fmt.Errorf("Neither an image ID nor an image name were able to be determined.")
}

func getFlavorID(client *gophercloud.ServiceClient, d *schema.ResourceData) (string, error) {
	flavorId := d.Get("flavor_id").(string)

	if flavorId != "" {
		return flavorId, nil
	}

	flavorCount := 0
	flavorName := d.Get("flavor_name").(string)
	if flavorName != "" {
		pager := flavors.ListDetail(client, nil)
		pager.EachPage(func(page pagination.Page) (bool, error) {
			flavorList, err := flavors.ExtractFlavors(page)
			if err != nil {
				return false, err
			}

			for _, f := range flavorList {
				if f.Name == flavorName {
					flavorCount++
					flavorId = f.ID
				}
			}
			return true, nil
		})

		switch flavorCount {
		case 0:
			return "", fmt.Errorf("Unable to find flavor: %s", flavorName)
		case 1:
			return flavorId, nil
		default:
			return "", fmt.Errorf("Found %d flavors matching %s", flavorCount, flavorName)
		}
	}
	return "", fmt.Errorf("Neither a flavor ID nor a flavor name were able to be determined.")
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
	if len(vols) > 0 {
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
				Target:     "in-use",
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
	}
	return nil
}

func detachVolumesFromInstance(computeClient *gophercloud.ServiceClient, blockClient *gophercloud.ServiceClient, serverId string, vols []interface{}) error {
	if len(vols) > 0 {
		for _, v := range vols {
			va := v.(map[string]interface{})
			aId := va["id"].(string)

			if err := volumeattach.Delete(computeClient, serverId, aId).ExtractErr(); err != nil {
				return err
			}

			stateConf := &resource.StateChangeConf{
				Pending:    []string{"detaching", "in-use"},
				Target:     "available",
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
	}

	return nil
}

func getVolumeAttachments(computeClient *gophercloud.ServiceClient, serverId string) ([]volumeattach.VolumeAttachment, error) {
	var attachments []volumeattach.VolumeAttachment
	err := volumeattach.List(computeClient, serverId).EachPage(func(page pagination.Page) (bool, error) {
		actual, err := volumeattach.ExtractVolumeAttachments(page)
		if err != nil {
			return false, err
		}

		attachments = actual
		return true, nil
	})

	if err != nil {
		return nil, err
	}

	return attachments, nil
}
