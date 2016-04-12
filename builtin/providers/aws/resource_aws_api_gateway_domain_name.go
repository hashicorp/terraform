package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

// AWS APIGateway domain name declaration
func resourceAwsApiGatewayDomain() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiGatewayDomainCreate,
		Read:   resourceAwsApiGatewayDomainRead,
		Delete: resourceAwsApiGatewayDomainDelete,

		Schema: map[string]*schema.Schema{
			"domain_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"certificate_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"certificate_body": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"certificate_private_key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"certificate_chain": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"distribution_domain": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsApiGatewayDomainCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	//Domain link with cloudfront distribution can take a while to actually delete
	err := resource.Retry(4*time.Minute, func() error {
		r, err := conn.CreateDomainName(&apigateway.CreateDomainNameInput{
			DomainName:            aws.String(d.Get("domain_name").(string)),
			CertificateName:       aws.String(d.Get("certificate_name").(string)),
			CertificateBody:       aws.String(d.Get("certificate_body").(string)),
			CertificateChain:      aws.String(d.Get("certificate_chain").(string)),
			CertificatePrivateKey: aws.String(d.Get("certificate_private_key").(string)),
		})

		if err != nil {
			log.Printf("[DEBUG] Error creating domain - not found : %s", err)
			if err, ok := err.(awserr.Error); ok && err.Code() != "BadRequestException" {
				return nil
			}
			return fmt.Errorf("Error creating domain name %s", err)
		}

		d.SetId(*r.DomainName)
		d.Set("distribution_domain", *r.DistributionDomainName)

		return nil
	})

	if err != nil {
		return fmt.Errorf("Error creating domain name %s", err)
	}

	return resourceAwsApiGatewayDomainRead(d, meta)
}

func resourceAwsApiGatewayDomainRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	_, err := conn.GetDomainName(&apigateway.GetDomainNameInput{
		DomainName: aws.String(d.Id()),
	})

	if err != nil {
		if err, ok := err.(awserr.Error); ok && err.Code() == "NotFound" {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading api gateway domain %s", err)
	}

	return nil
}

func resourceAwsApiGatewayDomainDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	_, err := conn.DeleteDomainName(&apigateway.DeleteDomainNameInput{
		DomainName: aws.String(d.Get("domain_name").(string)),
	})

	if err != nil {
		if err, ok := err.(awserr.Error); ok && err.Code() == "NotFound" {
			return nil
		}
		return fmt.Errorf("Error deleting domain name %s", err)
	}
	return nil
}
