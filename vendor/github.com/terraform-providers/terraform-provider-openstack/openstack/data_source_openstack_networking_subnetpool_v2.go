package openstack

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/subnetpools"
)

func dataSourceNetworkingSubnetPoolV2() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNetworkingSubnetPoolV2Read,
		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: false,
			},

			"default_quota": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ForceNew: false,
			},

			"project_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: false,
			},

			"updated_at": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: false,
			},

			"prefixes": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
				ForceNew: false,
			},

			"default_prefixlen": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ForceNew: false,
			},

			"min_prefixlen": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ForceNew: false,
			},

			"max_prefixlen": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ForceNew: false,
			},

			"address_scope_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
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
				Computed: true,
				ForceNew: false,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: false,
			},

			"is_default": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
				ForceNew: false,
			},

			"revision_number": {
				Type:     schema.TypeInt,
				Computed: true,
				ForceNew: false,
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

func dataSourceNetworkingSubnetPoolV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	listOpts := subnetpools.ListOpts{}

	if v, ok := d.GetOk("name"); ok {
		listOpts.Name = v.(string)
	}

	if v, ok := d.GetOk("default_quota"); ok {
		listOpts.DefaultQuota = v.(int)
	}

	if v, ok := d.GetOk("project_id"); ok {
		listOpts.ProjectID = v.(string)
	}

	if v, ok := d.GetOk("default_prefixlen"); ok {
		listOpts.DefaultPrefixLen = v.(int)
	}

	if v, ok := d.GetOk("min_prefixlen"); ok {
		listOpts.MinPrefixLen = v.(int)
	}

	if v, ok := d.GetOk("max_prefixlen"); ok {
		listOpts.MaxPrefixLen = v.(int)
	}

	if v, ok := d.GetOk("address_scope_id"); ok {
		listOpts.AddressScopeID = v.(string)
	}

	if v, ok := d.GetOk("ip_version"); ok {
		listOpts.IPVersion = v.(int)
	}

	if v, ok := d.GetOk("shared"); ok {
		shared := v.(bool)
		listOpts.Shared = &shared
	}

	if v, ok := d.GetOk("description"); ok {
		listOpts.Description = v.(string)
	}

	if v, ok := d.GetOk("is_default"); ok {
		isDefault := v.(bool)
		listOpts.IsDefault = &isDefault
	}

	tags := networkV2AttributesTags(d)
	if len(tags) > 0 {
		listOpts.Tags = strings.Join(tags, ",")
	}

	pages, err := subnetpools.List(networkingClient, listOpts).AllPages()
	if err != nil {
		return fmt.Errorf("Unable to retrieve openstack_networking_subnetpool_v2: %s", err)
	}

	allSubnetPools, err := subnetpools.ExtractSubnetPools(pages)
	if err != nil {
		return fmt.Errorf("Unable to extract openstack_networking_subnetpool_v2: %s", err)
	}

	if len(allSubnetPools) < 1 {
		return fmt.Errorf("Your query returned no openstack_networking_subnetpool_v2. " +
			"Please change your search criteria and try again.")
	}

	if len(allSubnetPools) > 1 {
		return fmt.Errorf("Your query returned more than one openstack_networking_subnetpool_v2." +
			" Please try a more specific search criteria")
	}

	subnetPool := allSubnetPools[0]

	log.Printf("[DEBUG] Retrieved openstack_networking_subnetpool_v2 %s: %+v", subnetPool.ID, subnetPool)
	d.SetId(subnetPool.ID)

	d.Set("name", subnetPool.Name)
	d.Set("default_quota", subnetPool.DefaultQuota)
	d.Set("project_id", subnetPool.ProjectID)
	d.Set("default_prefixlen", subnetPool.DefaultPrefixLen)
	d.Set("min_prefixlen", subnetPool.MinPrefixLen)
	d.Set("max_prefixlen", subnetPool.MaxPrefixLen)
	d.Set("address_scope_id", subnetPool.AddressScopeID)
	d.Set("ip_version", subnetPool.IPversion)
	d.Set("shared", subnetPool.Shared)
	d.Set("is_default", subnetPool.IsDefault)
	d.Set("description", subnetPool.Description)
	d.Set("revision_number", subnetPool.RevisionNumber)
	d.Set("all_tags", subnetPool.Tags)
	d.Set("region", GetRegion(d, config))

	if err := d.Set("created_at", subnetPool.CreatedAt.Format(time.RFC3339)); err != nil {
		log.Printf("[DEBUG] Unable to set openstack_networking_subnetpool_v2 created_at: %s", err)
	}
	if err := d.Set("updated_at", subnetPool.UpdatedAt.Format(time.RFC3339)); err != nil {
		log.Printf("[DEBUG] Unable to set openstack_networking_subnetpool_v2 updated_at: %s", err)
	}
	if err := d.Set("prefixes", subnetPool.Prefixes); err != nil {
		log.Printf("[WARN] unable to set prefixes: %s", err)
	}

	return nil
}
