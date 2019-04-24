package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/attributestags"
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
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},

			"default_quota": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: false,
			},

			"project_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"created_at": {
				Type:     schema.TypeString,
				ForceNew: false,
				Computed: true,
			},

			"updated_at": {
				Type:     schema.TypeString,
				ForceNew: false,
				Computed: true,
			},

			"prefixes": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"default_prefixlen": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: false,
				Computed: true,
			},

			"min_prefixlen": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: false,
				Computed: true,
			},

			"max_prefixlen": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: false,
				Computed: true,
			},

			"address_scope_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"ip_version": {
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

			"shared": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"is_default": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
			},

			"revision_number": {
				Type:     schema.TypeInt,
				ForceNew: false,
				Computed: true,
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
			Prefixes:         expandToStringSlice(d.Get("prefixes").([]interface{})),
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

	log.Printf("[DEBUG] openstack_networking_subnetpool_v2 create options: %#v", createOpts)
	s, err := subnetpools.Create(networkingClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating openstack_networking_subnetpool_v2: %s", err)
	}

	log.Printf("[DEBUG] Waiting for openstack_networking_subnetpool_v2 %s to become available.", s.ID)

	stateConf := &resource.StateChangeConf{
		Target:     []string{"ACTIVE"},
		Refresh:    networkingSubnetpoolV2StateRefreshFunc(networkingClient, s.ID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for openstack_networking_subnetpool_v2 %s to become available: %s", s.ID, err)
	}

	d.SetId(s.ID)

	tags := networkV2AttributesTags(d)
	if len(tags) > 0 {
		tagOpts := attributestags.ReplaceAllOpts{Tags: tags}
		tags, err := attributestags.ReplaceAll(networkingClient, "subnetpools", s.ID, tagOpts).Extract()
		if err != nil {
			return fmt.Errorf("Error setting tags on openstack_networking_subnetpool_v2 %s: %s", s.ID, err)
		}
		log.Printf("[DEBUG] Set tags %s on openstack_networking_subnetpool_v2 %s", tags, s.ID)
	}

	log.Printf("[DEBUG] Created openstack_networking_subnetpool_v2 %s: %#v", s.ID, s)
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
		return CheckDeleted(d, err, "Error getting openstack_networking_subnetpool_v2")
	}

	log.Printf("[DEBUG] Retrieved openstack_networking_subnetpool_v2 %s: %#v", d.Id(), s)

	d.Set("name", s.Name)
	d.Set("default_quota", s.DefaultQuota)
	d.Set("project_id", s.ProjectID)
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

	networkV2ReadAttributesTags(d, s.Tags)

	if err := d.Set("created_at", s.CreatedAt.Format(time.RFC3339)); err != nil {
		log.Printf("[DEBUG] Unable to set openstack_networking_subnetpool_v2 created_at: %s", err)
	}
	if err := d.Set("updated_at", s.UpdatedAt.Format(time.RFC3339)); err != nil {
		log.Printf("[DEBUG] Unable to set openstack_networking_subnetpool_v2 updated_at: %s", err)
	}
	if err := d.Set("prefixes", s.Prefixes); err != nil {
		log.Printf("[DEBUG] Unable to set openstack_networking_subnetpool_v2 prefixes: %s", err)
	}

	return nil
}

func resourceNetworkingSubnetPoolV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	var hasChange bool
	var updateOpts subnetpools.UpdateOpts

	if d.HasChange("name") {
		hasChange = true
		updateOpts.Name = d.Get("name").(string)
	}

	if d.HasChange("default_quota") {
		hasChange = true
		v := d.Get("default_quota").(int)
		updateOpts.DefaultQuota = &v
	}

	if d.HasChange("project_id") {
		hasChange = true
		updateOpts.ProjectID = d.Get("project_id").(string)
	}

	if d.HasChange("prefixes") {
		hasChange = true
		updateOpts.Prefixes = expandToStringSlice(d.Get("prefixes").([]interface{}))
	}

	if d.HasChange("default_prefixlen") {
		hasChange = true
		updateOpts.DefaultPrefixLen = d.Get("default_prefixlen").(int)
	}

	if d.HasChange("min_prefixlen") {
		hasChange = true
		updateOpts.MinPrefixLen = d.Get("min_prefixlen").(int)
	}

	if d.HasChange("max_prefixlen") {
		hasChange = true
		updateOpts.MaxPrefixLen = d.Get("max_prefixlen").(int)
	}

	if d.HasChange("address_scope_id") {
		hasChange = true
		v := d.Get("address_scope_id").(string)
		updateOpts.AddressScopeID = &v
	}

	if d.HasChange("description") {
		hasChange = true
		v := d.Get("description").(string)
		updateOpts.Description = &v
	}

	if d.HasChange("is_default") {
		hasChange = true
		v := d.Get("is_default").(bool)
		updateOpts.IsDefault = &v
	}

	if hasChange {
		log.Printf("[DEBUG] openstack_networking_subnetpool_v2 %s update options: %#v", d.Id(), updateOpts)
		_, err = subnetpools.Update(networkingClient, d.Id(), updateOpts).Extract()
		if err != nil {
			return fmt.Errorf("Error updating openstack_networking_subnetpool_v2 %s: %s", d.Id(), err)
		}
	}

	if d.HasChange("tags") {
		tags := networkV2UpdateAttributesTags(d)
		tagOpts := attributestags.ReplaceAllOpts{Tags: tags}
		tags, err := attributestags.ReplaceAll(networkingClient, "subnetpools", d.Id(), tagOpts).Extract()
		if err != nil {
			return fmt.Errorf("Error setting tags on openstack_networking_subnetpool_v2 %s: %s", d.Id(), err)
		}
		log.Printf("[DEBUG] Set tags %s on openstack_networking_subnetpool_v2 %s", tags, d.Id())
	}

	return resourceNetworkingSubnetPoolV2Read(d, meta)
}

func resourceNetworkingSubnetPoolV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	if err := subnetpools.Delete(networkingClient, d.Id()).ExtractErr(); err != nil {
		return CheckDeleted(d, err, "Error deleting openstack_networking_subnetpool_v2")
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE"},
		Target:     []string{"DELETED"},
		Refresh:    networkingSubnetpoolV2StateRefreshFunc(networkingClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for openstack_networking_subnetpool_v2 %s to delete: %s", d.Id(), err)
	}

	return nil
}
