package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsSfnStateMachine() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSfnStateMachineCreate,
		Read:   resourceAwsSfnStateMachineRead,
		Update: resourceAwsSfnStateMachineUpdate,
		Delete: resourceAwsSfnStateMachineDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"definition": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringLenBetween(0, 1024*1024), // 1048576
			},

			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateSfnStateMachineName,
			},

			"role_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateArn,
			},

			"creation_date": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsSfnStateMachineCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sfnconn
	log.Print("[DEBUG] Creating Step Function State Machine")

	params := &sfn.CreateStateMachineInput{
		Definition: aws.String(d.Get("definition").(string)),
		Name:       aws.String(d.Get("name").(string)),
		RoleArn:    aws.String(d.Get("role_arn").(string)),
	}

	var activity *sfn.CreateStateMachineOutput

	err := resource.Retry(5*time.Minute, func() *resource.RetryError {
		var err error
		activity, err = conn.CreateStateMachine(params)

		if err != nil {
			// Note: the instance may be in a deleting mode, hence the retry
			// when creating the step function. This can happen when we are
			// updating the resource (since there is no update API call).
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "StateMachineDeleting" {
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("Error creating Step Function State Machine: %s", err)
	}

	d.SetId(*activity.StateMachineArn)

	return resourceAwsSfnStateMachineRead(d, meta)
}

func resourceAwsSfnStateMachineRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sfnconn
	log.Printf("[DEBUG] Reading Step Function State Machine: %s", d.Id())

	sm, err := conn.DescribeStateMachine(&sfn.DescribeStateMachineInput{
		StateMachineArn: aws.String(d.Id()),
	})
	if err != nil {

		if awserr, ok := err.(awserr.Error); ok {
			if awserr.Code() == "NotFoundException" || awserr.Code() == "StateMachineDoesNotExist" {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	d.Set("definition", sm.Definition)
	d.Set("name", sm.Name)
	d.Set("role_arn", sm.RoleArn)
	d.Set("status", sm.Status)

	if err := d.Set("creation_date", sm.CreationDate.Format(time.RFC3339)); err != nil {
		log.Printf("[DEBUG] Error setting creation_date: %s", err)
	}

	return nil
}

func resourceAwsSfnStateMachineUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sfnconn

	params := &sfn.UpdateStateMachineInput{
		StateMachineArn: aws.String(d.Id()),
		Definition:      aws.String(d.Get("definition").(string)),
		RoleArn:         aws.String(d.Get("role_arn").(string)),
	}

	_, err := conn.UpdateStateMachine(params)

	log.Printf("[DEBUG] Updating Step Function State Machine: %#v", params)

	if err != nil {
		if isAWSErr(err, "StateMachineDoesNotExist", "State Machine Does Not Exist") {
			return fmt.Errorf("Error updating Step Function State Machine: %s", err)
		}
		return err
	}

	return resourceAwsSfnStateMachineRead(d, meta)
}

func resourceAwsSfnStateMachineDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sfnconn
	log.Printf("[DEBUG] Deleting Step Function State Machine: %s", d.Id())

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteStateMachine(&sfn.DeleteStateMachineInput{
			StateMachineArn: aws.String(d.Id()),
		})

		if err == nil {
			return nil
		}

		return resource.NonRetryableError(err)
	})
}
