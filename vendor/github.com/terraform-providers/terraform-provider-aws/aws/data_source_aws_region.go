package aws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsRegion() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsRegionRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"current": {
				Type:       schema.TypeBool,
				Optional:   true,
				Computed:   true,
				Deprecated: "Defaults to current provider region if no other filtering is enabled",
			},

			"endpoint": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsRegionRead(d *schema.ResourceData, meta interface{}) error {
	providerRegion := meta.(*AWSClient).region

	var region *endpoints.Region

	if v, ok := d.GetOk("endpoint"); ok {
		endpoint := v.(string)
		matchingRegion, err := findRegionByEc2Endpoint(endpoint)
		if err != nil {
			return err
		}
		region = matchingRegion
	}

	if v, ok := d.GetOk("name"); ok {
		name := v.(string)
		matchingRegion, err := findRegionByName(name)
		if err != nil {
			return err
		}
		if region != nil && region.ID() != matchingRegion.ID() {
			return fmt.Errorf("multiple regions matched; use additional constraints to reduce matches to a single region")
		}
		region = matchingRegion
	}

	// Default to provider current region if no other filters matched
	if region == nil {
		matchingRegion, err := findRegionByName(providerRegion)
		if err != nil {
			return err
		}
		region = matchingRegion
	}

	d.SetId(region.ID())
	d.Set("current", region.ID() == providerRegion)

	regionEndpointEc2, err := region.ResolveEndpoint(endpoints.Ec2ServiceID)
	if err != nil {
		return err
	}
	d.Set("endpoint", strings.TrimPrefix(regionEndpointEc2.URL, "https://"))

	d.Set("name", region.ID())

	d.Set("description", region.Description())

	return nil
}

func findRegionByEc2Endpoint(endpoint string) (*endpoints.Region, error) {
	for _, partition := range endpoints.DefaultPartitions() {
		for _, region := range partition.Regions() {
			regionEndpointEc2, err := region.ResolveEndpoint(endpoints.Ec2ServiceID)
			if err != nil {
				return nil, err
			}
			if strings.TrimPrefix(regionEndpointEc2.URL, "https://") == endpoint {
				return &region, nil
			}
		}
	}
	return nil, fmt.Errorf("region not found for endpoint: %s", endpoint)
}

func findRegionByName(name string) (*endpoints.Region, error) {
	for _, partition := range endpoints.DefaultPartitions() {
		for _, region := range partition.Regions() {
			if region.ID() == name {
				return &region, nil
			}
		}
	}
	return nil, fmt.Errorf("region not found for name: %s", name)
}
