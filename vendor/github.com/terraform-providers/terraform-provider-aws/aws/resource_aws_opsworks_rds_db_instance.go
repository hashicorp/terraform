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

func resourceAwsOpsworksRdsDbInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsOpsworksRdsDbInstanceRegister,
		Update: resourceAwsOpsworksRdsDbInstanceUpdate,
		Delete: resourceAwsOpsworksRdsDbInstanceDeregister,
		Read:   resourceAwsOpsworksRdsDbInstanceRead,

		Schema: map[string]*schema.Schema{
			"stack_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"rds_db_instance_arn": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"db_password": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			"db_user": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsOpsworksRdsDbInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn

	d.Partial(true)

	d.SetPartial("rds_db_instance_arn")
	req := &opsworks.UpdateRdsDbInstanceInput{
		RdsDbInstanceArn: aws.String(d.Get("rds_db_instance_arn").(string)),
	}

	requestUpdate := false
	if d.HasChange("db_user") {
		d.SetPartial("db_user")
		req.DbUser = aws.String(d.Get("db_user").(string))
		requestUpdate = true
	}
	if d.HasChange("db_password") {
		d.SetPartial("db_password")
		req.DbPassword = aws.String(d.Get("db_password").(string))
		requestUpdate = true
	}

	if requestUpdate {
		log.Printf("[DEBUG] Opsworks RDS DB Instance Modification request: %s", req)

		err := resource.Retry(2*time.Minute, func() *resource.RetryError {
			var cerr error
			_, cerr = client.UpdateRdsDbInstance(req)
			if cerr != nil {
				log.Printf("[INFO] client error")
				if opserr, ok := cerr.(awserr.Error); ok {
					log.Printf("[ERROR] OpsWorks error: %s message: %s", opserr.Code(), opserr.Message())
				}
				return resource.NonRetryableError(cerr)
			}
			return nil
		})

		if err != nil {
			return err
		}

	}

	d.Partial(false)

	return resourceAwsOpsworksRdsDbInstanceRead(d, meta)
}

func resourceAwsOpsworksRdsDbInstanceDeregister(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn

	req := &opsworks.DeregisterRdsDbInstanceInput{
		RdsDbInstanceArn: aws.String(d.Get("rds_db_instance_arn").(string)),
	}

	log.Printf("[DEBUG] Unregistering rds db instance '%s' from stack: %s", d.Get("rds_db_instance_arn"), d.Get("stack_id"))

	err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		var cerr error
		_, cerr = client.DeregisterRdsDbInstance(req)
		if cerr != nil {
			log.Printf("[INFO] client error")
			if opserr, ok := cerr.(awserr.Error); ok {
				if opserr.Code() == "ResourceNotFoundException" {
					log.Printf("[INFO] The db instance could not be found. Remove it from state.")
					d.SetId("")

					return nil
				}
				log.Printf("[ERROR] OpsWorks error: %s message: %s", opserr.Code(), opserr.Message())
			}
			return resource.NonRetryableError(cerr)
		}

		return nil
	})

	return err
}

func resourceAwsOpsworksRdsDbInstanceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn

	req := &opsworks.DescribeRdsDbInstancesInput{
		StackId: aws.String(d.Get("stack_id").(string)),
	}

	log.Printf("[DEBUG] Reading OpsWorks registered rds db instances for stack: %s", d.Get("stack_id"))

	resp, err := client.DescribeRdsDbInstances(req)
	if err != nil {
		return err
	}

	found := false
	id := ""
	for _, instance := range resp.RdsDbInstances {
		id = fmt.Sprintf("%s%s", *instance.RdsDbInstanceArn, *instance.StackId)

		if fmt.Sprintf("%s%s", d.Get("rds_db_instance_arn").(string), d.Get("stack_id").(string)) == id {
			found = true
			d.SetId(id)
			d.Set("stack_id", instance.StackId)
			d.Set("rds_db_instance_arn", instance.RdsDbInstanceArn)
			d.Set("db_user", instance.DbUser)
		}

	}

	if !found {
		d.SetId("")
		log.Printf("[INFO] The rds instance '%s' could not be found for stack: '%s'", d.Get("rds_db_instance_arn"), d.Get("stack_id"))
	}

	return nil
}

func resourceAwsOpsworksRdsDbInstanceRegister(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn

	req := &opsworks.RegisterRdsDbInstanceInput{
		StackId:          aws.String(d.Get("stack_id").(string)),
		RdsDbInstanceArn: aws.String(d.Get("rds_db_instance_arn").(string)),
		DbUser:           aws.String(d.Get("db_user").(string)),
		DbPassword:       aws.String(d.Get("db_password").(string)),
	}

	err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		var cerr error
		_, cerr = client.RegisterRdsDbInstance(req)
		if cerr != nil {
			log.Printf("[INFO] client error")
			if opserr, ok := cerr.(awserr.Error); ok {
				log.Printf("[ERROR] OpsWorks error: %s message: %s", opserr.Code(), opserr.Message())
			}
			return resource.NonRetryableError(cerr)
		}

		return nil
	})

	if err != nil {
		return err
	}

	return resourceAwsOpsworksRdsDbInstanceRead(d, meta)
}
