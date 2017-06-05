package opc

import (
	"fmt"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceOPCIPAddressReservation() *schema.Resource {
	return &schema.Resource{
		Create: resourceOPCIPAddressReservationCreate,
		Read:   resourceOPCIPAddressReservationRead,
		Update: resourceOPCIPAddressReservationUpdate,
		Delete: resourceOPCIPAddressReservationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"ip_address_pool": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(compute.PublicIPAddressPool),
					string(compute.PrivateIPAddressPool),
				}, true),
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"tags": tagsOptionalSchema(),
			"ip_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceOPCIPAddressReservationCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).IPAddressReservations()

	input := compute.CreateIPAddressReservationInput{
		Name:          d.Get("name").(string),
		IPAddressPool: d.Get("ip_address_pool").(string),
	}
	tags := getStringList(d, "tags")
	if len(tags) != 0 {
		input.Tags = tags
	}
	if description, ok := d.GetOk("description"); ok {
		input.Description = description.(string)
	}

	info, err := client.CreateIPAddressReservation(&input)
	if err != nil {
		return fmt.Errorf("Error creating IP Address Reservation: %s", err)
	}
	d.SetId(info.Name)
	return resourceOPCIPAddressReservationRead(d, meta)
}

func resourceOPCIPAddressReservationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).IPAddressReservations()

	getInput := compute.GetIPAddressReservationInput{
		Name: d.Id(),
	}
	result, err := client.GetIPAddressReservation(&getInput)
	if err != nil {
		// IP Address Reservation does not exist
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading ip address reservation %s: %s", d.Id(), err)
	}

	d.Set("name", result.Name)
	d.Set("description", result.Description)
	d.Set("ip_address_pool", result.IPAddressPool)
	d.Set("ip_address", result.IPAddress)
	d.Set("uri", result.Uri)

	if err := setStringList(d, "tags", result.Tags); err != nil {
		return err
	}
	return nil
}

func resourceOPCIPAddressReservationUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).IPAddressReservations()

	input := compute.UpdateIPAddressReservationInput{
		Name:          d.Get("name").(string),
		IPAddressPool: d.Get("ip_address_pool").(string),
	}
	tags := getStringList(d, "tags")
	if len(tags) != 0 {
		input.Tags = tags
	}
	if description, ok := d.GetOk("description"); ok {
		input.Description = description.(string)
	}

	info, err := client.UpdateIPAddressReservation(&input)
	if err != nil {
		return fmt.Errorf("Error updating IP Address Reservation: %s", err)
	}
	d.SetId(info.Name)
	return resourceOPCIPAddressReservationRead(d, meta)
}

func resourceOPCIPAddressReservationDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).IPAddressReservations()
	name := d.Id()

	input := compute.DeleteIPAddressReservationInput{
		Name: name,
	}
	if err := client.DeleteIPAddressReservation(&input); err != nil {
		return fmt.Errorf("Error deleting IP Address Reservation: %+v", err)
	}
	return nil
}
