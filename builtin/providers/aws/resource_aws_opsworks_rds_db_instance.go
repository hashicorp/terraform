package aws

import (
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
		Update: resourceAwsOpsworksRdsDbInstanceRegister,
		Delete: resourceAwsOpsworksRdsDbInstanceUnregister,
		Read:   resourceAwsOpsworksRdsDbInstanceRead,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"stack_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"rds_db_instance_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"db_password": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"db_user": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsOpsworksRdsDbInstanceUnregister(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn

	req := &opsworks.DeregisterRdsDbInstanceInput{
		RdsDbInstanceArn: aws.String(d.Get("rds_db_instance_arn").(string)),
	}

	log.Printf("[DEBUG] Unregistering rds db instance '%s' from stack: %s", d.Get("rds_db_instance_arn"), d.Get("stack_id"))

	_, err := client.DeregisterRdsDbInstance(req)
	if err != nil {
		if awserr, ok := err.(awserr.Error); ok {
			if awserr.Code() == "ResourceNotFoundException" {
				log.Printf("[INFO] The db instance could not be found")
				d.SetId("")

				return nil
			}
		}
		return err
	}

	return nil
}

func resourceAwsOpsworksRdsDbInstanceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn

	arns := []string{d.Get("rds_db_instance_arn").(string)}

	req := &opsworks.DescribeRdsDbInstancesInput{
		StackId:           aws.String(d.Get("stack_id").(string)),
		RdsDbInstanceArns: aws.StringSlice(arns),
	}

	log.Printf("[DEBUG] Reading OpsWorks registerd rds db instances for stack: %s", d.Get("stack_id"))

	resp, err := client.DescribeRdsDbInstances(req)
	if err != nil {
		if awserr, ok := err.(awserr.Error); ok {
			if awserr.Code() == "ResourceNotFoundException" {
				log.Printf("[INFO] No instance found not found")
				d.SetId("")

				return nil
			}
		}
		return err
	}

	found := false
	id := ""
	for _, instance := range resp.RdsDbInstances {
		id = *instance.RdsDbInstanceArn + *instance.StackId

		if d.Get("rds_db_instance_arn").(string)+d.Get("stack_id").(string) == id {
			found = true
			d.SetId(id)
			d.Set("id", id)
			d.Set("stack_id", instance.StackId)
			d.Set("rds_db_instance_arn", instance.RdsDbInstanceArn)
			d.Set("db_user", instance.DbUser)
			d.Set("db_password", instance.DbPassword)
		}

	}

	if false == found {
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

	return resourceAwsOpsworksRdsDbInstanceRead(d, meta)
}
