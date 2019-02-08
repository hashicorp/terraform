package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
)

func resourceNetworkingRouterV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceNetworkingRouterV2Create,
		Read:   resourceNetworkingRouterV2Read,
		Update: resourceNetworkingRouterV2Update,
		Delete: resourceNetworkingRouterV2Delete,
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
			"distributed": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"external_gateway": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      false,
				Computed:      true,
				Deprecated:    "use external_network_id instead",
				ConflictsWith: []string{"external_network_id"},
			},
			"external_network_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      false,
				Computed:      true,
				ConflictsWith: []string{"external_gateway"},
			},
			"enable_snat": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
				Computed: true,
			},
			"external_fixed_ip": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: false,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subnet_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"ip_address": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"tenant_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"value_specs": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
			"availability_zone_hints": {
				Type:     schema.TypeList,
				Computed: true,
				ForceNew: true,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"vendor_options": {
				Type:     schema.TypeSet,
				Optional: true,
				MinItems: 1,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"set_router_gateway_after_create": {
							Type:     schema.TypeBool,
							Default:  false,
							Optional: true,
						},
					},
				},
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
		},
	}
}

func resourceNetworkingRouterV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	createOpts := RouterCreateOpts{
		routers.CreateOpts{
			Name:                  d.Get("name").(string),
			Description:           d.Get("description").(string),
			TenantID:              d.Get("tenant_id").(string),
			AvailabilityZoneHints: resourceNetworkingAvailabilityZoneHintsV2(d),
		},
		MapValueSpecs(d),
	}

	if asuRaw, ok := d.GetOk("admin_state_up"); ok {
		asu := asuRaw.(bool)
		createOpts.AdminStateUp = &asu
	}

	if dRaw, ok := d.GetOkExists("distributed"); ok {
		d := dRaw.(bool)
		createOpts.Distributed = &d
	}

	// Get Vendor_options
	vendorOptionsRaw := d.Get("vendor_options").(*schema.Set)
	var vendorUpdateGateway bool
	if vendorOptionsRaw.Len() > 0 {
		vendorOptions := expandVendorOptions(vendorOptionsRaw.List())
		vendorUpdateGateway = vendorOptions["set_router_gateway_after_create"].(bool)
	}

	// Gateway settings
	var externalNetworkID string
	var gatewayInfo routers.GatewayInfo
	if v := d.Get("external_gateway").(string); v != "" {
		externalNetworkID = v
		gatewayInfo.NetworkID = externalNetworkID
	}

	if v := d.Get("external_network_id").(string); v != "" {
		externalNetworkID = v
		gatewayInfo.NetworkID = externalNetworkID
	}

	if esRaw, ok := d.GetOkExists("enable_snat"); ok {
		if externalNetworkID == "" {
			return fmt.Errorf("setting enable_snat requires external_network_id to be set")
		}
		es := esRaw.(bool)
		gatewayInfo.EnableSNAT = &es
	}

	externalFixedIPs := resourceRouterExternalFixedIPsV2(d)
	if len(externalFixedIPs) > 0 {
		if externalNetworkID == "" {
			return fmt.Errorf("setting an external_fixed_ip requires external_network_id to be set")
		}
		gatewayInfo.ExternalFixedIPs = externalFixedIPs
	}

	// vendorUpdateGateway is a flag for certain vendor-specific virtual routers
	// which do not allow gateway settings to be set during router creation.
	// If this flag was not enabled, then we can safely set the gateway
	// information during create.
	if !vendorUpdateGateway && externalNetworkID != "" {
		createOpts.GatewayInfo = &gatewayInfo
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	n, err := routers.Create(networkingClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack Neutron router: %s", err)
	}
	log.Printf("[INFO] Router ID: %s", n.ID)

	log.Printf("[DEBUG] Waiting for OpenStack Neutron Router (%s) to become available", n.ID)
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"BUILD", "PENDING_CREATE", "PENDING_UPDATE"},
		Target:     []string{"ACTIVE"},
		Refresh:    waitForRouterActive(networkingClient, n.ID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()

	d.SetId(n.ID)

	// If the vendorUpdateGateway flag was specified and if an external network
	// was specified, then set the gateway information after router creation.
	if vendorUpdateGateway && externalNetworkID != "" {
		log.Printf("[DEBUG] Adding External Network %s to router ID %s", externalNetworkID, d.Id())

		var updateOpts routers.UpdateOpts
		updateOpts.GatewayInfo = &gatewayInfo

		log.Printf("[DEBUG] Assigning external gateway to Router %s with options: %+v", d.Id(), updateOpts)
		_, err = routers.Update(networkingClient, d.Id(), updateOpts).Extract()
		if err != nil {
			return fmt.Errorf("Error updating OpenStack Neutron Router: %s", err)
		}
	}

	tags := networkV2AttributesTags(d)
	if len(tags) > 0 {
		tagOpts := attributestags.ReplaceAllOpts{Tags: tags}
		tags, err := attributestags.ReplaceAll(networkingClient, "routers", n.ID, tagOpts).Extract()
		if err != nil {
			return fmt.Errorf("Error creating Tags on Router: %s", err)
		}
		log.Printf("[DEBUG] Set Tags = %+v on Router %+v", tags, n.ID)
	}

	return resourceNetworkingRouterV2Read(d, meta)
}

func resourceNetworkingRouterV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	n, err := routers.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		if _, ok := err.(gophercloud.ErrDefault404); ok {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving OpenStack Neutron Router: %s", err)
	}

	log.Printf("[DEBUG] Retrieved Router %s: %+v", d.Id(), n)

	d.Set("name", n.Name)
	d.Set("description", n.Description)
	d.Set("admin_state_up", n.AdminStateUp)
	d.Set("distributed", n.Distributed)
	d.Set("tenant_id", n.TenantID)
	d.Set("region", GetRegion(d, config))

	networkV2ReadAttributesTags(d, n.Tags)

	if err := d.Set("availability_zone_hints", n.AvailabilityZoneHints); err != nil {
		log.Printf("[DEBUG] unable to set availability_zone_hints: %s", err)
	}

	// Gateway settings
	d.Set("external_gateway", n.GatewayInfo.NetworkID)
	d.Set("external_network_id", n.GatewayInfo.NetworkID)
	d.Set("enable_snat", n.GatewayInfo.EnableSNAT)

	var externalFixedIPs []map[string]string
	for _, v := range n.GatewayInfo.ExternalFixedIPs {
		externalFixedIPs = append(externalFixedIPs, map[string]string{
			"subnet_id":  v.SubnetID,
			"ip_address": v.IPAddress,
		})
	}

	if err = d.Set("external_fixed_ip", externalFixedIPs); err != nil {
		log.Printf("[DEBUG] unable to set external_fixed_ip: %s", err)
	}

	return nil
}

func resourceNetworkingRouterV2Update(d *schema.ResourceData, meta interface{}) error {
	routerId := d.Id()
	osMutexKV.Lock(routerId)
	defer osMutexKV.Unlock(routerId)

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	var hasChange bool
	var updateOpts routers.UpdateOpts
	if d.HasChange("name") {
		hasChange = true
		updateOpts.Name = d.Get("name").(string)
	}
	if d.HasChange("description") {
		hasChange = true
		description := d.Get("description").(string)
		updateOpts.Description = &description
	}
	if d.HasChange("admin_state_up") {
		hasChange = true
		asu := d.Get("admin_state_up").(bool)
		updateOpts.AdminStateUp = &asu
	}

	// Gateway settings
	var updateGatewaySettings bool
	var externalNetworkID string
	gatewayInfo := routers.GatewayInfo{}

	if v := d.Get("external_gateway").(string); v != "" {
		externalNetworkID = v
	}

	if v := d.Get("external_network_id").(string); v != "" {
		externalNetworkID = v
	}

	if externalNetworkID != "" {
		gatewayInfo.NetworkID = externalNetworkID
	}

	if d.HasChange("external_gateway") {
		updateGatewaySettings = true
	}

	if d.HasChange("external_network_id") {
		updateGatewaySettings = true
	}

	if d.HasChange("enable_snat") {
		updateGatewaySettings = true
		if externalNetworkID == "" {
			return fmt.Errorf("setting enable_snat requires external_network_id to be set")
		}

		enableSNAT := d.Get("enable_snat").(bool)
		gatewayInfo.EnableSNAT = &enableSNAT
	}

	if d.HasChange("external_fixed_ip") {
		updateGatewaySettings = true

		externalFixedIPs := resourceRouterExternalFixedIPsV2(d)
		gatewayInfo.ExternalFixedIPs = externalFixedIPs
		if len(externalFixedIPs) > 0 {
			if externalNetworkID == "" {
				return fmt.Errorf("setting an external_fixed_ip requires external_network_id to be set")
			}
		}
	}

	if updateGatewaySettings {
		hasChange = true
		updateOpts.GatewayInfo = &gatewayInfo
	}

	if hasChange {
		log.Printf("[DEBUG] Updating Router %s with options: %+v", d.Id(), updateOpts)
		_, err = routers.Update(networkingClient, d.Id(), updateOpts).Extract()
		if err != nil {
			return fmt.Errorf("Error updating OpenStack Neutron Router: %s", err)
		}
	}

	if d.HasChange("tags") {
		tags := networkV2UpdateAttributesTags(d)
		tagOpts := attributestags.ReplaceAllOpts{Tags: tags}
		tags, err := attributestags.ReplaceAll(networkingClient, "routers", d.Id(), tagOpts).Extract()
		if err != nil {
			return fmt.Errorf("Error updating Tags on Router: %s", err)
		}
		log.Printf("[DEBUG] Updated Tags = %+v on Router %+v", tags, d.Id())
	}

	return resourceNetworkingRouterV2Read(d, meta)
}

func resourceNetworkingRouterV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE"},
		Target:     []string{"DELETED"},
		Refresh:    waitForRouterDelete(networkingClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack Neutron Router: %s", err)
	}

	d.SetId("")
	return nil
}

func waitForRouterActive(networkingClient *gophercloud.ServiceClient, routerId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		r, err := routers.Get(networkingClient, routerId).Extract()
		if err != nil {
			return nil, r.Status, err
		}

		log.Printf("[DEBUG] OpenStack Neutron Router: %+v", r)
		return r, r.Status, nil
	}
}

func waitForRouterDelete(networkingClient *gophercloud.ServiceClient, routerId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		log.Printf("[DEBUG] Attempting to delete OpenStack Router %s.\n", routerId)

		r, err := routers.Get(networkingClient, routerId).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				log.Printf("[DEBUG] Successfully deleted OpenStack Router %s", routerId)
				return r, "DELETED", nil
			}
			return r, "ACTIVE", err
		}

		err = routers.Delete(networkingClient, routerId).ExtractErr()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				log.Printf("[DEBUG] Successfully deleted OpenStack Router %s", routerId)
				return r, "DELETED", nil
			}
			return r, "ACTIVE", err
		}

		log.Printf("[DEBUG] OpenStack Router %s still active.\n", routerId)
		return r, "ACTIVE", nil
	}
}

func resourceRouterExternalFixedIPsV2(d *schema.ResourceData) []routers.ExternalFixedIP {
	var externalFixedIPs []routers.ExternalFixedIP
	eFIPs := d.Get("external_fixed_ip").([]interface{})

	for _, eFIP := range eFIPs {
		v := eFIP.(map[string]interface{})
		fip := routers.ExternalFixedIP{
			SubnetID:  v["subnet_id"].(string),
			IPAddress: v["ip_address"].(string),
		}
		externalFixedIPs = append(externalFixedIPs, fip)
	}

	return externalFixedIPs
}
