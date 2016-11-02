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

func resourceAwsApiGatewayDomainName() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiGatewayDomainNameCreate,
		Read:   resourceAwsApiGatewayDomainNameRead,
		Update: resourceAwsApiGatewayDomainNameUpdate,
		Delete: resourceAwsApiGatewayDomainNameDelete,

		Schema: map[string]*schema.Schema{

			"certificate_body": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"certificate_chain": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"certificate_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"certificate_private_key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"domain_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"cloudfront_domain_name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"certificate_upload_date": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"cloudfront_zone_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsApiGatewayDomainNameCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Creating API Gateway Domain Name")

	domainName, err := conn.CreateDomainName(&apigateway.CreateDomainNameInput{
		CertificateBody:       aws.String(d.Get("certificate_body").(string)),
		CertificateChain:      aws.String(d.Get("certificate_chain").(string)),
		CertificateName:       aws.String(d.Get("certificate_name").(string)),
		CertificatePrivateKey: aws.String(d.Get("certificate_private_key").(string)),
		DomainName:            aws.String(d.Get("domain_name").(string)),
	})
	if err != nil {
		return fmt.Errorf("Error creating API Gateway Domain Name: %s", err)
	}

	d.SetId(*domainName.DomainName)
	d.Set("cloudfront_domain_name", domainName.DistributionDomainName)
	d.Set("cloudfront_zone_id", cloudFrontRoute53ZoneID)

	return resourceAwsApiGatewayDomainNameRead(d, meta)
}

func resourceAwsApiGatewayDomainNameRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Reading API Gateway Domain Name %s", d.Id())

	domainName, err := conn.GetDomainName(&apigateway.GetDomainNameInput{
		DomainName: aws.String(d.Id()),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFoundException" {
			log.Printf("[WARN] API gateway domain name %s has vanished\n", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("certificate_name", domainName.CertificateName)
	if err := d.Set("certificate_upload_date", domainName.CertificateUploadDate.Format(time.RFC3339)); err != nil {
		log.Printf("[DEBUG] Error setting certificate_upload_date: %s", err)
	}
	d.Set("cloudfront_domain_name", domainName.DistributionDomainName)
	d.Set("domain_name", domainName.DomainName)

	return nil
}

func resourceAwsApiGatewayDomainNameUpdateOperations(d *schema.ResourceData) []*apigateway.PatchOperation {
	operations := make([]*apigateway.PatchOperation, 0)

	if d.HasChange("certificate_body") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/certificate_body"),
			Value: aws.String(d.Get("certificate_body").(string)),
		})
	}

	if d.HasChange("certificate_chain") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/certificate_chain"),
			Value: aws.String(d.Get("certificate_chain").(string)),
		})
	}

	if d.HasChange("certificate_name") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/certificate_name"),
			Value: aws.String(d.Get("certificate_name").(string)),
		})
	}

	if d.HasChange("certificate_private_key") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/certificate_private_key"),
			Value: aws.String(d.Get("certificate_private_key").(string)),
		})
	}

	return operations
}

func resourceAwsApiGatewayDomainNameUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Updating API Gateway Domain Name %s", d.Id())

	_, err := conn.UpdateDomainName(&apigateway.UpdateDomainNameInput{
		DomainName:      aws.String(d.Id()),
		PatchOperations: resourceAwsApiGatewayDomainNameUpdateOperations(d),
	})
	if err != nil {
		return err
	}

	return resourceAwsApiGatewayDomainNameRead(d, meta)
}

func resourceAwsApiGatewayDomainNameDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Deleting API Gateway Domain Name: %s", d.Id())

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteDomainName(&apigateway.DeleteDomainNameInput{
			DomainName: aws.String(d.Id()),
		})

		if err == nil {
			return nil
		}

		if apigatewayErr, ok := err.(awserr.Error); ok && apigatewayErr.Code() == "NotFoundException" {
			return nil
		}

		return resource.NonRetryableError(err)
	})
}
