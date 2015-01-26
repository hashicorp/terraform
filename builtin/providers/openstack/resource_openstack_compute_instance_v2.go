package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"
	"github.com/rackspace/gophercloud/pagination"
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
				DefaultFunc: envDefaultFunc("OS_REGION_NAME"),
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"image_ref": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"flavor_ref": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"security_groups": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},
			"availability_zone": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"networks": &schema.Schema{
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
		},
	}
}

func resourceComputeInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := openstack.NewComputeV2(config.osClient, gophercloud.EndpointOpts{
		Region: d.Get("region").(string),
	})
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}


	var createOpts servers.CreateOptsBuilder

	serverCreateOpts := &servers.CreateOpts{
		Name:             d.Get("name").(string),
		ImageRef:         d.Get("image_ref").(string),
		FlavorRef:        d.Get("flavor_ref").(string),
		SecurityGroups:   resourceInstanceSecGroups(d),
		AvailabilityZone: d.Get("availability_zone").(string),
		Networks:         resourceInstanceNetworks(d),
		Metadata:         resourceInstanceMetadata(d),
		ConfigDrive:      d.Get("config_drive").(bool),
		AdminPass:        d.Get("admin_pass").(string),
	}

	if keyName, ok := d.Get("key_pair").(string); ok && keyName != "" {
		createOpts = &keypairs.CreateOptsExt{
			serverCreateOpts,
			keyName,
		}
	}

	log.Printf("[INFO] Requesting instance creation")
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

	return resourceComputeInstanceRead(d, meta)
}

func resourceComputeInstanceRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := openstack.NewComputeV2(config.osClient, gophercloud.EndpointOpts{
		Region: d.Get("region").(string),
	})
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	server, err := servers.Get(computeClient, d.Id()).Extract()
	if err != nil {
		return fmt.Errorf("Error retrieving OpenStack server: %s", err)
	}

	log.Printf("[DEBUG] Retreived Server %s: %+v", d.Id(), server)

	d.Set("name", server.Name)
	d.Set("access_ip_v4", server.AccessIPv4)
	d.Set("access_ip_v6", server.AccessIPv6)

	host := server.AccessIPv4
	if host == "" {
		if publicAddressesRaw, ok := server.Addresses["public"]; ok {
			publicAddresses := publicAddressesRaw.([]interface{})
			for _, paRaw := range publicAddresses {
				pa := paRaw.(map[string]interface{})
				if pa["version"].(float64) == 4 {
					host = pa["addr"].(string)
					d.Set("access_ip_v4", host)
				}
			}
		}
	}
	d.Set("host", host)

	log.Printf("host: %s", host)

	// Initialize the connection info
	d.SetConnInfo(map[string]string{
		"type": "ssh",
		"host": host,
	})

	d.Set("metadata", server.Metadata)

	var currentSG []string
	err = secgroups.ListByServer(computeClient, d.Id()).EachPage(func(page pagination.Page) (bool, error) {
		secGrpList, err := secgroups.ExtractSecurityGroups(page)
		if err != nil {
			return false, fmt.Errorf("Error setting security groups for OpenStack server: %s", err)
		}

		for _, sg := range secGrpList {
			currentSG = append(currentSG, sg.Name)
		}

		return true, nil
	})
	d.Set("security_groups", currentSG)

	newFlavor, ok := server.Flavor["id"].(string)
	if !ok {
		return fmt.Errorf("Error setting OpenStack server's flavor: %v", server.Flavor)
	}
	d.Set("flavor_ref", newFlavor)

	return nil
}

func resourceComputeInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := openstack.NewComputeV2(config.osClient, gophercloud.EndpointOpts{
		Region: d.Get("region").(string),
	})
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

	log.Printf("[DEBUG] Updating Server %s with options: %+v", d.Id(), updateOpts)

	_, err = servers.Update(computeClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack server: %s", err)
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
		oldSGSet, newSGSet := oldSGRaw.(*schema.Set), newSGRaw.(*schema.Set)
		secgroupsToAdd := newSGSet.Difference(oldSGSet)
		secgroupsToRemove := oldSGSet.Difference(newSGSet)

		log.Printf("[DEBUG] Security groups to add: %v", secgroupsToAdd)

		log.Printf("[DEBUG] Security groups to remove: %v", secgroupsToRemove)

		for _, g := range secgroupsToAdd.List() {
			err := secgroups.AddServerToGroup(computeClient, d.Id(), g.(string)).ExtractErr()
			if err != nil {
				return fmt.Errorf("Error adding security group to OpenStack server (%s): %s", d.Id(), err)
			}
			log.Printf("[DEBUG] Added security group (%s) to instance (%s)", g.(string), d.Id())
		}

		for _, g := range secgroupsToRemove.List() {
			err := secgroups.RemoveServerFromGroup(computeClient, d.Id(), g.(string)).ExtractErr()
			if err != nil {
				return fmt.Errorf("Error removing security group from OpenStack server (%s): %s", d.Id(), err)
			}
			log.Printf("[DEBUG] Removed security group (%s) from instance (%s)", g.(string), d.Id())
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

	if d.HasChange("flavor_ref") {
		resizeOpts := &servers.ResizeOpts{
			FlavorRef: d.Get("flavor_ref").(string),
		}
		err := servers.Resize(computeClient, d.Id(), resizeOpts).ExtractErr()
		if err != nil {
			return fmt.Errorf("Error resizing OpenStack server: %s", err)
		}

		// Wait for the instance to finish resizing.
		log.Printf("[DEBUG] Waiting for instance (%s) to finish resizing", d.Id())

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"RESIZE"},
			Target:     "VERIFY_RESIZE",
			Refresh:    ServerStateRefreshFunc(computeClient, d.Id()),
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
	computeClient, err := openstack.NewComputeV2(config.osClient, gophercloud.EndpointOpts{
		Region: d.Get("region").(string),
	})
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
		Target:     "",
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
		s, err := servers.Get(client, instanceID).Extract()
		if err != nil {
			return nil, "", err
		}

		return s, s.Status, nil
	}
}

func resourceInstanceSecGroups(d *schema.ResourceData) []string {
	rawSecGroups := d.Get("security_groups").(*schema.Set)
	secgroups := make([]string, rawSecGroups.Len())
	for i, raw := range rawSecGroups.List() {
		secgroups[i] = raw.(string)
	}
	return secgroups
}

func resourceInstanceNetworks(d *schema.ResourceData) []servers.Network {
	rawNetworks := d.Get("networks").([]interface{})
	networks := make([]servers.Network, len(rawNetworks))
	for i, raw := range rawNetworks {
		rawMap := raw.(map[string]interface{})
		networks[i] = servers.Network{
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
