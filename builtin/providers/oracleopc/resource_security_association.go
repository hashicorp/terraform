package opc

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/oracle/terraform-provider-compute/sdk/compute"
	"log"
)

func resourceSecurityAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceSecurityAssociationCreate,
		Read:   resourceSecurityAssociationRead,
		Delete: resourceSecurityAssociationDelete,

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

			"seclist": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceSecurityAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())

	vcable, seclist := getSecurityAssociationResourceData(d)

	log.Printf("[DEBUG] Creating security association between vcable %s and security list %s",
		vcable, seclist)

	client := meta.(*OPCClient).SecurityAssociations()
	info, err := client.CreateSecurityAssociation(vcable, seclist)
	if err != nil {
		return fmt.Errorf("Error creating security association between  vcable %s and security list %s: %s",
			vcable, seclist, err)
	}

	d.SetId(info.Name)
	updateSecurityAssociationResourceData(d, info)
	return nil
}

func updateSecurityAssociationResourceData(d *schema.ResourceData, info *compute.SecurityAssociationInfo) {
	d.Set("name", info.Name)
	d.Set("seclist", info.SecList)
	d.Set("vcable", info.VCable)
}

func resourceSecurityAssociationRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())
	client := meta.(*OPCClient).SecurityAssociations()
	name := d.Get("name").(string)

	log.Printf("[DEBUG] Reading state of security association %s", name)
	result, err := client.GetSecurityAssociation(name)
	if err != nil {
		// Security Association does not exist
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading security association %s: %s", name, err)
	}

	log.Printf("[DEBUG] Read state of security association %s: %#v", name, result)
	updateSecurityAssociationResourceData(d, result)
	return nil
}

func getSecurityAssociationResourceData(d *schema.ResourceData) (string, string) {
	return d.Get("vcable").(string), d.Get("seclist").(string)
}

func resourceSecurityAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Resource state: %#v", d.State())
	client := meta.(*OPCClient).SecurityAssociations()
	name := d.Get("name").(string)

	vcable, seclist := getSecurityAssociationResourceData(d)
	log.Printf("[DEBUG] Deleting security association %s between vcable %s and security list %s",
		name, vcable, seclist)

	if err := client.DeleteSecurityAssociation(name); err != nil {
		return fmt.Errorf("Error deleting security association %s between vcable %s and security list %s: %s",
			name, vcable, seclist, err)
	}
	return nil
}
