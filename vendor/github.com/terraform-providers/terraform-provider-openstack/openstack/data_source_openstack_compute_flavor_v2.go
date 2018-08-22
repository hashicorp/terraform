package openstack

import (
	"fmt"
	"log"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceComputeFlavorV2() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceComputeFlavorV2Read,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"min_ram": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"ram": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"vcpus": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"min_disk": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"disk": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"swap": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"rx_tx_factor": {
				Type:     schema.TypeFloat,
				Optional: true,
				ForceNew: true,
			},

			// Computed values
			"is_public": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

// dataSourceComputeFlavorV2Read performs the image lookup.
func dataSourceComputeFlavorV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	listOpts := flavors.ListOpts{
		MinDisk:    d.Get("min_disk").(int),
		MinRAM:     d.Get("min_ram").(int),
		AccessType: flavors.PublicAccess,
	}

	log.Printf("[DEBUG] List Options: %#v", listOpts)

	var flavor flavors.Flavor
	allPages, err := flavors.ListDetail(computeClient, listOpts).AllPages()
	if err != nil {
		return fmt.Errorf("Unable to query flavors: %s", err)
	}

	allFlavors, err := flavors.ExtractFlavors(allPages)
	if err != nil {
		return fmt.Errorf("Unable to retrieve flavors: %s", err)
	}

	// Loop through all flavors to find a more specific one.
	if len(allFlavors) > 1 {
		var filteredFlavors []flavors.Flavor
		for _, flavor := range allFlavors {
			if v := d.Get("name").(string); v != "" {
				if flavor.Name != v {
					continue
				}
			}

			// d.GetOk is used because 0 might be a valid choice.
			if v, ok := d.GetOk("ram"); ok {
				if flavor.RAM != v.(int) {
					continue
				}
			}

			if v, ok := d.GetOk("vcpus"); ok {
				if flavor.VCPUs != v.(int) {
					continue
				}
			}

			if v, ok := d.GetOk("disk"); ok {
				if flavor.Disk != v.(int) {
					continue
				}
			}

			if v, ok := d.GetOk("swap"); ok {
				if flavor.Swap != v.(int) {
					continue
				}
			}

			if v, ok := d.GetOk("rx_tx_factor"); ok {
				if flavor.RxTxFactor != v.(float64) {
					continue
				}
			}

			filteredFlavors = append(filteredFlavors, flavor)
		}

		allFlavors = filteredFlavors
	}

	if len(allFlavors) < 1 {
		return fmt.Errorf("Your query returned no results. " +
			"Please change your search criteria and try again.")
	}

	if len(allFlavors) > 1 {
		log.Printf("[DEBUG] Multiple results found: %#v", allFlavors)
		return fmt.Errorf("Your query returned more than one result. " +
			"Please try a more specific search criteria")
	} else {
		flavor = allFlavors[0]
	}

	log.Printf("[DEBUG] Single Flavor found: %s", flavor.ID)
	return dataSourceComputeFlavorV2Attributes(d, &flavor)
}

// dataSourceComputeFlavorV2Attributes populates the fields of an Image resource.
func dataSourceComputeFlavorV2Attributes(d *schema.ResourceData, flavor *flavors.Flavor) error {
	log.Printf("[DEBUG] openstack_compute_flavor_v2 details: %#v", flavor)

	d.SetId(flavor.ID)
	d.Set("name", flavor.Name)
	d.Set("disk", flavor.Disk)
	d.Set("ram", flavor.RAM)
	d.Set("rx_tx_factor", flavor.RxTxFactor)
	d.Set("swap", flavor.Swap)
	d.Set("vcpus", flavor.VCPUs)
	d.Set("is_public", flavor.IsPublic)

	return nil
}
