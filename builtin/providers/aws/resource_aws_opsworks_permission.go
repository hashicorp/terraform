package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/opsworks"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsOpsworksPermission() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsOpsworksSetPermission,
		Update: resourceAwsOpsworksSetPermission,
		Delete: resourceAwsOpsworksPermissionDelete,
		Read:   resourceAwsOpsworksPermissionRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"allow_ssh": {
				Type:     schema.TypeBool,
				Computed: true,
				Optional: true,
			},
			"allow_sudo": {
				Type:     schema.TypeBool,
				Computed: true,
				Optional: true,
			},
			"user_arn": {
				Type:     schema.TypeString,
				Required: true,
			},
			// one of deny, show, deploy, manage, iam_only
			"level": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)

					expected := [5]string{"deny", "show", "deploy", "manage", "iam_only"}

					found := false
					for _, b := range expected {
						if b == value {
							found = true
						}
					}
					if !found {
						errors = append(errors, fmt.Errorf(
							"%q has to be one of [deny, show, deploy, manage, iam_only]", k))
					}
					return
				},
			},
			"stack_id": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
			},
		},
	}
}

func resourceAwsOpsworksPermissionDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsOpsworksPermissionRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn

	req := &opsworks.DescribePermissionsInput{
		IamUserArn: aws.String(d.Get("user_arn").(string)),
		StackId:    aws.String(d.Get("stack_id").(string)),
	}

	log.Printf("[DEBUG] Reading OpsWorks prermissions for: %s on stack: %s", d.Get("user_arn"), d.Get("stack_id"))

	resp, err := client.DescribePermissions(req)
	if err != nil {
		if awserr, ok := err.(awserr.Error); ok {
			if awserr.Code() == "ResourceNotFoundException" {
				log.Printf("[INFO] Permission not found")
				d.SetId("")
				return nil
			}
		}
		return err
	}

	found := false
	id := ""
	for _, permission := range resp.Permissions {
		id = *permission.IamUserArn + *permission.StackId

		if d.Get("user_arn").(string)+d.Get("stack_id").(string) == id {
			found = true
			d.SetId(id)
			d.Set("id", id)
			d.Set("allow_ssh", permission.AllowSsh)
			d.Set("allow_sudo", permission.AllowSudo)
			d.Set("user_arn", permission.IamUserArn)
			d.Set("stack_id", permission.StackId)
			d.Set("level", permission.Level)
		}

	}

	if false == found {
		d.SetId("")
		log.Printf("[INFO] The correct permission could not be found for: %s on stack: %s", d.Get("user_arn"), d.Get("stack_id"))
	}

	return nil
}

func resourceAwsOpsworksSetPermission(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn

	req := &opsworks.SetPermissionInput{
		AllowSudo:  aws.Bool(d.Get("allow_sudo").(bool)),
		AllowSsh:   aws.Bool(d.Get("allow_ssh").(bool)),
		Level:      aws.String(d.Get("level").(string)),
		IamUserArn: aws.String(d.Get("user_arn").(string)),
		StackId:    aws.String(d.Get("stack_id").(string)),
	}

	err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		var cerr error
		_, cerr = client.SetPermission(req)
		if cerr != nil {
			log.Printf("[INFO] client error")
			if opserr, ok := cerr.(awserr.Error); ok {
				// XXX: handle errors
				log.Printf("[ERROR] OpsWorks error: %s message: %s", opserr.Code(), opserr.Message())
				return resource.RetryableError(cerr)
			}
			return resource.NonRetryableError(cerr)
		}
		return nil
	})

	if err != nil {
		return err
	}

	return resourceAwsOpsworksPermissionRead(d, meta)
}
