package rackspace

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud"
	osBootfromvolume "github.com/rackspace/gophercloud/openstack/compute/v2/extensions/bootfromvolume"
	osVolumeAttach "github.com/rackspace/gophercloud/openstack/compute/v2/extensions/volumeattach"
	osFlavors "github.com/rackspace/gophercloud/openstack/compute/v2/flavors"
	osImages "github.com/rackspace/gophercloud/openstack/compute/v2/images"
	osServers "github.com/rackspace/gophercloud/openstack/compute/v2/servers"
	osNetworks "github.com/rackspace/gophercloud/openstack/networking/v2/networks"
	osPorts "github.com/rackspace/gophercloud/openstack/networking/v2/ports"
	"github.com/rackspace/gophercloud/pagination"
	rsFlavors "github.com/rackspace/gophercloud/rackspace/compute/v2/flavors"
	rsImages "github.com/rackspace/gophercloud/rackspace/compute/v2/images"
	rsServers "github.com/rackspace/gophercloud/rackspace/compute/v2/servers"
	rsVolumeAttach "github.com/rackspace/gophercloud/rackspace/compute/v2/volumeattach"
	rsNetworks "github.com/rackspace/gophercloud/rackspace/networking/v2/networks"
	rsPorts "github.com/rackspace/gophercloud/rackspace/networking/v2/ports"
)

func resourceComputeInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeInstanceCreate,
		Read:   resourceComputeInstanceRead,
		Update: resourceComputeInstanceUpdate,
		Delete: resourceComputeInstanceDelete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: envDefaultFunc("RS_REGION_NAME"),
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
				DefaultFunc: envDefaultFunc("RS_IMAGE_ID"),
			},
			"image_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
				DefaultFunc: envDefaultFunc("RS_IMAGE_NAME"),
			},
			"flavor_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
				Computed:    true,
				DefaultFunc: envDefaultFunc("RS_FLAVOR_ID"),
			},
			"flavor_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
				Computed:    true,
				DefaultFunc: envDefaultFunc("RS_FLAVOR_NAME"),
			},
			"network": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"uuid": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"port": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"fixed_ip": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
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
			"keypair": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"user_data": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"personality": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"security_groups": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
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
		},
	}
}

func resourceComputeInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace compute client: %s", err)
	}

	imageID, err := getImageID(computeClient, d)
	if err != nil {
		return err
	}

	flavorID, err := getFlavorID(computeClient, d)
	if err != nil {
		return err
	}

	createOpts := &rsServers.CreateOpts{
		Name:           d.Get("name").(string),
		ImageRef:       imageID,
		FlavorRef:      flavorID,
		Networks:       resourceInstanceNetworks(d),
		Metadata:       resourceInstanceMetadata(d),
		SecurityGroups: resourceInstanceSecGroups(d),
		ConfigDrive:    d.Get("config_drive").(bool),
		AdminPass:      d.Get("admin_pass").(string),
		KeyPair:        d.Get("keypair").(string),
		BlockDevice:    resourceInstanceBlockDevice(d),
	}

	log.Printf("[INFO] Requesting instance creation")
	server, err := rsServers.Create(computeClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating Rackspace server: %s", err)
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
		Refresh:    ServerStateRefreshFunc(computeClient, server.ID),
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

	// were volume attachments specified?
	if v := d.Get("volume"); v != nil {
		vols := v.(*schema.Set).List()
		if len(vols) > 0 {
			blockClient, err := config.blockStorageClient(d.Get("region").(string))
			if err != nil {
				return fmt.Errorf("Error creating OpenStack block storage client: %s", err)
			}
			if err := attachVolumesToInstance(computeClient, blockClient, d.Id(), vols); err != nil {
				return err
			}
		}
	}

	return resourceComputeInstanceRead(d, meta)
}

func resourceComputeInstanceRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace compute client: %s", err)
	}

	server, err := rsServers.Get(computeClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "server")
	}

	log.Printf("[DEBUG] Retreived Server %s: %+v", d.Id(), server)

	d.Set("region", d.Get("region").(string))
	d.Set("name", server.Name)
	d.Set("access_ip_v4", server.AccessIPv4)
	d.Set("access_ip_v6", server.AccessIPv6)

	hostv4 := server.AccessIPv4
	if hostv4 == "" {
		if publicAddressesRaw, ok := server.Addresses["public"]; ok {
			publicAddresses := publicAddressesRaw.([]interface{})
			for _, paRaw := range publicAddresses {
				pa := paRaw.(map[string]interface{})
				if pa["version"].(float64) == 4 {
					hostv4 = pa["addr"].(string)
					break
				}
			}
		}
	}

	// If no host found, just get the first IPv4 we find
	if hostv4 == "" {
		for _, networkAddresses := range server.Addresses {
			for _, element := range networkAddresses.([]interface{}) {
				address := element.(map[string]interface{})
				if address["version"].(float64) == 4 {
					hostv4 = address["addr"].(string)
					break
				}
			}
		}
	}
	d.Set("access_ip_v4", hostv4)
	log.Printf("hostv4: %s", hostv4)

	hostv6 := server.AccessIPv6
	if hostv6 == "" {
		if publicAddressesRaw, ok := server.Addresses["public"]; ok {
			publicAddresses := publicAddressesRaw.([]interface{})
			for _, paRaw := range publicAddresses {
				pa := paRaw.(map[string]interface{})
				if pa["version"].(float64) == 4 {
					hostv6 = fmt.Sprintf("[%s]", pa["addr"].(string))
					break
				}
			}
		}
	}

	// If no hostv6 found, just get the first IPv6 we find
	if hostv6 == "" {
		for _, networkAddresses := range server.Addresses {
			for _, element := range networkAddresses.([]interface{}) {
				address := element.(map[string]interface{})
				if address["version"].(float64) == 6 {
					hostv6 = fmt.Sprintf("[%s]", address["addr"].(string))
					break
				}
			}
		}
	}
	d.Set("access_ip_v6", hostv6)
	log.Printf("hostv6: %s", hostv6)

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

	d.Set("metadata", server.Metadata)

	flavorID, ok := server.Flavor["id"].(string)
	if !ok {
		return fmt.Errorf("Error setting OpenStack server's flavor: %v", server.Flavor)
	}
	d.Set("flavor_id", flavorID)

	flavor, err := rsFlavors.Get(computeClient, flavorID).Extract()
	if err != nil {
		return err
	}
	d.Set("flavor_name", flavor.Name)

	imageID, ok := server.Image["id"].(string)
	if !ok {
		return fmt.Errorf("Error setting OpenStack server's image: %v", server.Image)
	}
	d.Set("image_id", imageID)

	image, err := rsImages.Get(computeClient, imageID).Extract()
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

func resourceComputeInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace compute client: %s", err)
	}

	var updateOpts osServers.UpdateOpts
	if d.HasChange("name") {
		updateOpts.Name = d.Get("name").(string)
	}
	if d.HasChange("access_ip_v4") {
		updateOpts.AccessIPv4 = d.Get("access_ip_v4").(string)
	}
	if d.HasChange("access_ip_v6") {
		updateOpts.AccessIPv4 = d.Get("access_ip_v6").(string)
	}

	if updateOpts != (osServers.UpdateOpts{}) {
		_, err := rsServers.Update(computeClient, d.Id(), updateOpts).Extract()
		if err != nil {
			return fmt.Errorf("Error updating OpenStack server: %s", err)
		}
	}

	if d.HasChange("metadata") {
		var metadataOpts osServers.MetadataOpts
		metadataOpts = make(osServers.MetadataOpts)
		newMetadata := d.Get("metadata").(map[string]interface{})
		for k, v := range newMetadata {
			metadataOpts[k] = v.(string)
		}

		_, err := rsServers.UpdateMetadata(computeClient, d.Id(), metadataOpts).Extract()
		if err != nil {
			return fmt.Errorf("Error updating OpenStack server (%s) metadata: %s", d.Id(), err)
		}
	}

	if d.HasChange("admin_pass") {
		if newPwd, ok := d.Get("admin_pass").(string); ok {
			err := rsServers.ChangeAdminPassword(computeClient, d.Id(), newPwd).ExtractErr()
			if err != nil {
				return fmt.Errorf("Error changing admin password of Rackspace server (%s): %s", d.Id(), err)
			}
		}
	}

	if d.HasChange("volume") {
		// old attachments and new attachments
		oldAttachments, newAttachments := d.GetChange("volume")

		// for each old attachment, detach the volume
		oldAttachmentSet := oldAttachments.(*schema.Set).List()
		if len(oldAttachmentSet) > 0 {
			blockClient, err := config.blockStorageClient(d.Get("region").(string))
			if err != nil {
				return err
			}
			if err := detachVolumesFromInstance(computeClient, blockClient, d.Id(), oldAttachmentSet); err != nil {
				return err
			}
		}

		// for each new attachment, attach the volume
		newAttachmentSet := newAttachments.(*schema.Set).List()
		if len(newAttachmentSet) > 0 {
			blockClient, err := config.blockStorageClient(d.Get("region").(string))
			if err != nil {
				return err
			}
			if err := attachVolumesToInstance(computeClient, blockClient, d.Id(), newAttachmentSet); err != nil {
				return err
			}
		}

		d.SetPartial("volume")
	}

	if d.HasChange("flavor_id") || d.HasChange("flavor_name") {
		flavorID, err := getFlavorID(computeClient, d)
		if err != nil {
			return err
		}
		resizeOpts := &osServers.ResizeOpts{
			FlavorRef: flavorID,
		}
		err = rsServers.Resize(computeClient, d.Id(), resizeOpts).ExtractErr()
		if err != nil {
			return fmt.Errorf("Error resizing Rackspace server: %s", err)
		}

		// Wait for the instance to finish resizing.
		log.Printf("[DEBUG] Waiting for instance (%s) to finish resizing", d.Id())

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"RESIZE"},
			Target:     "VERIFY_RESIZE",
			Refresh:    ServerStateRefreshFunc(computeClient, d.Id()),
			Timeout:    120 * time.Minute,
			Delay:      10 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf("Error waiting for instance (%s) to resize: %s", d.Id(), err)
		}

		// Confirm resize.
		log.Printf("[DEBUG] Confirming resize")
		err = rsServers.ConfirmResize(computeClient, d.Id()).ExtractErr()
		if err != nil {
			return fmt.Errorf("Error confirming resize of Rackspace server: %s", err)
		}

		stateConf = &resource.StateChangeConf{
			Pending:    []string{"VERIFY_RESIZE"},
			Target:     "ACTIVE",
			Refresh:    ServerStateRefreshFunc(computeClient, d.Id()),
			Timeout:    3 * time.Minute,
			Delay:      10 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf("Error waiting for instance (%s) to confirm resize: %s", d.Id(), err)
		}

	}

	return resourceComputeInstanceRead(d, meta)
}

func resourceComputeInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace compute client: %s", err)
	}

	err = rsServers.Delete(computeClient, d.Id()).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting Rackspace server: %s", err)
	}

	// Wait for the instance to delete before moving on.
	log.Printf("[DEBUG] Waiting for instance (%s) to delete", d.Id())

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE"},
		Target:     "DELETED",
		Refresh:    ServerStateRefreshFunc(computeClient, d.Id()),
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

// ServerStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// an OpenStack instance.
func ServerStateRefreshFunc(client *gophercloud.ServiceClient, instanceID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		s, err := rsServers.Get(client, instanceID).Extract()
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

func resourceInstanceSecGroups(d *schema.ResourceData) []string {
	rawSecGroups := d.Get("security_groups").([]interface{})
	secgroups := make([]string, len(rawSecGroups))
	for i, raw := range rawSecGroups {
		secgroups[i] = raw.(string)
	}
	return secgroups
}

func resourceInstanceNetworks(d *schema.ResourceData) []osServers.Network {
	rawNetworks := d.Get("network").([]interface{})
	networks := make([]osServers.Network, len(rawNetworks))
	for i, raw := range rawNetworks {
		rawMap := raw.(map[string]interface{})
		networks[i] = osServers.Network{
			UUID:    rawMap["uuid"].(string),
			Port:    rawMap["port"].(string),
			FixedIP: rawMap["fixed_ip"].(string),
		}
	}
	return networks
}

func resourceInstanceMetadata(d *schema.ResourceData) map[string]string {
	m := make(map[string]string)
	for key, val := range d.Get("metadata").(map[string]interface{}) {
		m[key] = val.(string)
	}
	return m
}

func resourceInstanceBlockDevice(d *schema.ResourceData) []osBootfromvolume.BlockDevice {
	bd := d.Get("block_device").(map[string]interface{})
	sourceType := osBootfromvolume.SourceType(bd["source_type"].(string))
	bfvOpts := []osBootfromvolume.BlockDevice{
		osBootfromvolume.BlockDevice{
			UUID:            bd["uuid"].(string),
			SourceType:      sourceType,
			VolumeSize:      bd["volume_size"].(int),
			DestinationType: bd["destination_type"].(string),
			BootIndex:       bd["boot_index"].(int),
		},
	}

	return bfvOpts
}

func getFirstNetworkID(networkingClient *gophercloud.ServiceClient, instanceID string) (string, error) {
	pager := rsNetworks.List(networkingClient, osNetworks.ListOpts{})

	var networkdID string
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		networkList, err := osNetworks.ExtractNetworks(page)
		if err != nil {
			return false, err
		}

		if len(networkList) > 0 {
			networkdID = networkList[0].ID
			return false, nil
		}
		return false, fmt.Errorf("No network found for the instance %s", instanceID)
	})
	if err != nil {
		return "", err
	}
	return networkdID, nil

}

func getInstancePortID(networkingClient *gophercloud.ServiceClient, instanceID, networkID string) (string, error) {
	pager := rsPorts.List(networkingClient, osPorts.ListOpts{
		DeviceID:  instanceID,
		NetworkID: networkID,
	})

	var portID string
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		portList, err := osPorts.ExtractPorts(page)
		if err != nil {
			return false, err
		}
		for _, port := range portList {
			portID = port.ID
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return "", err
	}
	return portID, nil
}

func getImageID(client *gophercloud.ServiceClient, d *schema.ResourceData) (string, error) {
	imageID := d.Get("image_id").(string)

	if imageID != "" {
		return imageID, nil
	}

	imageCount := 0
	imageName := d.Get("image_name").(string)
	if imageName != "" {
		pager := rsImages.ListDetail(client, &osImages.ListOpts{
			Name: imageName,
		})
		pager.EachPage(func(page pagination.Page) (bool, error) {
			imageList, err := osImages.ExtractImages(page)
			if err != nil {
				return false, err
			}

			for _, i := range imageList {
				if i.Name == imageName {
					imageCount++
					imageID = i.ID
				}
			}
			return true, nil
		})

		switch imageCount {
		case 0:
			return "", fmt.Errorf("Unable to find image: %s", imageName)
		case 1:
			return imageID, nil
		default:
			return "", fmt.Errorf("Found %d images matching %s", imageCount, imageName)
		}
	}
	return "", fmt.Errorf("Neither an image ID nor an image name were able to be determined.")
}

func getFlavorID(client *gophercloud.ServiceClient, d *schema.ResourceData) (string, error) {
	flavorID := d.Get("flavor_id").(string)

	if flavorID != "" {
		return flavorID, nil
	}

	flavorCount := 0
	flavorName := d.Get("flavor_name").(string)
	if flavorName != "" {
		pager := rsFlavors.ListDetail(client, nil)
		pager.EachPage(func(page pagination.Page) (bool, error) {
			flavorList, err := osFlavors.ExtractFlavors(page)
			if err != nil {
				return false, err
			}

			for _, f := range flavorList {
				if f.Name == flavorName {
					flavorCount++
					flavorID = f.ID
				}
			}
			return true, nil
		})

		switch flavorCount {
		case 0:
			return "", fmt.Errorf("Unable to find flavor: %s", flavorName)
		case 1:
			return flavorID, nil
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
	buf.WriteString(fmt.Sprintf("%s-", m["device"].(string)))
	return hashcode.String(buf.String())
}

func attachVolumesToInstance(computeClient *gophercloud.ServiceClient, blockClient *gophercloud.ServiceClient, serverID string, vols []interface{}) error {
	if len(vols) > 0 {
		for _, v := range vols {
			va := v.(map[string]interface{})
			volumeID := va["volume_id"].(string)
			device := va["device"].(string)

			s := ""
			if serverID != "" {
				s = serverID
			} else if va["server_id"] != "" {
				s = va["server_id"].(string)
			} else {
				return fmt.Errorf("Unable to determine server ID to attach volume.")
			}

			vaOpts := &osVolumeAttach.CreateOpts{
				Device:   device,
				VolumeID: volumeID,
			}

			if _, err := rsVolumeAttach.Create(computeClient, s, vaOpts).Extract(); err != nil {
				return err
			}

			stateConf := &resource.StateChangeConf{
				Target:     "in-use",
				Refresh:    VolumeStateRefreshFunc(blockClient, va["volume_id"].(string)),
				Timeout:    30 * time.Minute,
				Delay:      5 * time.Second,
				MinTimeout: 2 * time.Second,
			}

			if _, err := stateConf.WaitForState(); err != nil {
				return err
			}

			log.Printf("[INFO] Attached volume %s to instance %s", volumeID, serverID)
		}
	}
	return nil
}

func detachVolumesFromInstance(computeClient *gophercloud.ServiceClient, blockClient *gophercloud.ServiceClient, serverID string, vols []interface{}) error {
	if len(vols) > 0 {
		for _, v := range vols {
			va := v.(map[string]interface{})
			aID := va["id"].(string)

			if err := rsVolumeAttach.Delete(computeClient, serverID, aID).ExtractErr(); err != nil {
				return err
			}

			stateConf := &resource.StateChangeConf{
				Target:     "available",
				Refresh:    VolumeStateRefreshFunc(blockClient, va["volume_id"].(string)),
				Timeout:    30 * time.Minute,
				Delay:      5 * time.Second,
				MinTimeout: 2 * time.Second,
			}

			if _, err := stateConf.WaitForState(); err != nil {
				return err
			}
			log.Printf("[INFO] Detached volume %s from instance %s", va["volume_id"], serverID)
		}
	}

	return nil
}

func getVolumeAttachments(computeClient *gophercloud.ServiceClient, serverID string) ([]osVolumeAttach.VolumeAttachment, error) {
	var attachments []osVolumeAttach.VolumeAttachment
	err := rsVolumeAttach.List(computeClient, serverID).EachPage(func(page pagination.Page) (bool, error) {
		actual, err := osVolumeAttach.ExtractVolumeAttachments(page)
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
