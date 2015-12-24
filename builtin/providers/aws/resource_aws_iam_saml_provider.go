package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIamSamlProvider() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamSamlProviderCreate,
		Read:   resourceAwsIamSamlProviderRead,
		Update: resourceAwsIamSamlProviderUpdate,
		Delete: resourceAwsIamSamlProviderDelete,

		Schema: map[string]*schema.Schema{
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"valid_until": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"saml_metadata_document": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsIamSamlProviderCreate(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	input := &iam.CreateSAMLProviderInput{
		Name:                 aws.String(d.Get("name").(string)),
		SAMLMetadataDocument: aws.String(d.Get("saml_metadata_document").(string)),
	}

	out, err := iamconn.CreateSAMLProvider(input)
	if err != nil {
		return err
	}

	d.SetId(*out.SAMLProviderArn)

	return resourceAwsIamSamlProviderRead(d, meta)
}

func resourceAwsIamSamlProviderRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	input := &iam.GetSAMLProviderInput{
		SAMLProviderArn: aws.String(d.Id()),
	}
	out, err := iamconn.GetSAMLProvider(input)
	if err != nil {
		return err
	}

	validUntil := out.ValidUntil.Format(time.RFC1123)
	d.Set("arn", d.Id())
	d.Set("valid_until", validUntil)
	d.Set("saml_metadata_document", *out.SAMLMetadataDocument)

	return nil
}

func resourceAwsIamSamlProviderUpdate(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	input := &iam.UpdateSAMLProviderInput{
		SAMLProviderArn:      aws.String(d.Id()),
		SAMLMetadataDocument: aws.String(d.Get("saml_metadata_document").(string)),
	}
	_, err := iamconn.UpdateSAMLProvider(input)
	if err != nil {
		return err
	}

	return resourceAwsIamSamlProviderRead(d, meta)
}

func resourceAwsIamSamlProviderDelete(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	input := &iam.DeleteSAMLProviderInput{
		SAMLProviderArn: aws.String(d.Id()),
	}
	_, err := iamconn.DeleteSAMLProvider(input)

	return err
}
