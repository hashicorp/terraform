package aws

import (
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/opsworks"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsOpsworksPermission() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsOpsworksSetPermission,
		Update: resourceAwsOpsworksSetPermission,
		Delete: resourceAwsOpsworksPermissionDelete,
		Read:   resourceAwsOpsworksPermissionRead,

		Schema: map[string]*schema.Schema{
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
			"level": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					"deny",
					"show",
					"deploy",
					"manage",
					"iam_only",
				}, false),
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
			d.Set("allow_ssh", permission.AllowSsh)
			d.Set("allow_sudo", permission.AllowSudo)
			d.Set("user_arn", permission.IamUserArn)
			d.Set("stack_id", permission.StackId)
			d.Set("level", permission.Level)
		}

	}

	if !found {
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
		IamUserArn: aws.String(d.Get("user_arn").(string)),
		StackId:    aws.String(d.Get("stack_id").(string)),
	}

	if v, ok := d.GetOk("level"); ok {
		req.Level = aws.String(v.(string))
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
