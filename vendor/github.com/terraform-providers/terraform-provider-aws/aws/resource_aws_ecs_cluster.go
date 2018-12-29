package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsEcsCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEcsClusterCreate,
		Read:   resourceAwsEcsClusterRead,
		Update: resourceAwsEcsClusterUpdate,
		Delete: resourceAwsEcsClusterDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAwsEcsClusterImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"tags": tagsSchema(),
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsEcsClusterImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	d.Set("name", d.Id())
	d.SetId(arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Region:    meta.(*AWSClient).region,
		AccountID: meta.(*AWSClient).accountid,
		Service:   "ecs",
		Resource:  fmt.Sprintf("cluster/%s", d.Id()),
	}.String())
	return []*schema.ResourceData{d}, nil
}

func resourceAwsEcsClusterCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecsconn

	clusterName := d.Get("name").(string)
	log.Printf("[DEBUG] Creating ECS cluster %s", clusterName)

	out, err := conn.CreateCluster(&ecs.CreateClusterInput{
		ClusterName: aws.String(clusterName),
		Tags:        tagsFromMapECS(d.Get("tags").(map[string]interface{})),
	})
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] ECS cluster %s created", *out.Cluster.ClusterArn)

	d.SetId(*out.Cluster.ClusterArn)

	return resourceAwsEcsClusterRead(d, meta)
}

func resourceAwsEcsClusterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecsconn

	input := &ecs.DescribeClustersInput{
		Clusters: []*string{aws.String(d.Id())},
		Include:  []*string{aws.String(ecs.ClusterFieldTags)},
	}

	log.Printf("[DEBUG] Reading ECS Cluster: %s", input)
	var out *ecs.DescribeClustersOutput
	err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		var err error
		out, err = conn.DescribeClusters(input)

		if err != nil {
			return resource.NonRetryableError(err)
		}

		if out == nil || len(out.Failures) > 0 {
			if d.IsNewResource() {
				return resource.RetryableError(&resource.NotFoundError{})
			}
			return resource.NonRetryableError(&resource.NotFoundError{})
		}

		return nil
	})

	if isResourceNotFoundError(err) {
		log.Printf("[WARN] ECS Cluster (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading ECS Cluster (%s): %s", d.Id(), err)
	}

	var cluster *ecs.Cluster
	for _, c := range out.Clusters {
		if aws.StringValue(c.ClusterArn) == d.Id() {
			cluster = c
			break
		}
	}

	if cluster == nil {
		log.Printf("[WARN] ECS Cluster (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	// Status==INACTIVE means deleted cluster
	if aws.StringValue(cluster.Status) == "INACTIVE" {
		log.Printf("[WARN] ECS Cluster (%s) deleted, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("arn", cluster.ClusterArn)
	d.Set("name", cluster.ClusterName)

	if err := d.Set("tags", tagsToMapECS(cluster.Tags)); err != nil {
		return fmt.Errorf("error setting tags: %s", err)
	}

	return nil
}

func resourceAwsEcsClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecsconn

	if d.HasChange("tags") {
		oldTagsRaw, newTagsRaw := d.GetChange("tags")
		oldTagsMap := oldTagsRaw.(map[string]interface{})
		newTagsMap := newTagsRaw.(map[string]interface{})
		createTags, removeTags := diffTagsECS(tagsFromMapECS(oldTagsMap), tagsFromMapECS(newTagsMap))

		if len(removeTags) > 0 {
			removeTagKeys := make([]*string, len(removeTags))
			for i, removeTag := range removeTags {
				removeTagKeys[i] = removeTag.Key
			}

			input := &ecs.UntagResourceInput{
				ResourceArn: aws.String(d.Id()),
				TagKeys:     removeTagKeys,
			}

			log.Printf("[DEBUG] Untagging ECS Cluster: %s", input)
			if _, err := conn.UntagResource(input); err != nil {
				return fmt.Errorf("error untagging ECS Cluster (%s): %s", d.Id(), err)
			}
		}

		if len(createTags) > 0 {
			input := &ecs.TagResourceInput{
				ResourceArn: aws.String(d.Id()),
				Tags:        createTags,
			}

			log.Printf("[DEBUG] Tagging ECS Cluster: %s", input)
			if _, err := conn.TagResource(input); err != nil {
				return fmt.Errorf("error tagging ECS Cluster (%s): %s", d.Id(), err)
			}
		}
	}

	return nil
}

func resourceAwsEcsClusterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecsconn

	log.Printf("[DEBUG] Deleting ECS cluster %s", d.Id())

	err := resource.Retry(10*time.Minute, func() *resource.RetryError {
		out, err := conn.DeleteCluster(&ecs.DeleteClusterInput{
			Cluster: aws.String(d.Id()),
		})

		if err == nil {
			log.Printf("[DEBUG] ECS cluster %s deleted: %s", d.Id(), out)
			return nil
		}

		awsErr, ok := err.(awserr.Error)
		if !ok {
			return resource.NonRetryableError(err)
		}

		if awsErr.Code() == "ClusterContainsContainerInstancesException" {
			log.Printf("[TRACE] Retrying ECS cluster %q deletion after %q", d.Id(), awsErr.Code())
			return resource.RetryableError(err)
		}

		if awsErr.Code() == "ClusterContainsServicesException" {
			log.Printf("[TRACE] Retrying ECS cluster %q deletion after %q", d.Id(), awsErr.Code())
			return resource.RetryableError(err)
		}

		return resource.NonRetryableError(err)
	})
	if err != nil {
		return err
	}

	clusterName := d.Get("name").(string)
	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		log.Printf("[DEBUG] Checking if ECS Cluster %q is INACTIVE", d.Id())
		out, err := conn.DescribeClusters(&ecs.DescribeClustersInput{
			Clusters: []*string{aws.String(clusterName)},
		})

		for _, c := range out.Clusters {
			if *c.ClusterName == clusterName {
				if *c.Status == "INACTIVE" {
					return nil
				}

				return resource.RetryableError(
					fmt.Errorf("ECS Cluster %q is still %q", clusterName, *c.Status))
			}
		}

		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] ECS cluster %q deleted", d.Id())
	return nil
}
