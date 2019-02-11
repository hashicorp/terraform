package openstack

import (
	"fmt"
	"log"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/sharedfilesystems/v2/sharenetworks"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceSharedFilesystemShareNetworkV2() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceSharedFilesystemShareNetworkV2Read,

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"project_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"neutron_net_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"neutron_subnet_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"security_service_id": {
				Type:     schema.TypeString,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"security_service_ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"network_type": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"segmentation_id": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"cidr": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ip_version": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func dataSourceSharedFilesystemShareNetworkV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	sfsClient, err := config.sharedfilesystemV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack sharedfilesystem sfsClient: %s", err)
	}

	listOpts := sharenetworks.ListOpts{
		Name:            d.Get("name").(string),
		Description:     d.Get("description").(string),
		ProjectID:       d.Get("project_id").(string),
		NeutronNetID:    d.Get("neutron_net_id").(string),
		NeutronSubnetID: d.Get("neutron_subnet_id").(string),
		NetworkType:     d.Get("network_type").(string),
	}

	if v, ok := d.GetOkExists("ip_version"); ok {
		listOpts.IPVersion = gophercloud.IPVersion(v.(int))
	}

	if v, ok := d.GetOkExists("segmentation_id"); ok {
		listOpts.SegmentationID = v.(int)
	}

	allPages, err := sharenetworks.ListDetail(sfsClient, listOpts).AllPages()
	if err != nil {
		return fmt.Errorf("Unable to query share networks: %s", err)
	}

	allShareNetworks, err := sharenetworks.ExtractShareNetworks(allPages)
	if err != nil {
		return fmt.Errorf("Unable to retrieve share networks: %s", err)
	}

	if len(allShareNetworks) < 1 {
		return fmt.Errorf("Your query returned no results. " +
			"Please change your search criteria and try again.")
	}

	var securityServiceID string
	var securityServiceIDs []string
	if v, ok := d.GetOkExists("security_service_id"); ok {
		// filtering by "security_service_id"
		securityServiceID = v.(string)
		var filteredShareNetworks []sharenetworks.ShareNetwork

		log.Printf("[DEBUG] Filtering share networks by a %s security service ID", securityServiceID)
		for _, shareNetwork := range allShareNetworks {
			tmp, err := resourceSharedFilesystemShareNetworkV2GetSvcByShareNetID(sfsClient, shareNetwork.ID)
			if err != nil {
				return err
			}
			if strSliceContains(tmp, securityServiceID) {
				filteredShareNetworks = append(filteredShareNetworks, shareNetwork)
				securityServiceIDs = tmp
			}
		}

		if len(filteredShareNetworks) == 0 {
			return fmt.Errorf("Your query returned no results after the security service ID filter. " +
				"Please change your search criteria and try again.")
		}
		allShareNetworks = filteredShareNetworks
	}

	var shareNetwork sharenetworks.ShareNetwork
	if len(allShareNetworks) > 1 {
		log.Printf("[DEBUG] Multiple results found: %#v", allShareNetworks)
		return fmt.Errorf("Your query returned more than one result. Please try a more " +
			"specific search criteria.")
	} else {
		shareNetwork = allShareNetworks[0]
	}

	// skip extra calls if "security_service_id" filter was already used
	if securityServiceID == "" {
		securityServiceIDs, err = resourceSharedFilesystemShareNetworkV2GetSvcByShareNetID(sfsClient, shareNetwork.ID)
		if err != nil {
			return err
		}
	}

	d.SetId(shareNetwork.ID)
	d.Set("name", shareNetwork.Name)
	d.Set("description", shareNetwork.Description)
	d.Set("project_id", shareNetwork.ProjectID)
	d.Set("neutron_net_id", shareNetwork.NeutronNetID)
	d.Set("neutron_subnet_id", shareNetwork.NeutronSubnetID)
	d.Set("security_service_ids", securityServiceIDs)
	d.Set("network_type", shareNetwork.NetworkType)
	d.Set("ip_version", shareNetwork.IPVersion)
	d.Set("segmentation_id", shareNetwork.SegmentationID)
	d.Set("cidr", shareNetwork.CIDR)
	d.Set("region", GetRegion(d, config))

	return nil
}
