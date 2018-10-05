package openstack

import (
	"fmt"
	"log"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/endpoints"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/services"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceIdentityEndpointV3() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceIdentityEndpointV3Read,

		Schema: map[string]*schema.Schema{
			"service_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"service_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"interface": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "public",
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if value != "public" && value != "internal" && value != "admin" {
						errors = append(errors, fmt.Errorf(
							"Only 'public', 'internal', 'public'  are supported values for 'interface'"))
					}
					return
				},
			},
			"url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
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

	log.Printf("[DEBUG] List Options: %#v", listOpts)

	var endpoint endpoints.Endpoint
	allPages, err := endpoints.List(identityClient, listOpts).AllPages()
	if err != nil {
		return fmt.Errorf("Unable to query endpoints: %s", err)
	}

	allEndpoints, err := endpoints.ExtractEndpoints(allPages)
	if err != nil {
		return fmt.Errorf("Unable to retrieve endpoints: %s", err)
	}

	serviceName := d.Get("service_name").(string)
	if len(allEndpoints) > 1 && serviceName != "" {
		var filteredEndpoints []endpoints.Endpoint
		// Query all services to further filter results
		allServicePages, err := services.List(identityClient, nil).AllPages()
		if err != nil {
			return fmt.Errorf("Unable to query services: %s", err)
		}

		allServices, err := services.ExtractServices(allServicePages)
		if err != nil {
			return fmt.Errorf("Unable to retrieve services: %s", err)
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
		return fmt.Errorf("Your query returned no results. " +
			"Please change your search criteria and try again.")
	}

	if len(allEndpoints) > 1 {
		log.Printf("[DEBUG] Multiple results found: %#v", allEndpoints)
		return fmt.Errorf("Your query returned more than one result")
	}
	endpoint = allEndpoints[0]

	log.Printf("[DEBUG] Single endpoint found: %s", endpoint.ID)
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
