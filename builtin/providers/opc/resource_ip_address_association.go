package opc

import (
	"fmt"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceOPCIPAddressAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceOPCIPAddressAssociationCreate,
		Read:   resourceOPCIPAddressAssociationRead,
		Update: resourceOPCIPAddressAssociationUpdate,
		Delete: resourceOPCIPAddressAssociationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"ip_address_reservation": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"vnic": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"tags": tagsOptionalSchema(),
			"uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceOPCIPAddressAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).IPAddressAssociations()

	input := compute.CreateIPAddressAssociationInput{
		Name: d.Get("name").(string),
	}

	if ipAddressReservation, ok := d.GetOk("ip_address_reservation"); ok {
		input.IPAddressReservation = ipAddressReservation.(string)
	}

	if vnic, ok := d.GetOk("vnic"); ok {
		input.Vnic = vnic.(string)
	}

	tags := getStringList(d, "tags")
	if len(tags) != 0 {
		input.Tags = tags
	}

	if description, ok := d.GetOk("description"); ok {
		input.Description = description.(string)
	}

	info, err := client.CreateIPAddressAssociation(&input)
	if err != nil {
		return fmt.Errorf("Error creating IP Address Association: %s", err)
	}

	d.SetId(info.Name)
	return resourceOPCIPAddressAssociationRead(d, meta)
}

func resourceOPCIPAddressAssociationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).IPAddressAssociations()
	name := d.Id()

	getInput := compute.GetIPAddressAssociationInput{
		Name: name,
	}
	result, err := client.GetIPAddressAssociation(&getInput)
	if err != nil {
		// IP Address Association does not exist
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading IP Address Association %s: %s", name, err)
	}
	if result == nil {
		d.SetId("")
		return fmt.Errorf("Error reading IP Address Association %s: %s", name, err)
	}

	d.Set("name", result.Name)
	d.Set("ip_address_reservation", result.IPAddressReservation)
	d.Set("vnic", result.Vnic)
	d.Set("description", result.Description)
	d.Set("uri", result.Uri)
	if err := setStringList(d, "tags", result.Tags); err != nil {
		return err
	}
	return nil
}

func resourceOPCIPAddressAssociationUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).IPAddressAssociations()

	input := compute.UpdateIPAddressAssociationInput{
		Name: d.Get("name").(string),
	}

	if ipAddressReservation, ok := d.GetOk("ip_address_reservation"); ok {
		input.IPAddressReservation = ipAddressReservation.(string)
	}

	if vnic, ok := d.GetOk("vnic"); ok {
		input.Vnic = vnic.(string)
	}

	tags := getStringList(d, "tags")
	if len(tags) != 0 {
		input.Tags = tags
	}

	if description, ok := d.GetOk("description"); ok {
		input.Description = description.(string)
	}

	info, err := client.UpdateIPAddressAssociation(&input)
	if err != nil {
		return fmt.Errorf("Error updating IP Address Association: %s", err)
	}

	d.SetId(info.Name)
	return resourceOPCIPAddressAssociationRead(d, meta)
}

func resourceOPCIPAddressAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).IPAddressAssociations()
	name := d.Id()

	input := compute.DeleteIPAddressAssociationInput{
		Name: name,
	}
	if err := client.DeleteIPAddressAssociation(&input); err != nil {
		return fmt.Errorf("Error deleting IP Address Association: %s", err)
	}
	return nil
}
