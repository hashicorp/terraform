package openstack

import (
	"fmt"
	"log"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/endpoints"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/services"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func dataSourceIdentityEndpointV3() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceIdentityEndpointV3Read,

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"service_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"service_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"interface": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "public",
				ValidateFunc: validation.StringInSlice([]string{
					"public", "internal", "admin",
				}, false),
			},

			"url": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

// dataSourceIdentityEndpointV3Read performs the endpoint lookup.
func dataSourceIdentityEndpointV3Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	identityClient, err := config.identityV3Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack identity client: %s", err)
	}

	availability := gophercloud.AvailabilityPublic
	switch d.Get("interface") {
	case "internal":
		availability = gophercloud.AvailabilityInternal
	case "admin":
		availability = gophercloud.AvailabilityAdmin
	}

	listOpts := endpoints.ListOpts{
		Availability: availability,
		ServiceID:    d.Get("service_id").(string),
	}

	log.Printf("[DEBUG] openstack_identity_endpoint_v3 list options: %#v", listOpts)

	var endpoint endpoints.Endpoint
	allPages, err := endpoints.List(identityClient, listOpts).AllPages()
	if err != nil {
		return fmt.Errorf("Unable to query openstack_identity_endpoint_v3: %s", err)
	}

	allEndpoints, err := endpoints.ExtractEndpoints(allPages)
	if err != nil {
		return fmt.Errorf("Unable to retrieve openstack_identity_endpoint_v3: %s", err)
	}

	serviceName := d.Get("service_name").(string)
	if len(allEndpoints) > 1 && serviceName != "" {
		var filteredEndpoints []endpoints.Endpoint

		// Query all services to further filter results
		allServicePages, err := services.List(identityClient, nil).AllPages()
		if err != nil {
			return fmt.Errorf("Unable to query openstack_identity_endpoint_v3 services: %s", err)
		}

		allServices, err := services.ExtractServices(allServicePages)
		if err != nil {
			return fmt.Errorf("Unable to retrieve openstack_identity_endpoint_v3 services: %s", err)
		}

		for _, endpoint := range allEndpoints {
			for _, service := range allServices {
				if v, ok := service.Extra["name"].(string); ok {
					if endpoint.ServiceID == service.ID && serviceName == v {
						endpoint.Name = v
						filteredEndpoints = append(filteredEndpoints, endpoint)
					}
				}
			}
		}

		allEndpoints = filteredEndpoints
	}

	if len(allEndpoints) < 1 {
		return fmt.Errorf("Your openstack_identity_endpoint_v3 query returned no results. " +
			"Please change your search criteria and try again.")
	}

	if len(allEndpoints) > 1 {
		return fmt.Errorf("Your openstack_identity_endpoint_v3 query returned more than one result")
	}
	endpoint = allEndpoints[0]

	return dataSourceIdentityEndpointV3Attributes(d, &endpoint)
}

// dataSourceIdentityEndpointV3Attributes populates the fields of an Endpoint resource.
func dataSourceIdentityEndpointV3Attributes(d *schema.ResourceData, endpoint *endpoints.Endpoint) error {
	log.Printf("[DEBUG] openstack_identity_endpoint_v3 details: %#v", endpoint)

	d.SetId(endpoint.ID)
	d.Set("interface", endpoint.Availability)
	d.Set("region", endpoint.Region)
	d.Set("service_id", endpoint.ServiceID)
	d.Set("service_name", endpoint.Name)
	d.Set("url", endpoint.URL)

	return nil
}
