package opc

import (
	"fmt"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceOPCSecurityAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceOPCSecurityAssociationCreate,
		Read:   resourceOPCSecurityAssociationRead,
		Delete: resourceOPCSecurityAssociationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"vcable": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"seclist": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceOPCSecurityAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).SecurityAssociations()

	name := d.Get("name").(string)
	vcable := d.Get("vcable").(string)
	seclist := d.Get("seclist").(string)

	input := compute.CreateSecurityAssociationInput{
		Name:    name,
		SecList: seclist,
		VCable:  vcable,
	}
	info, err := client.CreateSecurityAssociation(&input)
	if err != nil {
		return fmt.Errorf("Error creating security association between vcable %s and security list %s: %s", vcable, seclist, err)
	}

	d.SetId(info.Name)

	return resourceOPCSecurityAssociationRead(d, meta)
}

func resourceOPCSecurityAssociationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).SecurityAssociations()

	name := d.Id()

	input := compute.GetSecurityAssociationInput{
		Name: name,
	}
	result, err := client.GetSecurityAssociation(&input)
	if err != nil {
		// Security Association does not exist
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading security association %s: %s", name, err)
	}

	d.Set("name", result.Name)
	d.Set("seclist", result.SecList)
	d.Set("vcable", result.VCable)

	return nil
}

func resourceOPCSecurityAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).SecurityAssociations()

	name := d.Id()

	input := compute.DeleteSecurityAssociationInput{
		Name: name,
	}
	if err := client.DeleteSecurityAssociation(&input); err != nil {
		return fmt.Errorf("Error deleting Security Association '%s': %v", name, err)
	}
	return nil
}
