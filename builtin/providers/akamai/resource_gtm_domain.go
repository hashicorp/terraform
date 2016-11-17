package akamai

import (
	"log"

	"github.com/Comcast/go-edgegrid/edgegrid"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAkamaiGTMDomain() *schema.Resource {
	return &schema.Resource{
		Create: resourceGTMDomainCreate,
		Read:   resourceGTMDomainRead,
		Update: resourceGTMDomainUpdate,
		Delete: resourceGTMDomainDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceGTMDomainCreate(d *schema.ResourceData, meta interface{}) error {
	name := d.Get("name").(string)
	log.Printf("[INFO] Creating GTM domain: %s", name)
	created, err := meta.(*Clients).GTM.DomainCreate(name, d.Get("type").(string))
	if err != nil {
		return err
	}

	d.SetId(created.Domain.Name)

	return resourceGTMDomainRead(d, meta)
}

func resourceGTMDomainRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Reading GTM domain: %s", d.Id())
	domain, err := meta.(*Clients).GTM.Domain(d.Id())
	if err != nil {
		return err
	}

	d.Set("name", domain.Name)

	return nil
}

func resourceGTMDomainUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Updating GTM domain: %s", d.Id())
	_, err := meta.(*Clients).GTM.DomainUpdate(&edgegrid.Domain{
		Name: d.Get("name").(string),
		Type: d.Get("type").(string),
	})
	if err != nil {
		return err
	}

	return resourceGTMDomainRead(d, meta)
}

// NOTE: this 403s due to Akamai's permissions policy
func resourceGTMDomainDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Deleting domain: %s", d.Id())
	err := meta.(*Clients).GTM.DomainDelete(d.Get("name").(string))
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}
