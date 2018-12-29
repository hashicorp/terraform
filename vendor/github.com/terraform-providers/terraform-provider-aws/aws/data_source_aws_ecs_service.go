package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsEcsService() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsEcsServiceRead,

		Schema: map[string]*schema.Schema{
			"service_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cluster_arn": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"desired_count": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"launch_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"scheduling_strategy": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"task_definition": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsEcsServiceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecsconn

	clusterArn := d.Get("cluster_arn").(string)
	serviceName := d.Get("service_name").(string)

	params := &ecs.DescribeServicesInput{
		Cluster:  aws.String(clusterArn),
		Services: []*string{aws.String(serviceName)},
	}

	log.Printf("[DEBUG] Reading ECS Service: %s", params)
	desc, err := conn.DescribeServices(params)

	if err != nil {
		return err
	}

	if desc == nil || len(desc.Services) == 0 {
		return fmt.Errorf("service with name %q in cluster %q not found", serviceName, clusterArn)
	}

	if len(desc.Services) > 1 {
		return fmt.Errorf("multiple services with name %q found in cluster %q", serviceName, clusterArn)
	}

	service := desc.Services[0]
	d.SetId(aws.StringValue(service.ServiceArn))

	d.Set("service_name", service.ServiceName)
	d.Set("arn", service.ServiceArn)
	d.Set("cluster_arn", service.ClusterArn)
	d.Set("desired_count", service.DesiredCount)
	d.Set("launch_type", service.LaunchType)
	d.Set("scheduling_strategy", service.SchedulingStrategy)
	d.Set("task_definition", service.TaskDefinition)

	return nil
}
