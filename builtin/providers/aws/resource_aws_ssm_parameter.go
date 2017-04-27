package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSsmParameter() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSsmParameterCreate,
		Read:   resourceAwsSsmParameterRead,
		Update: resourceAwsSsmParameterUpdate,
		Delete: resourceAwsSsmParameterDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateSsmParameterType,
			},
			"value": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsSsmParameterCreate(d *schema.ResourceData, meta interface{}) error {
	return putAwsSSMParameter(d, meta, false)
}

func resourceAwsSsmParameterRead(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[DEBUG] Reading SSM Parameter: %s", d.Id())

	paramInput := &ssm.GetParametersInput{
		Names: []*string{
			aws.String(d.Get("name").(string)),
		},
		WithDecryption: aws.Bool(true),
	}

	resp, err := ssmconn.GetParameters(paramInput)

	if err != nil {
		return errwrap.Wrapf("[ERROR] Error describing SSM parameter: {{err}}", err)
	}

	if len(resp.InvalidParameters) > 0 {
		return fmt.Errorf("[ERROR] SSM Parameter %s is invalid", d.Id())
	}

	param := resp.Parameters[0]
	d.Set("name", param.Name)
	d.Set("type", param.Type)
	d.Set("value", param.Value)

	return nil
}

func resourceAwsSsmParameterUpdate(d *schema.ResourceData, meta interface{}) error {
	return putAwsSSMParameter(d, meta, true)
}

func resourceAwsSsmParameterDelete(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[INFO] Deleting SSM Parameter: %s", d.Id())

	paramInput := &ssm.DeleteParameterInput{
		Name: aws.String(d.Get("name").(string)),
	}

	_, err := ssmconn.DeleteParameter(paramInput)
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func putAwsSSMParameter(d *schema.ResourceData, meta interface{}, overwrite bool) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[INFO] Creating SSM Parameter: %s", d.Get("name").(string))

	paramInput := &ssm.PutParameterInput{
		Name:      aws.String(d.Get("name").(string)),
		Type:      aws.String(d.Get("type").(string)),
		Value:     aws.String(d.Get("value").(string)),
		Overwrite: aws.Bool(overwrite),
	}

	log.Printf("[DEBUG] Waiting for SSM Parameter %q to be updated", d.Get("name").(string))
	err := resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := ssmconn.PutParameter(paramInput)

		if err != nil {
			return resource.NonRetryableError(err)
		}

		d.SetId(d.Get("name").(string))
		return nil
	})

	if err != nil {
		return errwrap.Wrapf("[ERROR] Error creating SSM parameter: {{err}}", err)
	}

	return resourceAwsSsmParameterRead(d, meta)
}
