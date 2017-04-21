package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsLightsailDomain() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLightsailDomainCreate,
		Read:   resourceAwsLightsailDomainRead,
		Delete: resourceAwsLightsailDomainDelete,

		Schema: map[string]*schema.Schema{
			"domain_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsLightsailDomainCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lightsailconn
	_, err := conn.CreateDomain(&lightsail.CreateDomainInput{
		DomainName: aws.String(d.Get("domain_name").(string)),
	})

	if err != nil {
		return err
	}

	d.SetId(d.Get("domain_name").(string))

	return resourceAwsLightsailDomainRead(d, meta)
}

func resourceAwsLightsailDomainRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lightsailconn
	resp, err := conn.GetDomain(&lightsail.GetDomainInput{
		DomainName: aws.String(d.Id()),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NotFoundException" {
				log.Printf("[WARN] Lightsail Domain (%s) not found, removing from state", d.Id())
				d.SetId("")
				return nil
			}
			return err
		}
		return err
	}

	d.Set("arn", resp.Domain.Arn)
	return nil
}

func resourceAwsLightsailDomainDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lightsailconn
	_, err := conn.DeleteDomain(&lightsail.DeleteDomainInput{
		DomainName: aws.String(d.Id()),
	})

	if err != nil {
		return err
	}
	return nil
}
