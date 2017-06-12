package aws

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIamSamlProvider() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamSamlProviderCreate,
		Read:   resourceAwsIamSamlProviderRead,
		Update: resourceAwsIamSamlProviderUpdate,
		Delete: resourceAwsIamSamlProviderDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"valid_until": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"saml_metadata_document": {
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
		if iamerr, ok := err.(awserr.Error); ok && iamerr.Code() == "NoSuchEntity" {
			log.Printf("[WARN] IAM SAML Provider %q not found.", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	validUntil := out.ValidUntil.Format(time.RFC1123)
	d.Set("arn", d.Id())
	name, err := extractNameFromIAMSamlProviderArn(d.Id(), meta.(*AWSClient).partition)
	if err != nil {
		return err
	}
	d.Set("name", name)
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

func extractNameFromIAMSamlProviderArn(arn, partition string) (string, error) {
	// arn:aws:iam::123456789012:saml-provider/tf-salesforce-test
	r := regexp.MustCompile(fmt.Sprintf("^arn:%s:iam::[0-9]{12}:saml-provider/(.+)$", partition))
	submatches := r.FindStringSubmatch(arn)
	if len(submatches) != 2 {
		return "", fmt.Errorf("Unable to extract name from a given ARN: %q", arn)
	}
	return submatches[1], nil
}
