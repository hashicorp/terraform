package google

import (
	"fmt"
	"log"

	"strings"

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
			"ip_cidr_range": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"network": &schema.Schema{
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

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
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

func splitSubnetID(id string) (region string, name string) {
	parts := strings.Split(id, "/")
	region = parts[0]
	name = parts[1]
	return
}

func resourceComputeSubnetworkCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Build the subnetwork parameters
	subnetwork := &compute.Subnetwork{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		IpCidrRange: d.Get("ip_cidr_range").(string),
		Network:     d.Get("network").(string),
	}

	log.Printf("[DEBUG] Subnetwork insert request: %#v", subnetwork)
	op, err := config.clientCompute.Subnetworks.Insert(
		project, region, subnetwork).Do()

	if err != nil {
		return fmt.Errorf("Error creating subnetwork: %s", err)
	}

	// It probably maybe worked, so store the ID now. ID is a combination of region + subnetwork
	// name because subnetwork names are not unique in a project, per the Google docs:
	// "When creating a new subnetwork, its name has to be unique in that project for that region, even across networks.
	// The same name can appear twice in a project, as long as each one is in a different region."
	// https://cloud.google.com/compute/docs/subnetworks
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

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	name := d.Get("name").(string)

	subnetwork, err := config.clientCompute.Subnetworks.Get(
		project, region, name).Do()
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

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Delete the subnetwork
	op, err := config.clientCompute.Subnetworks.Delete(
		project, region, d.Get("name").(string)).Do()
	if err != nil {
		return fmt.Errorf("Error deleting subnetwork: %s", err)
	}

	err = computeOperationWaitRegion(config, op, region, "Deleting Subnetwork")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
