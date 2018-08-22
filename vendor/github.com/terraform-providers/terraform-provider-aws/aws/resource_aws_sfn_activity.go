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
)

func resourceAwsSfnActivity() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSfnActivityCreate,
		Read:   resourceAwsSfnActivityRead,
		Delete: resourceAwsSfnActivityDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateMaxLength(80),
			},

			"creation_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsSfnActivityCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sfnconn
	log.Print("[DEBUG] Creating Step Function Activity")

	params := &sfn.CreateActivityInput{
		Name: aws.String(d.Get("name").(string)),
	}

	activity, err := conn.CreateActivity(params)
	if err != nil {
		return fmt.Errorf("Error creating Step Function Activity: %s", err)
	}

	d.SetId(*activity.ActivityArn)

	return resourceAwsSfnActivityRead(d, meta)
}

func resourceAwsSfnActivityRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sfnconn
	log.Printf("[DEBUG] Reading Step Function Activity: %s", d.Id())

	sm, err := conn.DescribeActivity(&sfn.DescribeActivityInput{
		ActivityArn: aws.String(d.Id()),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "ActivityDoesNotExist" {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", sm.Name)

	if err := d.Set("creation_date", sm.CreationDate.Format(time.RFC3339)); err != nil {
		log.Printf("[DEBUG] Error setting creation_date: %s", err)
	}

	return nil
}

func resourceAwsSfnActivityDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sfnconn
	log.Printf("[DEBUG] Deleting Step Functions Activity: %s", d.Id())

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteActivity(&sfn.DeleteActivityInput{
			ActivityArn: aws.String(d.Id()),
		})

		if err == nil {
			return nil
		}

		return resource.NonRetryableError(err)
	})
}
