package mailgun

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/pearkes/mailgun"
)

func resourceMailgunDomain() *schema.Resource {
	return &schema.Resource{
		Create: resourceMailgunDomainCreate,
		Read:   resourceMailgunDomainRead,
		Delete: resourceMailgunDomainDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"spam_action": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
				Optional: true,
			},

			"smtp_password": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
				Required: true,
			},

			"smtp_login": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
			},

			"wildcard": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
				ForceNew: true,
				Optional: true,
			},
		},
	}
}

func resourceMailgunDomainCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*mailgun.Client)

	opts := mailgun.CreateDomain{}

	opts.Name = d.Get("name").(string)
	opts.SmtpPassword = d.Get("smtp_password").(string)
	opts.SpamAction = d.Get("spam_action").(string)
	opts.Wildcard = d.Get("wildcard").(bool)

	log.Printf("[DEBUG] Domain create configuration: %#v", opts)

	domain, err := client.CreateDomain(&opts)

	if err != nil {
		return err
	}

	d.SetId(domain)

	log.Printf("[INFO] Domain ID: %s", d.Id())

	// Retrieve and update state of domain
	_, err = resource_mailgin_domain_retrieve(d.Id(), client, d)

	if err != nil {
		return err
	}

	return nil
}

func resourceMailgunDomainDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*mailgun.Client)

	log.Printf("[INFO] Deleting Domain: %s", d.Id())

	// Destroy the domain
	err := client.DestroyDomain(d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting domain: %s", err)
	}

	return nil
}

func resourceMailgunDomainRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*mailgun.Client)

	_, err := resource_mailgin_domain_retrieve(d.Id(), client, d)

	if err != nil {
		return err
	}

	return nil
}

func resource_mailgin_domain_retrieve(id string, client *mailgun.Client, d *schema.ResourceData) (*mailgun.Domain, error) {
	domain, err := client.RetrieveDomain(id)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving domain: %s", err)
	}

	d.Set("name", domain.Name)
	d.Set("smtp_password", domain.SmtpPassword)
	d.Set("smtp_login", domain.SmtpLogin)
	d.Set("wildcard", domain.Wildcard)
	d.Set("spam_action", domain.SpamAction)

	return &domain, nil
}
