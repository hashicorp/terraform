package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/subnetpools"
)

func resourceNetworkingSubnetPoolV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceNetworkingSubnetPoolV2Create,
		Read:   resourceNetworkingSubnetPoolV2Read,
		Update: resourceNetworkingSubnetPoolV2Update,
		Delete: resourceNetworkingSubnetPoolV2Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"default_quota": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: false,
			},
			"project_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"created_at": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: false,
				Computed: true,
			},
			"updated_at": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: false,
				Computed: true,
			},
			"prefixes": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"default_prefixlen": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: false,
				Computed: true,
			},
			"min_prefixlen": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: false,
				Computed: true,
			},
			"max_prefixlen": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: false,
				Computed: true,
			},
			"address_scope_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"ip_version": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(int)
					if value != 4 && value != 6 {
						errors = append(errors, fmt.Errorf(
							"Only 4 and 6 are supported values for 'ip_version'"))
					}
					return
				},
			},
			"shared": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"is_default": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
			},
			"revision_number": &schema.Schema{
				Type:     schema.TypeInt,
				ForceNew: false,
				Computed: true,
			},
			"value_specs": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceNetworkingSubnetPoolV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	createOpts := SubnetPoolCreateOpts{
		subnetpools.CreateOpts{
			Name:             d.Get("name").(string),
			DefaultQuota:     d.Get("default_quota").(int),
			ProjectID:        d.Get("project_id").(string),
			Prefixes:         resourceSubnetPoolPrefixesV2(d),
			DefaultPrefixLen: d.Get("default_prefixlen").(int),
			MinPrefixLen:     d.Get("min_prefixlen").(int),
			MaxPrefixLen:     d.Get("max_prefixlen").(int),
			AddressScopeID:   d.Get("address_scope_id").(string),
			Shared:           d.Get("shared").(bool),
			Description:      d.Get("description").(string),
			IsDefault:        d.Get("is_default").(bool),
		},
		MapValueSpecs(d),
	}

	s, err := subnetpools.Create(networkingClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack Neutron Subnetpool: %s", err)
	}

	log.Printf("[DEBUG] Waiting for Subnetpool (%s) to become available", s.ID)
	stateConf := &resource.StateChangeConf{
		Target:     []string{"ACTIVE"},
		Refresh:    waitForSubnetPoolActive(networkingClient, s.ID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack Neutron Subnetpool: %s", err)
	}

	d.SetId(s.ID)

	log.Printf("[DEBUG] Created Subnetpool %s: %#v", s.ID, s)
	return resourceNetworkingSubnetPoolV2Read(d, meta)
}

func resourceNetworkingSubnetPoolV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	s, err := subnetpools.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "subnetpool")
	}

	log.Printf("[DEBUG] Retrieved Subnetpool %s: %#v", d.Id(), s)

	d.Set("name", s.Name)
	d.Set("default_quota", s.DefaultQuota)
	d.Set("project_id", s.ProjectID)
	d.Set("created_at", s.CreatedAt)
	d.Set("updated_at", s.UpdatedAt)
	d.Set("default_prefixlen", s.DefaultPrefixLen)
	d.Set("min_prefixlen", s.MinPrefixLen)
	d.Set("max_prefixlen", s.MaxPrefixLen)
	d.Set("address_scope_id", s.AddressScopeID)
	d.Set("ip_version", s.IPversion)
	d.Set("shared", s.Shared)
	d.Set("is_default", s.IsDefault)
	d.Set("description", s.Description)
	d.Set("revision_number", s.RevisionNumber)
	d.Set("region", GetRegion(d, config))

	if err := d.Set("prefixes", s.Prefixes); err != nil {
		log.Printf("[WARN] unable to set prefixes: %s", err)
	}

	return nil
}

func resourceNetworkingSubnetPoolV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	var updateOpts subnetpools.UpdateOpts

	if d.HasChange("name") {
		updateOpts.Name = d.Get("name").(string)
	}

	if d.HasChange("default_quota") {
		v := d.Get("default_quota").(int)
		updateOpts.DefaultQuota = &v
	}

	if d.HasChange("project_id") {
		updateOpts.ProjectID = d.Get("project_id").(string)
	}

	if d.HasChange("prefixes") {
		updateOpts.Prefixes = resourceSubnetPoolPrefixesV2(d)
	}

	if d.HasChange("default_prefixlen") {
		updateOpts.DefaultPrefixLen = d.Get("default_prefixlen").(int)
	}

	if d.HasChange("min_prefixlen") {
		updateOpts.MinPrefixLen = d.Get("min_prefixlen").(int)
	}

	if d.HasChange("max_prefixlen") {
		updateOpts.MaxPrefixLen = d.Get("max_prefixlen").(int)
	}

	if d.HasChange("address_scope_id") {
		v := d.Get("address_scope_id").(string)
		updateOpts.AddressScopeID = &v
	}

	if d.HasChange("description") {
		v := d.Get("description").(string)
		updateOpts.Description = &v
	}

	if d.HasChange("is_default") {
		v := d.Get("is_default").(bool)
		updateOpts.IsDefault = &v
	}

	log.Printf("[DEBUG] Updating Subnetpool %s with options: %+v", d.Id(), updateOpts)

	_, err = subnetpools.Update(networkingClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack Neutron Subnetpool: %s", err)
	}

	return resourceNetworkingSubnetPoolV2Read(d, meta)
}

func resourceNetworkingSubnetPoolV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE"},
		Target:     []string{"DELETED"},
		Refresh:    waitForSubnetPoolDelete(networkingClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack Neutron Subnetpool: %s", err)
	}

	return nil
}

func waitForSubnetPoolActive(networkingClient *gophercloud.ServiceClient, subnetPoolID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		s, err := subnetpools.Get(networkingClient, subnetPoolID).Extract()
		if err != nil {
			return nil, "", err
		}

		log.Printf("[DEBUG] OpenStack Neutron subnetpool: %+v", s)
		return s, "ACTIVE", nil
	}
}

func waitForSubnetPoolDelete(networkingClient *gophercloud.ServiceClient, subnetPoolID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		log.Printf("[DEBUG] Attempting to delete OpenStack Subnetpool %s.\n", subnetPoolID)

		s, err := subnetpools.Get(networkingClient, subnetPoolID).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				log.Printf("[DEBUG] Successfully deleted OpenStack Subnetpool %s", subnetPoolID)
				return s, "DELETED", nil
			}
			return s, "ACTIVE", err
		}

		err = subnetpools.Delete(networkingClient, subnetPoolID).ExtractErr()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				log.Printf("[DEBUG] Successfully deleted OpenStack Subnetpool %s", subnetPoolID)
				return s, "DELETED", nil
			}
			if errCode, ok := err.(gophercloud.ErrUnexpectedResponseCode); ok {
				if errCode.Actual == 409 {
					return s, "ACTIVE", nil
				}
			}
			return s, "ACTIVE", err
		}

		log.Printf("[DEBUG] OpenStack Subnetpool %s still active.\n", subnetPoolID)
		return s, "ACTIVE", nil
	}
}

func resourceSubnetPoolPrefixesV2(d *schema.ResourceData) []string {
	rawPrefixes := d.Get("prefixes").([]interface{})
	prefixes := make([]string, len(rawPrefixes))
	for i, rawPrefix := range rawPrefixes {
		prefixes[i] = rawPrefix.(string)
	}
	return prefixes
}
