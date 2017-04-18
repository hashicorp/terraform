package google

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/googleapi"
)

func dataSourceGoogleComputeNetwork() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceGoogleComputeNetworkRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"gateway_ipv4": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"subnetworks_self_links": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceGoogleComputeNetworkRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}
	network, err := config.clientCompute.Networks.Get(
		project, d.Get("name").(string)).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			// The resource doesn't exist anymore

			return fmt.Errorf("Network Not Found : %s", d.Get("name"))
		}

		return fmt.Errorf("Error reading network: %s", err)
	}
	d.Set("gateway_ipv4", network.GatewayIPv4)
	d.Set("self_link", network.SelfLink)
	d.Set("description", network.Description)
	d.Set("subnetworks_self_links", network.Subnetworks)
	d.SetId(network.Name)
	return nil
}
