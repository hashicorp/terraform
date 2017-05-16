package opc

import (
	"fmt"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceOPCIPAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceOPCIPAssociationCreate,
		Read:   resourceOPCIPAssociationRead,
		Delete: resourceOPCIPAssociationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"vcable": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"parent_pool": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceOPCIPAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	vCable := d.Get("vcable").(string)
	parentPool := d.Get("parent_pool").(string)

	client := meta.(*compute.Client).IPAssociations()
	input := compute.CreateIPAssociationInput{
		ParentPool: parentPool,
		VCable:     vCable,
	}
	info, err := client.CreateIPAssociation(&input)
	if err != nil {
		return fmt.Errorf("Error creating ip association between vcable %s and parent pool %s: %s", vCable, parentPool, err)
	}

	d.SetId(info.Name)

	return resourceOPCIPAssociationRead(d, meta)
}

func resourceOPCIPAssociationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).IPAssociations()

	name := d.Id()
	input := compute.GetIPAssociationInput{
		Name: name,
	}
	result, err := client.GetIPAssociation(&input)
	if err != nil {
		// IP Association does not exist
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading ip association '%s': %s", name, err)
	}

	d.Set("name", result.Name)
	d.Set("parent_pool", result.ParentPool)
	d.Set("vcable", result.VCable)

	return nil
}

func resourceOPCIPAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).IPAssociations()

	name := d.Id()
	input := compute.DeleteIPAssociationInput{
		Name: name,
	}
	if err := client.DeleteIPAssociation(&input); err != nil {
		return fmt.Errorf("Error deleting ip association '%s': %s", name, err)
	}

	return nil
}
