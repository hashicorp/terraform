package aws

import (
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsEcsCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEcsClusterCreate,
		Read:   resourceAwsEcsClusterRead,
		Delete: resourceAwsEcsClusterDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsEcsClusterCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecsconn

	clusterName := d.Get("name").(string)
	log.Printf("[DEBUG] Creating ECS cluster %s", clusterName)

	out, err := conn.CreateCluster(&ecs.CreateClusterInput{
		ClusterName: aws.String(clusterName),
	})
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] ECS cluster %s created", *out.Cluster.ClusterArn)

	d.SetId(*out.Cluster.ClusterArn)
	d.Set("name", *out.Cluster.ClusterName)
	return nil
}

func resourceAwsEcsClusterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecsconn

	clusterName := d.Get("name").(string)
	log.Printf("[DEBUG] Reading ECS cluster %s", clusterName)
	out, err := conn.DescribeClusters(&ecs.DescribeClustersInput{
		Clusters: []*string{aws.String(clusterName)},
	})
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Received ECS clusters: %s", out.Clusters)

	d.SetId(*out.Clusters[0].ClusterArn)
	d.Set("name", *out.Clusters[0].ClusterName)

	return nil
}

func resourceAwsEcsClusterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecsconn

	log.Printf("[DEBUG] Deleting ECS cluster %s", d.Id())

	return resource.Retry(10*time.Minute, func() error {
		out, err := conn.DeleteCluster(&ecs.DeleteClusterInput{
			Cluster: aws.String(d.Id()),
		})

		if err == nil {
			log.Printf("[DEBUG] ECS cluster %s deleted: %s", d.Id(), out)
			return nil
		}

		awsErr, ok := err.(awserr.Error)
		if !ok {
			return resource.RetryError{Err: err}
		}

		if awsErr.Code() == "ClusterContainsContainerInstancesException" {
			log.Printf("[TRACE] Retrying ECS cluster %q deletion after %q", d.Id(), awsErr.Code())
			return err
		}

		if awsErr.Code() == "ClusterContainsServicesException" {
			log.Printf("[TRACE] Retrying ECS cluster %q deletion after %q", d.Id(), awsErr.Code())
			return err
		}

		return resource.RetryError{Err: err}
	})
}
