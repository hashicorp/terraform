package aws

import (
	"fmt"
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

	for _, c := range out.Clusters {
		if *c.ClusterName == clusterName {
			// Status==INACTIVE means deleted cluster
			if *c.Status == "INACTIVE" {
				log.Printf("[DEBUG] Removing ECS cluster %q because it's INACTIVE", *c.ClusterArn)
				d.SetId("")
				return nil
			}

			d.SetId(*c.ClusterArn)
			d.Set("name", c.ClusterName)
			return nil
		}
	}

	log.Printf("[ERR] No matching ECS Cluster found for (%s)", d.Id())
	d.SetId("")
	return nil
}

func resourceAwsEcsClusterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecsconn

	log.Printf("[DEBUG] Deleting ECS cluster %s", d.Id())

	err := resource.Retry(10*time.Minute, func() error {
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
	if err != nil {
		return err
	}

	clusterName := d.Get("name").(string)
	err = resource.Retry(5*time.Minute, func() error {
		log.Printf("[DEBUG] Checking if ECS Cluster %q is INACTIVE", d.Id())
		out, err := conn.DescribeClusters(&ecs.DescribeClustersInput{
			Clusters: []*string{aws.String(clusterName)},
		})

		for _, c := range out.Clusters {
			if *c.ClusterName == clusterName {
				if *c.Status == "INACTIVE" {
					return nil
				}

				return fmt.Errorf("ECS Cluster %q is still %q", clusterName, *c.Status)
			}
		}

		if err != nil {
			return resource.RetryError{Err: err}
		}

		return nil
	})
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] ECS cluster %q deleted", d.Id())
	return nil
}
