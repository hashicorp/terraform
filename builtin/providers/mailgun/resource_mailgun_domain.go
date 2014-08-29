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

			"receiving_records": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"priority": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"record_type": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"valid": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"value": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"sending_records": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"record_type": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"valid": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"value": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
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

func resource_mailgin_domain_retrieve(id string, client *mailgun.Client, d *schema.ResourceData) (*mailgun.DomainResponse, error) {
	resp, err := client.RetrieveDomain(id)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving domain: %s", err)
	}

	d.Set("name", resp.Domain.Name)
	d.Set("smtp_password", resp.Domain.SmtpPassword)
	d.Set("smtp_login", resp.Domain.SmtpLogin)
	d.Set("wildcard", resp.Domain.Wildcard)
	d.Set("spam_action", resp.Domain.SpamAction)

	d.Set("receiving_records", make([]interface{}, len(resp.ReceivingRecords)))
	for i, r := range resp.ReceivingRecords {
		prefix := fmt.Sprintf("receiving_records.%d", i)
		d.Set(prefix+".priority", r.Priority)
		d.Set(prefix+".valid", r.Valid)
		d.Set(prefix+".value", r.Value)
		d.Set(prefix+".record_type", r.RecordType)
	}

	d.Set("sending_records", make([]interface{}, len(resp.SendingRecords)))
	for i, r := range resp.SendingRecords {
		prefix := fmt.Sprintf("sending_records.%d", i)
		d.Set(prefix+".name", r.Name)
		d.Set(prefix+".valid", r.Valid)
		d.Set(prefix+".value", r.Value)
		d.Set(prefix+".record_type", r.RecordType)
	}

	return &resp, nil
}
