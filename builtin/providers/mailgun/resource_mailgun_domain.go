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
	_, err = resourceMailginDomainRetrieve(d.Id(), client, d)

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

	_, err := resourceMailginDomainRetrieve(d.Id(), client, d)

	if err != nil {
		return err
	}

	return nil
}

func resourceMailginDomainRetrieve(id string, client *mailgun.Client, d *schema.ResourceData) (*mailgun.DomainResponse, error) {
	resp, err := client.RetrieveDomain(id)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving domain: %s", err)
	}

	d.Set("name", resp.Domain.Name)
	d.Set("smtp_password", resp.Domain.SmtpPassword)
	d.Set("smtp_login", resp.Domain.SmtpLogin)
	d.Set("wildcard", resp.Domain.Wildcard)
	d.Set("spam_action", resp.Domain.SpamAction)

	receivingRecords := make([]map[string]interface{}, len(resp.ReceivingRecords))
	for i, r := range resp.ReceivingRecords {
		receivingRecords[i] = make(map[string]interface{})
		receivingRecords[i]["priority"] = r.Priority
		receivingRecords[i]["valid"] = r.Valid
		receivingRecords[i]["value"] = r.Value
		receivingRecords[i]["record_type"] = r.RecordType
	}
	d.Set("receiving_records", receivingRecords)

	sendingRecords := make([]map[string]interface{}, len(resp.SendingRecords))
	for i, r := range resp.SendingRecords {
		sendingRecords[i] = make(map[string]interface{})
		sendingRecords[i]["name"] = r.Name
		sendingRecords[i]["valid"] = r.Valid
		sendingRecords[i]["value"] = r.Value
		sendingRecords[i]["record_type"] = r.RecordType
	}
	d.Set("sending_records", sendingRecords)

	return &resp, nil
}
