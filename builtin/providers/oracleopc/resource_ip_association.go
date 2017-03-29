package opc

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/oracle/terraform-provider-compute/sdk/compute"
	"log"
)

func resourceIPAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceIPAssociationCreate,
		Read:   resourceIPAssociationRead,
		Delete: resourceIPAssociationDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"vcable": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"parentpool": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceIPAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())

	vcable, parentpool := getIPAssociationResourceData(d)

	log.Printf("[DEBUG] Creating ip association between vcable %s and parent pool %s",
		vcable, parentpool)

	client := meta.(*OPCClient).IPAssociations()
	info, err := client.CreateIPAssociation(vcable, parentpool)
	if err != nil {
		return fmt.Errorf("Error creating ip association between  vcable %s and parent pool %s: %s",
			vcable, parentpool, err)
	}

	d.SetId(info.Name)
	updateIPAssociationResourceData(d, info)
	return nil
}

func updateIPAssociationResourceData(d *schema.ResourceData, info *compute.IPAssociationInfo) {
	d.Set("name", info.Name)
	d.Set("parentpool", info.ParentPool)
	d.Set("vcable", info.VCable)
}

func resourceIPAssociationRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())
	client := meta.(*OPCClient).IPAssociations()
	name := d.Get("name").(string)

	log.Printf("[DEBUG] Reading state of ip association %s", name)
	result, err := client.GetIPAssociation(name)
	if err != nil {
		// IP Association does not exist
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading ip association %s: %s", name, err)
	}

	log.Printf("[DEBUG] Read state of ip association %s: %#v", name, result)
	updateIPAssociationResourceData(d, result)
	return nil
}

func getIPAssociationResourceData(d *schema.ResourceData) (string, string) {
	return d.Get("vcable").(string), d.Get("parentpool").(string)
}

func resourceIPAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())
	client := meta.(*OPCClient).IPAssociations()
	name := d.Get("name").(string)

	vcable, parentpool := getIPAssociationResourceData(d)
	log.Printf("[DEBUG] Deleting ip association %s between vcable %s and parent pool %s",
		name, vcable, parentpool)

	if err := client.DeleteIPAssociation(name); err != nil {
		return fmt.Errorf("Error deleting ip association %s between vcable %s and parent pool %s: %s",
			name, vcable, parentpool, err)
	}
	return nil
}
