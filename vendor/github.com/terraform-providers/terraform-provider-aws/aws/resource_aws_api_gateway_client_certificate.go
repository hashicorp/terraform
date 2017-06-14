package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsApiGatewayClientCertificate() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiGatewayClientCertificateCreate,
		Read:   resourceAwsApiGatewayClientCertificateRead,
		Update: resourceAwsApiGatewayClientCertificateUpdate,
		Delete: resourceAwsApiGatewayClientCertificateDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"created_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"expiration_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"pem_encoded_certificate": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsApiGatewayClientCertificateCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	input := apigateway.GenerateClientCertificateInput{}
	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}
	log.Printf("[DEBUG] Generating API Gateway Client Certificate: %s", input)
	out, err := conn.GenerateClientCertificate(&input)
	if err != nil {
		return fmt.Errorf("Failed to generate client certificate: %s", err)
	}

	d.SetId(*out.ClientCertificateId)

	return resourceAwsApiGatewayClientCertificateRead(d, meta)
}

func resourceAwsApiGatewayClientCertificateRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	input := apigateway.GetClientCertificateInput{
		ClientCertificateId: aws.String(d.Id()),
	}
	out, err := conn.GetClientCertificate(&input)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFoundException" {
			log.Printf("[WARN] API Gateway Client Certificate %s not found, removing", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}
	log.Printf("[DEBUG] Received API Gateway Client Certificate: %s", out)

	d.Set("description", out.Description)
	d.Set("created_date", out.CreatedDate)
	d.Set("expiration_date", out.ExpirationDate)
	d.Set("pem_encoded_certificate", out.PemEncodedCertificate)

	return nil
}

func resourceAwsApiGatewayClientCertificateUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	operations := make([]*apigateway.PatchOperation, 0)
	if d.HasChange("description") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String("replace"),
			Path:  aws.String("/description"),
			Value: aws.String(d.Get("description").(string)),
		})
	}

	input := apigateway.UpdateClientCertificateInput{
		ClientCertificateId: aws.String(d.Id()),
		PatchOperations:     operations,
	}

	log.Printf("[DEBUG] Updating API Gateway Client Certificate: %s", input)
	_, err := conn.UpdateClientCertificate(&input)
	if err != nil {
		return fmt.Errorf("Updating API Gateway Client Certificate failed: %s", err)
	}

	return resourceAwsApiGatewayClientCertificateRead(d, meta)
}

func resourceAwsApiGatewayClientCertificateDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	log.Printf("[DEBUG] Deleting API Gateway Client Certificate: %s", d.Id())
	input := apigateway.DeleteClientCertificateInput{
		ClientCertificateId: aws.String(d.Id()),
	}
	_, err := conn.DeleteClientCertificate(&input)
	if err != nil {
		return fmt.Errorf("Deleting API Gateway Client Certificate failed: %s", err)
	}

	return nil
}
