package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
)

func resourceAwsRoute53QueryLog() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRoute53QueryLogCreate,
		Read:   resourceAwsRoute53QueryLogRead,
		Delete: resourceAwsRoute53QueryLogDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"cloudwatch_log_group_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},

			"zone_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsRoute53QueryLogCreate(d *schema.ResourceData, meta interface{}) error {
	r53 := meta.(*AWSClient).r53conn

	input := &route53.CreateQueryLoggingConfigInput{
		CloudWatchLogsLogGroupArn: aws.String(d.Get("cloudwatch_log_group_arn").(string)),
		HostedZoneId:              aws.String(d.Get("zone_id").(string)),
	}

	log.Printf("[DEBUG] Creating Route53 query logging configuration: %#v", input)
	out, err := r53.CreateQueryLoggingConfig(input)
	if err != nil {
		return fmt.Errorf("Error creating Route53 query logging configuration: %s", err)
	}
	log.Printf("[DEBUG] Route53 query logging configuration created: %#v", out)

	d.SetId(*out.QueryLoggingConfig.Id)

	return resourceAwsRoute53QueryLogRead(d, meta)
}

func resourceAwsRoute53QueryLogRead(d *schema.ResourceData, meta interface{}) error {
	r53 := meta.(*AWSClient).r53conn

	input := &route53.GetQueryLoggingConfigInput{
		Id: aws.String(d.Id()),
	}
	log.Printf("[DEBUG] Reading Route53 query logging configuration: %#v", input)
	out, err := r53.GetQueryLoggingConfig(input)
	if err != nil {
		return fmt.Errorf("Error reading Route53 query logging configuration: %s", err)
	}
	log.Printf("[DEBUG] Route53 query logging configuration received: %#v", out)

	d.Set("cloudwatch_log_group_arn", out.QueryLoggingConfig.CloudWatchLogsLogGroupArn)
	d.Set("zone_id", out.QueryLoggingConfig.HostedZoneId)

	return nil
}

func resourceAwsRoute53QueryLogDelete(d *schema.ResourceData, meta interface{}) error {
	r53 := meta.(*AWSClient).r53conn

	input := &route53.DeleteQueryLoggingConfigInput{
		Id: aws.String(d.Id()),
	}
	log.Printf("[DEBUG] Deleting Route53 query logging configuration: %#v", input)
	_, err := r53.DeleteQueryLoggingConfig(input)
	if err != nil {
		return fmt.Errorf("Error deleting Route53 query logging configuration: %s", err)
	}

	return nil
}
