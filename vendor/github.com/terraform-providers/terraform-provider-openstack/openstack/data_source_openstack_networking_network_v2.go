package openstack

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/vlantransparent"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
)

func dataSourceNetworkingNetworkV2() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNetworkingNetworkV2Read,

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"network_id": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"status": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"matching_subnet_cidr": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"tenant_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["tenant_id"],
			},

			"admin_state_up": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"shared": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"external": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"availability_zone_hints": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"transparent_vlan": {
				Type:     schema.TypeBool,
				Optional: true,
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

func dataSourceNetworkingNetworkV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	// Prepare basic listOpts.
	var listOpts networks.ListOptsBuilder

	var status string
	if v, ok := d.GetOk("status"); ok {
		status = v.(string)
	}

	listOpts = networks.ListOpts{
		ID:          d.Get("network_id").(string),
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		TenantID:    d.Get("tenant_id").(string),
		Status:      status,
	}

	// Add the external attribute if specified.
	if v, ok := d.GetOkExists("external"); ok {
		isExternal := v.(bool)
		listOpts = external.ListOptsExt{
			ListOptsBuilder: listOpts,
			External:        &isExternal,
		}
	}

	// Add the transparent VLAN attribute if specified.
	if v, ok := d.GetOkExists("transparent_vlan"); ok {
		isVLANTransparent := v.(bool)
		listOpts = vlantransparent.ListOptsExt{
			ListOptsBuilder: listOpts,
			VLANTransparent: &isVLANTransparent,
		}
	}

	tags := networkV2AttributesTags(d)
	if len(tags) > 0 {
		listOpts = networks.ListOpts{Tags: strings.Join(tags, ",")}
	}

	pages, err := networks.List(networkingClient, listOpts).AllPages()
	if err != nil {
		return err
	}

	// First extract into a normal networks.Network in order to see if
	// there were any results at all.
	tmpAllNetworks, err := networks.ExtractNetworks(pages)
	if err != nil {
		return err
	}

	if len(tmpAllNetworks) < 1 {
		return fmt.Errorf("Your query returned no results. " +
			"Please change your search criteria and try again.")
	}

	type networkWithExternalExt struct {
		networks.Network
		external.NetworkExternalExt
		vlantransparent.TransparentExt
	}
	var allNetworks []networkWithExternalExt
	err = networks.ExtractNetworksInto(pages, &allNetworks)
	if err != nil {
		return fmt.Errorf("Unable to retrieve openstack_networking_networks_v2: %s", err)
	}

	var refinedNetworks []networkWithExternalExt
	if cidr := d.Get("matching_subnet_cidr").(string); cidr != "" {
		for _, n := range allNetworks {
			for _, s := range n.Subnets {
				subnet, err := subnets.Get(networkingClient, s).Extract()
				if err != nil {
					if _, ok := err.(gophercloud.ErrDefault404); ok {
						continue
					}
					return fmt.Errorf("Unable to retrieve openstack_networking_network_v2 subnet: %s", err)
				}
				if cidr == subnet.CIDR {
					refinedNetworks = append(refinedNetworks, n)
				}
			}
		}
	} else {
		refinedNetworks = allNetworks
	}

	if len(refinedNetworks) < 1 {
		return fmt.Errorf("Your query returned no results. " +
			"Please change your search criteria and try again.")
	}

	if len(refinedNetworks) > 1 {
		return fmt.Errorf("Your query returned more than one result." +
			" Please try a more specific search criteria")
	}

	network := refinedNetworks[0]

	if err = d.Set("availability_zone_hints", network.AvailabilityZoneHints); err != nil {
		log.Printf("[DEBUG] Unable to set availability_zone_hints for openstack_networking_network_v2 %s: %s", network.ID, err)
	}

	log.Printf("[DEBUG] Retrieved openstack_networking_network_v2 %s: %+v", network.ID, network)
	d.SetId(network.ID)

	d.Set("name", network.Name)
	d.Set("description", network.Description)
	d.Set("admin_state_up", strconv.FormatBool(network.AdminStateUp))
	d.Set("shared", strconv.FormatBool(network.Shared))
	d.Set("external", network.External)
	d.Set("tenant_id", network.TenantID)
	d.Set("transparent_vlan", network.VLANTransparent)
	d.Set("all_tags", network.Tags)
	d.Set("region", GetRegion(d, config))

	return nil
}
