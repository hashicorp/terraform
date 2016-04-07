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

		Schema: map[string]*schema.Schema{
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: false,
				Required: true,
				ForceNew: true,
			},
			"client-id-list": &schema.Schema{
				Elem:     &schema.Schema{Type: schema.TypeString},
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
			},
			"thumbprint-list": &schema.Schema{
				Elem:     &schema.Schema{Type: schema.TypeString},
				Type:     schema.TypeList,
				Required: true,
			},
		},
	}
}

func stringListToStringSlice(stringList []interface{}) []string {
	ret := []string{}
	for _, v := range stringList {
		ret = append(ret, v.(string))
	}
	return ret
}

func resourceAwsIamOpenIDConnectProviderCreate(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	input := &iam.CreateOpenIDConnectProviderInput{
		Url:            aws.String(d.Get("url").(string)),
		ClientIDList:   aws.StringSlice(stringListToStringSlice(d.Get("client-id-list").([]interface{}))),
		ThumbprintList: aws.StringSlice(stringListToStringSlice(d.Get("thumbprint-list").([]interface{}))),
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
	d.Set("url", *out.Url)
	d.Set("client-id-list", out.ClientIDList)
	d.Set("thumbprint-list", out.ThumbprintList)

	return nil
}

func resourceAwsIamOpenIDConnectProviderUpdate(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	if d.HasChange("thumbprint-list") {
		input := &iam.UpdateOpenIDConnectProviderThumbprintInput{
			OpenIDConnectProviderArn: aws.String(d.Id()),
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
		if err, ok := err.(awserr.Error); ok && err.Code() == "NotFound" {
			return nil
		}
		return fmt.Errorf("Error deleting platform application %s", err)
	}

	return nil
}
