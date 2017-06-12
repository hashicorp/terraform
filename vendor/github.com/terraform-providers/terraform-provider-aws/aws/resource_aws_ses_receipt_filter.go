package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSesReceiptFilter() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSesReceiptFilterCreate,
		Read:   resourceAwsSesReceiptFilterRead,
		Delete: resourceAwsSesReceiptFilterDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"cidr": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"policy": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsSesReceiptFilterCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sesConn

	name := d.Get("name").(string)

	createOpts := &ses.CreateReceiptFilterInput{
		Filter: &ses.ReceiptFilter{
			Name: aws.String(name),
			IpFilter: &ses.ReceiptIpFilter{
				Cidr:   aws.String(d.Get("cidr").(string)),
				Policy: aws.String(d.Get("policy").(string)),
			},
		},
	}

	_, err := conn.CreateReceiptFilter(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating SES receipt filter: %s", err)
	}

	d.SetId(name)

	return resourceAwsSesReceiptFilterRead(d, meta)
}

func resourceAwsSesReceiptFilterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sesConn

	listOpts := &ses.ListReceiptFiltersInput{}

	response, err := conn.ListReceiptFilters(listOpts)
	if err != nil {
		return err
	}

	found := false
	for _, element := range response.Filters {
		if *element.Name == d.Id() {
			d.Set("cidr", element.IpFilter.Cidr)
			d.Set("policy", element.IpFilter.Policy)
			d.Set("name", element.Name)
			found = true
		}
	}

	if !found {
		log.Printf("[WARN] SES Receipt Filter (%s) not found", d.Id())
		d.SetId("")
	}

	return nil
}

func resourceAwsSesReceiptFilterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sesConn

	deleteOpts := &ses.DeleteReceiptFilterInput{
		FilterName: aws.String(d.Id()),
	}

	_, err := conn.DeleteReceiptFilter(deleteOpts)
	if err != nil {
		return fmt.Errorf("Error deleting SES receipt filter: %s", err)
	}

	return nil
}
