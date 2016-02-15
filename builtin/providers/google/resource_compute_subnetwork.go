package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func resourceComputeSubnetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeSubnetworkCreate,
		Read:   resourceComputeSubnetworkRead,
		Delete: resourceComputeSubnetworkDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"network": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ip_cidr_range": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"gateway_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func createSubnetID(s *compute.Subnetwork) string {
	return fmt.Sprintf("%s/%s", s.Region, s.Name)
}

func resourceComputeSubnetworkCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Build the subnetwork parameters
	subnetwork := &compute.Subnetwork{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		IpCidrRange: d.Get("ip_cidr_range").(string),
		Network:     d.Get("network").(string),
	}
	region := d.Get("region").(string)

	log.Printf("[DEBUG] Subnetwork insert request: %#v", subnetwork)
	op, err := config.clientCompute.Subnetworks.Insert(
		config.Project, region, subnetwork).Do()

	if err != nil {
		return fmt.Errorf("Error creating subnetwork: %s", err)
	}

	// It probably maybe worked, so store the ID now
	// Subnetwork name is not guaranteed to be unique in a project, but must be unique within a region
	subnetwork.Region = region
	d.SetId(createSubnetID(subnetwork))

	err = computeOperationWaitRegion(config, op, region, "Creating Subnetwork")
	if err != nil {
		return err
	}

	return resourceComputeSubnetworkRead(d, meta)
}

func resourceComputeSubnetworkRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	name := d.Get("name").(string)
	region := d.Get("region").(string)

	subnetwork, err := config.clientCompute.Subnetworks.Get(
		config.Project, region, name).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing Subnetwork %q because it's gone", name)
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading subnetwork: %s", err)
	}

	d.Set("gateway_address", subnetwork.GatewayAddress)
	d.Set("self_link", subnetwork.SelfLink)

	return nil
}

func resourceComputeSubnetworkDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	region := d.Get("region").(string)

	// Delete the network
	op, err := config.clientCompute.Subnetworks.Delete(
		config.Project, region, d.Get("name").(string)).Do()
	if err != nil {
		return fmt.Errorf("Error deleting network: %s", err)
	}

	err = computeOperationWaitRegion(config, op, region, "Deleting Network")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
