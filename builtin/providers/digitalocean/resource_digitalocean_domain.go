package digitalocean

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/pearkes/digitalocean"
)

func resourceDomain() *schema.Resource {
	return &schema.Resource{
		Create: resourceDomainCreate,
		Read:   resourceDomainRead,
		Delete: resourceDomainDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceDomainCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*ResourceProvider)
	client := p.client

	// Build up our creation options
	opts := digitalocean.CreateDomain{
		Name:      d.Get("name").(string),
		IPAddress: d.Get("ip_address").(string),
	}

	log.Printf("[DEBUG] Domain create configuration: %#v", opts)
	name, err := client.CreateDomain(&opts)
	if err != nil {
		return fmt.Errorf("Error creating Domain: %s", err)
	}

	d.SetId(name)
	log.Printf("[INFO] Domain Name: %s", name)

	return nil
}

func resourceDomainDelete(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*ResourceProvider)
	client := p.client

	log.Printf("[INFO] Deleting Domain: %s", d.Id())
	err := client.DestroyDomain(d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting Domain: %s", err)
	}

	d.SetId("")
	return nil
}

func resourceDomainRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*ResourceProvider)
	client := p.client

	domain, err := client.RetrieveDomain(d.Id())
	if err != nil {
		// If the domain is somehow already destroyed, mark as
		// succesfully gone
		if strings.Contains(err.Error(), "404 Not Found") {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving domain: %s", err)
	}

	d.Set("name", domain.Name)

	return nil
}
