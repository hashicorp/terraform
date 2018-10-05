package openstack

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
)

func dataSourceNetworkingNetworkV2() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNetworkingNetworkV2Read,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"network_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"status": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"matching_subnet_cidr": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"tenant_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["tenant_id"],
			},
			"admin_state_up": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"shared": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"external": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"availability_zone_hints": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceNetworkingNetworkV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))

	var listOpts networks.ListOptsBuilder

	var status string
	if v, ok := d.GetOk("status"); ok {
		status = v.(string)
	}

	listOpts = networks.ListOpts{
		ID:       d.Get("network_id").(string),
		Name:     d.Get("name").(string),
		TenantID: d.Get("tenant_id").(string),
		Status:   status,
	}

	if v, ok := d.GetOkExists("external"); ok {
		isExternal := v.(bool)
		listOpts = external.ListOptsExt{
			ListOptsBuilder: listOpts,
			External:        &isExternal,
		}
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
	}
	var allNetworks []networkWithExternalExt
	err = networks.ExtractNetworksInto(pages, &allNetworks)
	if err != nil {
		return fmt.Errorf("Unable to retrieve networks: %s", err)
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
					return fmt.Errorf("Unable to retrieve network subnet: %s", err)
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
		log.Printf("[DEBUG] Unable to set availability_zone_hints: %s", err)
	}

	log.Printf("[DEBUG] Retrieved Network %s: %+v", network.ID, network)
	d.SetId(network.ID)

	d.Set("name", network.Name)
	d.Set("admin_state_up", strconv.FormatBool(network.AdminStateUp))
	d.Set("shared", strconv.FormatBool(network.Shared))
	d.Set("external", network.External)
	d.Set("tenant_id", network.TenantID)
	d.Set("region", GetRegion(d, config))

	return nil
}
