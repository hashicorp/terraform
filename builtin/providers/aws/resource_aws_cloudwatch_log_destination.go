package aws

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

func resourceAwsCloudWatchLogDestination() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudWatchLogDestinationPut,
		Update: resourceAwsCloudWatchLogDestinationPut,

		Read:   resourceAwsCloudWatchLogDestinationRead,
		Delete: resourceAwsCloudWatchLogDestinationDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"role_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"target_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsCloudWatchLogDestinationPut(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	name := d.Get("name").(string)
	role_arn := d.Get("role_arn").(string)
	target_arn := d.Get("target_arn").(string)

	params := &cloudwatchlogs.PutDestinationInput{
		DestinationName: aws.String(name),
		RoleArn:         aws.String(role_arn),
		TargetArn:       aws.String(target_arn),
	}

	resp, err := conn.PutDestination(params)

	if err != nil {
		return fmt.Errorf("Error creating Destination with name %s: %#v", name, err)
	}

	d.SetId(*resp.Destination.Arn)
	d.Set("arn", *resp.Destination.Arn)
	return resourceAwsCloudWatchLogDestinationRead(d, meta)
}

func resourceAwsCloudWatchLogDestinationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	name := d.Get("name").(string)

	params := &cloudwatchlogs.DescribeDestinationsInput{
		DestinationNamePrefix: aws.String(name),
	}

	resp, err := conn.DescribeDestinations(params)
	if err != nil {
		return fmt.Errorf("Error reading Destinations with name prefix %s: %#v", name, err)
	}

	for _, destination := range resp.Destinations {
		if *destination.DestinationName == name {
			d.SetId(*destination.Arn)
			d.Set("arn", *destination.Arn)
			d.Set("role_arn", *destination.RoleArn)
			d.Set("target_arn", *destination.TargetArn)
			return nil
		}
	}

	d.SetId("")
	return nil
}

func resourceAwsCloudWatchLogDestinationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	name := d.Get("name").(string)

	params := &cloudwatchlogs.DeleteDestinationInput{
		DestinationName: aws.String(name),
	}
	_, err := conn.DeleteDestination(params)
	if err != nil {
		return fmt.Errorf("Error deleting Destination with name %s", name)
	}
	d.SetId("")
	return nil
}
