package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceGoogleComputeSubnetworks() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceGoogleComputeSubnetworksRead,
		Schema: map[string]*schema.Schema{
			"project": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"region": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"network": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"names": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"gateway_addresses": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"ip_cidr_ranges": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"self_links": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceGoogleComputeSubnetworksRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project := config.Project
	if p, ok := d.GetOk("project"); ok {
		project = p.(string)
		log.Printf("[DEBUG] Use specified project %s", project)
	} else {
		log.Printf("[DEBUG] Use provider's default project %s", project)
	}

	region := config.Region
	if r, ok := d.GetOk("region"); ok {
		region = r.(string)
		log.Printf("[DEBUG] Use specified region %s", region)
	} else {
		log.Printf("[DEBUG] Use provider's default region %s", region)
	}

	call := config.clientCompute.Subnetworks.List(project, region)
	if n, ok := d.GetOk("network"); ok {
		log.Printf("[DEBUG] Filter by network %s", n)
		filter := fmt.Sprintf("(network eq https://www.googleapis.com/compute/v1/projects/%s/global/networks/%s)", project, n)
		call = call.Filter(filter)
	}

	subnets, err := call.Do()
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Fetched subnetwork list of %d items", len(subnets.Items))

	d.SetId(subnets.Id)

	names := make([]string, len(subnets.Items))
	gateway_addresses := make([]string, len(subnets.Items))
	ip_cidr_ranges := make([]string, len(subnets.Items))
	self_links := make([]string, len(subnets.Items))
	for i, item := range subnets.Items {
		names[i] = item.Name
		gateway_addresses[i] = item.GatewayAddress
		ip_cidr_ranges[i] = item.IpCidrRange
		self_links[i] = item.SelfLink
	}
	d.Set("names", names)
	d.Set("gateway_addresses", gateway_addresses)
	d.Set("ip_cidr_ranges", ip_cidr_ranges)
	d.Set("self_links", self_links)

	return nil
}
