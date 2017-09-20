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
			"cluster_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"pending_tasks_count": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"running_tasks_count": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"registered_container_instances_count": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsEcsClusterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecsconn

	desc, err := conn.DescribeClusters(&ecs.DescribeClustersInput{
		Clusters: []*string{aws.String(d.Get("cluster_name").(string))},
	})

	if err != nil {
		return err
	}

	for _, cluster := range desc.Clusters {
		if aws.StringValue(cluster.ClusterName) != d.Get("cluster_name").(string) {
			continue
		}
		d.SetId(aws.StringValue(cluster.ClusterArn))
		d.Set("arn", cluster.ClusterArn)
		d.Set("status", cluster.Status)
		d.Set("pending_tasks_count", cluster.PendingTasksCount)
		d.Set("running_tasks_count", cluster.RunningTasksCount)
		d.Set("registered_container_instances_count", cluster.RegisteredContainerInstancesCount)
	}

	if d.Id() == "" {
		return fmt.Errorf("cluster with name %q not found", d.Get("cluster_name").(string))
	}

	return nil
}
