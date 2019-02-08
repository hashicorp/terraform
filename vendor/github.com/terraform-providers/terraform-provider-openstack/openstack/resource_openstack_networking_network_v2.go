package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/provider"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/vlantransparent"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
)

func resourceNetworkingNetworkV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceNetworkingNetworkV2Create,
		Read:   resourceNetworkingNetworkV2Read,
		Update: resourceNetworkingNetworkV2Update,
		Delete: resourceNetworkingNetworkV2Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"admin_state_up": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
				Computed: true,
			},

			"shared": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
				Computed: true,
			},

			"external": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
				Computed: true,
			},

			"tenant_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"segments": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"physical_network": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"network_type": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"segmentation_id": {
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},

			"value_specs": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"tags": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"all_tags": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"availability_zone_hints": {
				Type:     schema.TypeSet,
				Computed: true,
				ForceNew: true,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"transparent_vlan": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
		},
	}
}

func resourceNetworkingNetworkV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	azHints := d.Get("availability_zone_hints").(*schema.Set)

	createOpts := NetworkCreateOpts{
		networks.CreateOpts{
			Name:                  d.Get("name").(string),
			Description:           d.Get("description").(string),
			TenantID:              d.Get("tenant_id").(string),
			AvailabilityZoneHints: expandToStringSlice(azHints.List()),
		},
		MapValueSpecs(d),
	}

	if v, ok := d.GetOkExists("admin_state_up"); ok {
		asu := v.(bool)
		createOpts.AdminStateUp = &asu
	}

	if v, ok := d.GetOkExists("shared"); ok {
		shared := v.(bool)
		createOpts.Shared = &shared
	}

	segments := expandNetworkingNetworkSegmentsV2(d.Get("segments").(*schema.Set))
	isExternal := d.Get("external").(bool)
	isVLANTransparent := d.Get("transparent_vlan").(bool)

	// Declare a finalCreateOpts interface.
	var finalCreateOpts networks.CreateOptsBuilder
	finalCreateOpts = createOpts

	// Add networking segments if specified.
	if len(segments) > 0 {
		finalCreateOpts = provider.CreateOptsExt{
			CreateOptsBuilder: finalCreateOpts,
			Segments:          segments,
		}
	}

	// Add the external attribute if specified.
	if isExternal {
		finalCreateOpts = external.CreateOptsExt{
			CreateOptsBuilder: finalCreateOpts,
			External:          &isExternal,
		}
	}

	// Add the transparent VLAN attribute if specified.
	if isVLANTransparent {
		finalCreateOpts = vlantransparent.CreateOptsExt{
			CreateOptsBuilder: finalCreateOpts,
			VLANTransparent:   &isVLANTransparent,
		}
	}

	log.Printf("[DEBUG] openstack_networking_network_v2 create options: %#v", finalCreateOpts)
	n, err := networks.Create(networkingClient, finalCreateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating openstack_networking_network_v2: %s", err)
	}

	log.Printf("[DEBUG] Waiting for openstack_networking_network_v2 %s to become available.", n.ID)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"BUILD"},
		Target:     []string{"ACTIVE", "DOWN"},
		Refresh:    resourceNetworkingNetworkV2StateRefreshFunc(networkingClient, n.ID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for openstack_networking_network_v2 %s to become available: %s", n.ID, err)
	}

	d.SetId(n.ID)

	tags := networkV2AttributesTags(d)
	if len(tags) > 0 {
		tagOpts := attributestags.ReplaceAllOpts{Tags: tags}
		tags, err := attributestags.ReplaceAll(networkingClient, "networks", n.ID, tagOpts).Extract()
		if err != nil {
			return fmt.Errorf("Error setting tags on openstack_networking_network_v2 %s: %s", n.ID, err)
		}
		log.Printf("[DEBUG] Set tags %s on openstack_networking_network_v2 %s", tags, n.ID)
	}

	log.Printf("[DEBUG] Created openstack_networking_network_v2 %s: %#v", n.ID, n)
	return resourceNetworkingNetworkV2Read(d, meta)
}

func resourceNetworkingNetworkV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	var n struct {
		networks.Network
		external.NetworkExternalExt
		vlantransparent.TransparentExt
	}
	err = networks.Get(networkingClient, d.Id()).ExtractInto(&n)
	if err != nil {
		return CheckDeleted(d, err, "Error getting openstack_networking_network_v2")
	}

	log.Printf("[DEBUG] Retrieved openstack_networking_network_v2 %s: %#v", d.Id(), n)

	d.Set("name", n.Name)
	d.Set("description", n.Description)
	d.Set("admin_state_up", n.AdminStateUp)
	d.Set("shared", n.Shared)
	d.Set("external", n.External)
	d.Set("tenant_id", n.TenantID)
	d.Set("region", GetRegion(d, config))
	d.Set("transparent_vlan", n.VLANTransparent)

	networkV2ReadAttributesTags(d, n.Tags)

	if err := d.Set("availability_zone_hints", n.AvailabilityZoneHints); err != nil {
		log.Printf("[DEBUG] Unable to set openstack_networking_network_v2 %s availability_zone_hints: %s", d.Id(), err)
	}

	return nil
}

func resourceNetworkingNetworkV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	// Declare finalUpdateOpts interface and basic updateOpts structure.
	var (
		finalUpdateOpts networks.UpdateOptsBuilder
		updateOpts      networks.UpdateOpts
	)

	// Populate basic updateOpts.
	if d.HasChange("name") {
		updateOpts.Name = d.Get("name").(string)
	}
	if d.HasChange("description") {
		description := d.Get("description").(string)
		updateOpts.Description = &description
	}
	if d.HasChange("admin_state_up") {
		asu := d.Get("admin_state_up").(bool)
		updateOpts.AdminStateUp = &asu
	}
	if d.HasChange("shared") {
		shared := d.Get("shared").(bool)
		updateOpts.Shared = &shared
	}

	// Change tags if needed.
	if d.HasChange("tags") {
		tags := networkV2UpdateAttributesTags(d)
		tagOpts := attributestags.ReplaceAllOpts{Tags: tags}
		tags, err := attributestags.ReplaceAll(networkingClient, "networks", d.Id(), tagOpts).Extract()
		if err != nil {
			return fmt.Errorf("Error setting tags on openstack_networking_network_v2 %s: %s", d.Id(), err)
		}
		log.Printf("[DEBUG] Set tags %s on openstack_networking_network_v2 %s", tags, d.Id())
	}

	// Save basic updateOpts into finalUpdateOpts.
	finalUpdateOpts = updateOpts

	// Populate extensions options.
	isExternal := false
	if d.HasChange("external") {
		isExternal = d.Get("external").(bool)
		finalUpdateOpts = external.UpdateOptsExt{
			UpdateOptsBuilder: finalUpdateOpts,
			External:          &isExternal,
		}
	}

	log.Printf("[DEBUG] openstack_networking_network_v2 %s update options: %#v", d.Id(), finalUpdateOpts)
	_, err = networks.Update(networkingClient, d.Id(), finalUpdateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating openstack_networking_network_v2 %s: %s", d.Id(), err)
	}

	return resourceNetworkingNetworkV2Read(d, meta)
}

func resourceNetworkingNetworkV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	if err := networks.Delete(networkingClient, d.Id()).ExtractErr(); err != nil {
		return CheckDeleted(d, err, "Error deleting openstack_networking_network_v2")
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE"},
		Target:     []string{"DELETED"},
		Refresh:    resourceNetworkingNetworkV2StateRefreshFunc(networkingClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for openstack_networking_network_v2 %s to delete: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}
