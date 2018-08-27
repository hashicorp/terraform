package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIamOpenIDConnectProvider() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamOpenIDConnectProviderCreate,
		Read:   resourceAwsIamOpenIDConnectProviderRead,
		Update: resourceAwsIamOpenIDConnectProviderUpdate,
		Delete: resourceAwsIamOpenIDConnectProviderDelete,
		Exists: resourceAwsIamOpenIDConnectProviderExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"url": {
				Type:             schema.TypeString,
				Computed:         false,
				Required:         true,
				ForceNew:         true,
				ValidateFunc:     validateOpenIdURL,
				DiffSuppressFunc: suppressOpenIdURL,
			},
			"client_id_list": {
				Elem:     &schema.Schema{Type: schema.TypeString},
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
			},
			"thumbprint_list": {
				Elem:     &schema.Schema{Type: schema.TypeString},
				Type:     schema.TypeList,
				Required: true,
			},
		},
	}
}

func resourceAwsIamOpenIDConnectProviderCreate(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	input := &iam.CreateOpenIDConnectProviderInput{
		Url:            aws.String(d.Get("url").(string)),
		ClientIDList:   expandStringList(d.Get("client_id_list").([]interface{})),
		ThumbprintList: expandStringList(d.Get("thumbprint_list").([]interface{})),
	}

	out, err := iamconn.CreateOpenIDConnectProvider(input)
	if err != nil {
		return err
	}

	d.SetId(*out.OpenIDConnectProviderArn)

	return resourceAwsIamOpenIDConnectProviderRead(d, meta)
}

func resourceAwsIamOpenIDConnectProviderRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	input := &iam.GetOpenIDConnectProviderInput{
		OpenIDConnectProviderArn: aws.String(d.Id()),
	}
	out, err := iamconn.GetOpenIDConnectProvider(input)
	if err != nil {
		return err
	}

	d.Set("arn", d.Id())
	d.Set("url", out.Url)
	d.Set("client_id_list", flattenStringList(out.ClientIDList))
	d.Set("thumbprint_list", flattenStringList(out.ThumbprintList))

	return nil
}

func resourceAwsIamOpenIDConnectProviderUpdate(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	if d.HasChange("thumbprint_list") {
		input := &iam.UpdateOpenIDConnectProviderThumbprintInput{
			OpenIDConnectProviderArn: aws.String(d.Id()),
			ThumbprintList:           expandStringList(d.Get("thumbprint_list").([]interface{})),
		}

		_, err := iamconn.UpdateOpenIDConnectProviderThumbprint(input)
		if err != nil {
			return err
		}
	}

	return resourceAwsIamOpenIDConnectProviderRead(d, meta)
}

func resourceAwsIamOpenIDConnectProviderDelete(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	input := &iam.DeleteOpenIDConnectProviderInput{
		OpenIDConnectProviderArn: aws.String(d.Id()),
	}
	_, err := iamconn.DeleteOpenIDConnectProvider(input)

	if err != nil {
		if err, ok := err.(awserr.Error); ok && err.Code() == "NoSuchEntity" {
			return nil
		}
		return fmt.Errorf("Error deleting platform application %s", err)
	}

	return nil
}

func resourceAwsIamOpenIDConnectProviderExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	iamconn := meta.(*AWSClient).iamconn

	input := &iam.GetOpenIDConnectProviderInput{
		OpenIDConnectProviderArn: aws.String(d.Id()),
	}
	_, err := iamconn.GetOpenIDConnectProvider(input)
	if err != nil {
		if err, ok := err.(awserr.Error); ok && err.Code() == "NoSuchEntity" {
			return false, nil
		}
		return true, err
	}

	return true, nil
}
