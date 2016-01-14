package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func resourceComputeGlobalAddress() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeGlobalAddressCreate,
		Read:   resourceComputeGlobalAddressRead,
		Delete: resourceComputeGlobalAddressDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"address": &schema.Schema{
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

func resourceComputeGlobalAddressCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Build the address parameter
	addr := &compute.Address{Name: d.Get("name").(string)}
	op, err := config.clientCompute.GlobalAddresses.Insert(
		config.Project, addr).Do()
	if err != nil {
		return fmt.Errorf("Error creating address: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(addr.Name)

	err = computeOperationWaitGlobal(config, op, "Creating Global Address")
	if err != nil {
		return err
	}

	return resourceComputeGlobalAddressRead(d, meta)
}

func resourceComputeGlobalAddressRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	addr, err := config.clientCompute.GlobalAddresses.Get(
		config.Project, d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing Global Address %q because it's gone", d.Get("name").(string))
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading address: %s", err)
	}

	d.Set("address", addr.Address)
	d.Set("self_link", addr.SelfLink)

	return nil
}

func resourceComputeGlobalAddressDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Delete the address
	log.Printf("[DEBUG] address delete request")
	op, err := config.clientCompute.GlobalAddresses.Delete(
		config.Project, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting address: %s", err)
	}

	err = computeOperationWaitGlobal(config, op, "Deleting Global Address")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
