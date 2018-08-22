package aws

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

func resourceAwsCloudWatchLogDestinationPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudWatchLogDestinationPolicyPut,
		Update: resourceAwsCloudWatchLogDestinationPolicyPut,
		Read:   resourceAwsCloudWatchLogDestinationPolicyRead,
		Delete: resourceAwsCloudWatchLogDestinationPolicyDelete,

		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				d.Set("destination_name", d.Id())
				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"destination_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"access_policy": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsCloudWatchLogDestinationPolicyPut(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	destination_name := d.Get("destination_name").(string)
	access_policy := d.Get("access_policy").(string)

	params := &cloudwatchlogs.PutDestinationPolicyInput{
		DestinationName: aws.String(destination_name),
		AccessPolicy:    aws.String(access_policy),
	}

	_, err := conn.PutDestinationPolicy(params)

	if err != nil {
		return fmt.Errorf("Error creating DestinationPolicy with destination_name %s: %#v", destination_name, err)
	}

	d.SetId(destination_name)
	return resourceAwsCloudWatchLogDestinationPolicyRead(d, meta)
}

func resourceAwsCloudWatchLogDestinationPolicyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn
	destination_name := d.Get("destination_name").(string)
	destination, exists, err := lookupCloudWatchLogDestination(conn, destination_name, nil)
	if err != nil {
		return err
	}

	if !exists {
		d.SetId("")
		return nil
	}

	if destination.AccessPolicy != nil {
		d.SetId(destination_name)
		d.Set("access_policy", *destination.AccessPolicy)
	} else {
		d.SetId("")
	}

	return nil
}

func resourceAwsCloudWatchLogDestinationPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}
