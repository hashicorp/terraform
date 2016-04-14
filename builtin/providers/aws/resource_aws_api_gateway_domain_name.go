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
				ForceNew: true,
			},
		},
	}
}

func resourceAwsApiGatewayDomainCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	//Domain link with cloudfront distribution can take a while to actually delete
	err := resource.Retry(4*time.Minute, func() *resource.RetryError {
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
				return &resource.RetryError{
					Err:       err,
					Retryable: false,
				}
			}
			return &resource.RetryError{
				Err:       fmt.Errorf("Error creating domain name %s", err),
				Retryable: true,
			}
		}

		d.SetId(fmt.Sprintf("api-gateway-domain-name/%s", *r.DomainName))
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

	res, err := conn.GetDomainName(&apigateway.GetDomainNameInput{
		DomainName: aws.String(d.Get("domain_name").(string)),
	})

	if err != nil {
		if err, ok := err.(awserr.Error); ok && err.Code() == "NotFound" {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading api gateway domain %s", err)
	}

	d.Set("certificate_name", *res.CertificateName)
	d.Set("distribution_domain", *res.DistributionDomainName)

	return nil
}

func patchFor(d *schema.ResourceData, key string, res string, patches []*apigateway.PatchOperation) {
	if d.HasChange(key) {
		patches = append(patches, &apigateway.PatchOperation{
			Path:  aws.String(fmt.Sprintf("/%s", res)),
			Op:    aws.String("replace"),
			Value: aws.String(d.Get(key).(string)),
		})
	}
}

func resourceAwsApiGatewayDomainUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	var patches []*apigateway.PatchOperation

	patchFor(d, "certificate_name", "certificateName", patches)
	patchFor(d, "certificate_body", "certificateBody", patches)
	patchFor(d, "certificate_private_key", "certificatePrivateKey", patches)
	patchFor(d, "certificate_chain", "certificateChain", patches)

	if len(patches) == 0 {
		_, err := conn.UpdateDomainName(&apigateway.UpdateDomainNameInput{
			DomainName:      aws.String(d.Get("domain_name").(string)),
			PatchOperations: patches,
		})
		if err != nil {
			if err, ok := err.(awserr.Error); ok && err.Code() == "NotFound" {
				d.SetId("")
				return nil
			}
			return fmt.Errorf("Error updating api gateway domain %s", err)
		}
	}

	return resourceAwsApiGatewayDomainRead(d, meta)
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
