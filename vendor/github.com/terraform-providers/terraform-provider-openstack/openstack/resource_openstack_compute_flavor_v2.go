package openstack

import (
	"fmt"
	"log"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceComputeFlavorV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeFlavorV2Create,
		Read:   resourceComputeFlavorV2Read,
		Update: resourceComputeFlavorV2Update,
		Delete: resourceComputeFlavorV2Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"ram": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"vcpus": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"disk": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"swap": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"rx_tx_factor": &schema.Schema{
				Type:     schema.TypeFloat,
				Optional: true,
				ForceNew: true,
				Default:  1,
			},
			"is_public": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"ephemeral": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"extra_specs": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceComputeFlavorV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	disk := d.Get("disk").(int)
	swap := d.Get("swap").(int)
	isPublic := d.Get("is_public").(bool)
	ephemeral := d.Get("ephemeral").(int)
	createOpts := flavors.CreateOpts{
		Name:       d.Get("name").(string),
		RAM:        d.Get("ram").(int),
		VCPUs:      d.Get("vcpus").(int),
		Disk:       &disk,
		Swap:       &swap,
		RxTxFactor: d.Get("rx_tx_factor").(float64),
		IsPublic:   &isPublic,
		Ephemeral:  &ephemeral,
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	fl, err := flavors.Create(computeClient, &createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack flavor: %s", err)
	}

	d.SetId(fl.ID)

	extraSpecsRaw := d.Get("extra_specs").(map[string]interface{})
	if len(extraSpecsRaw) > 0 {
		extraSpecs := make(flavors.ExtraSpecsOpts, len(extraSpecsRaw))
		for k, v := range extraSpecsRaw {
			extraSpecs[k] = v.(string)
		}

		_, err := flavors.CreateExtraSpecs(computeClient, fl.ID, extraSpecs).Extract()
		if err != nil {
			return fmt.Errorf("Error creating extra specs for flavor %s: %s", fl.ID, err)
		}
	}

	return resourceComputeFlavorV2Read(d, meta)
}

func resourceComputeFlavorV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	fl, err := flavors.Get(computeClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "flavor")
	}

	d.Set("name", fl.Name)
	d.Set("ram", fl.RAM)
	d.Set("vcpus", fl.VCPUs)
	d.Set("disk", fl.Disk)
	d.Set("swap", fl.Swap)
	d.Set("rx_tx_factor", fl.RxTxFactor)
	d.Set("is_public", fl.IsPublic)
	// d.Set("ephemeral", fl.Ephemeral) TODO: Implement this in gophercloud
	d.Set("region", GetRegion(d, config))

	es, err := flavors.ListExtraSpecs(computeClient, d.Id()).Extract()
	if err != nil {
		return fmt.Errorf("Error reading extra specs for flavor %s: %s", d.Id(), err)
	}

	if err := d.Set("extra_specs", es); err != nil {
		log.Printf("[WARN] Unable to set extra specs for flavor %s: %s", d.Id(), err)
	}

	return nil
}

func resourceComputeFlavorV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	if d.HasChange("extra_specs") {
		oldES, newES := d.GetChange("extra_specs")

		// Delete all old extra specs.
		for oldKey, _ := range oldES.(map[string]interface{}) {
			if err := flavors.DeleteExtraSpec(computeClient, d.Id(), oldKey).ExtractErr(); err != nil {
				return fmt.Errorf("Error deleting extra spec %s from flavor %s: %s", oldKey, d.Id(), err)
			}
		}

		// Add new extra specs.
		newESRaw := newES.(map[string]interface{})
		if len(newESRaw) > 0 {
			extraSpecs := make(flavors.ExtraSpecsOpts, len(newESRaw))
			for k, v := range newESRaw {
				extraSpecs[k] = v.(string)
			}

			_, err := flavors.CreateExtraSpecs(computeClient, d.Id(), extraSpecs).Extract()
			if err != nil {
				return fmt.Errorf("Error creating extra specs for flavor %s: %s", d.Id(), err)
			}
		}
	}

	return resourceComputeFlavorV2Read(d, meta)
}

func resourceComputeFlavorV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	computeClient, err := config.computeV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack compute client: %s", err)
	}

	err = flavors.Delete(computeClient, d.Id()).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack flavor: %s", err)
	}
	d.SetId("")
	return nil
}
