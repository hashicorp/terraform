package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsEcsTaskDefinition() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsEcsTaskDefinitionRead,

		Schema: map[string]*schema.Schema{
			"task_definition": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			// Computed values.
			"family": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"network_mode": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"revision": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"task_role_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsEcsTaskDefinitionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecsconn

	params := &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(d.Get("task_definition").(string)),
	}
	log.Printf("[DEBUG] Reading ECS Task Definition: %s", params)
	desc, err := conn.DescribeTaskDefinition(params)

	if err != nil {
		return fmt.Errorf("Failed getting task definition %s %q", err, d.Get("task_definition").(string))
	}

	taskDefinition := *desc.TaskDefinition

	d.SetId(aws.StringValue(taskDefinition.TaskDefinitionArn))
	d.Set("family", aws.StringValue(taskDefinition.Family))
	d.Set("network_mode", aws.StringValue(taskDefinition.NetworkMode))
	d.Set("revision", aws.Int64Value(taskDefinition.Revision))
	d.Set("status", aws.StringValue(taskDefinition.Status))
	d.Set("task_role_arn", aws.StringValue(taskDefinition.TaskRoleArn))

	if d.Id() == "" {
		return fmt.Errorf("task definition %q not found", d.Get("task_definition").(string))
	}

	return nil
}
