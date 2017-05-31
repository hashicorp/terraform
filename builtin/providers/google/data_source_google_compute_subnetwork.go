package google

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/googleapi"
)

func dataSourceGoogleComputeSubnetwork() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceGoogleComputeSubnetworkRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"ip_cidr_range": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"private_ip_google_access": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},
			"network": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"gateway_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func dataSourceGoogleComputeSubnetworkRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}
	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	subnetwork, err := config.clientCompute.Subnetworks.Get(
		project, region, d.Get("name").(string)).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			// The resource doesn't exist anymore

			return fmt.Errorf("Subnetwork Not Found")
		}

		return fmt.Errorf("Error reading Subnetwork: %s", err)
	}

	d.Set("ip_cidr_range", subnetwork.IpCidrRange)
	d.Set("private_ip_google_access", subnetwork.PrivateIpGoogleAccess)
	d.Set("self_link", subnetwork.SelfLink)
	d.Set("description", subnetwork.Description)
	d.Set("gateway_address", subnetwork.GatewayAddress)
	d.Set("network", subnetwork.Network)

	//Subnet id creation is defined in resource_compute_subnetwork.go
	subnetwork.Region = region
	d.SetId(createSubnetID(subnetwork))
	return nil
}
