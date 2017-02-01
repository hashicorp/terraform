package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsEcsCluster() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsEcsClusterRead,

		Schema: map[string]*schema.Schema{
			"cluster_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"pending_tasks_count": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},

			"running_tasks_count": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},

			"registered_container_instances_count": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsEcsClusterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecsconn

	desc, err := conn.DescribeClusters(&ecs.DescribeClustersInput{})

	if err != nil {
		return err
	}

	var c *ecs.Cluster
	for _, cluster := range desc.Clusters {
		if aws.StringValue(cluster.ClusterName) == d.Get("cluster_name").(string) {
			c = cluster
			break
		}
	}

	if c == nil {
		return fmt.Errorf("cluster with name %q not found", d.Get("cluster_name").(string))
	}

	d.SetId(aws.StringValue(c.ClusterArn))
	d.Set("status", c.Status)
	d.Set("pending_tasks_count", c.PendingTasksCount)
	d.Set("running_tasks_count", c.RunningTasksCount)
	d.Set("registered_container_instances_count", c.RegisteredContainerInstancesCount)

	return nil
}
