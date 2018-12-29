package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/applicationautoscaling"
)

func resourceAwsAppautoscalingTarget() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAppautoscalingTargetPut,
		Read:   resourceAwsAppautoscalingTargetRead,
		Update: resourceAwsAppautoscalingTargetPut,
		Delete: resourceAwsAppautoscalingTargetDelete,

		Schema: map[string]*schema.Schema{
			"max_capacity": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"min_capacity": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"resource_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"role_arn": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"scalable_dimension": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"service_namespace": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsAppautoscalingTargetPut(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appautoscalingconn

	var targetOpts applicationautoscaling.RegisterScalableTargetInput

	targetOpts.MaxCapacity = aws.Int64(int64(d.Get("max_capacity").(int)))
	targetOpts.MinCapacity = aws.Int64(int64(d.Get("min_capacity").(int)))
	targetOpts.ResourceId = aws.String(d.Get("resource_id").(string))
	targetOpts.ScalableDimension = aws.String(d.Get("scalable_dimension").(string))
	targetOpts.ServiceNamespace = aws.String(d.Get("service_namespace").(string))

	if roleArn, exists := d.GetOk("role_arn"); exists {
		targetOpts.RoleARN = aws.String(roleArn.(string))
	}

	log.Printf("[DEBUG] Application autoscaling target create configuration %s", targetOpts)
	var err error
	err = resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, err = conn.RegisterScalableTarget(&targetOpts)

		if err != nil {
			if isAWSErr(err, "ValidationException", "Unable to assume IAM role") {
				log.Printf("[DEBUG] Retrying creation of Application Autoscaling Scalable Target due to possible issues with IAM: %s", err)
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("Error creating application autoscaling target: %s", err)
	}

	d.SetId(d.Get("resource_id").(string))
	log.Printf("[INFO] Application AutoScaling Target ID: %s", d.Id())

	return resourceAwsAppautoscalingTargetRead(d, meta)
}

func resourceAwsAppautoscalingTargetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appautoscalingconn

	namespace := d.Get("service_namespace").(string)
	dimension := d.Get("scalable_dimension").(string)
	t, err := getAwsAppautoscalingTarget(d.Id(), namespace, dimension, conn)
	if err != nil {
		return err
	}
	if t == nil {
		log.Printf("[WARN] Application AutoScaling Target (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("max_capacity", t.MaxCapacity)
	d.Set("min_capacity", t.MinCapacity)
	d.Set("resource_id", t.ResourceId)
	d.Set("role_arn", t.RoleARN)
	d.Set("scalable_dimension", t.ScalableDimension)
	d.Set("service_namespace", t.ServiceNamespace)

	return nil
}

func resourceAwsAppautoscalingTargetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appautoscalingconn

	namespace := d.Get("service_namespace").(string)
	dimension := d.Get("scalable_dimension").(string)

	t, err := getAwsAppautoscalingTarget(d.Id(), namespace, dimension, conn)
	if err != nil {
		return err
	}
	if t == nil {
		log.Printf("[INFO] Application AutoScaling Target %q not found", d.Id())
		return nil
	}

	log.Printf("[DEBUG] Application AutoScaling Target destroy: %#v", d.Id())
	deleteOpts := applicationautoscaling.DeregisterScalableTargetInput{
		ResourceId:        aws.String(d.Get("resource_id").(string)),
		ServiceNamespace:  aws.String(d.Get("service_namespace").(string)),
		ScalableDimension: aws.String(d.Get("scalable_dimension").(string)),
	}

	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		if _, err := conn.DeregisterScalableTarget(&deleteOpts); err != nil {
			if awserr, ok := err.(awserr.Error); ok {
				// @TODO: We should do stuff here depending on the actual error returned
				return resource.RetryableError(awserr)
			}
			// Non recognized error, no retry.
			return resource.NonRetryableError(err)
		}
		// Successful delete
		return nil
	})
	if err != nil {
		return err
	}

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		if t, _ = getAwsAppautoscalingTarget(d.Id(), namespace, dimension, conn); t != nil {
			return resource.RetryableError(
				fmt.Errorf("Application AutoScaling Target still exists"))
		}
		return nil
	})
}

func getAwsAppautoscalingTarget(resourceId, namespace, dimension string,
	conn *applicationautoscaling.ApplicationAutoScaling) (*applicationautoscaling.ScalableTarget, error) {

	describeOpts := applicationautoscaling.DescribeScalableTargetsInput{
		ResourceIds:      []*string{aws.String(resourceId)},
		ServiceNamespace: aws.String(namespace),
	}

	log.Printf("[DEBUG] Application AutoScaling Target describe configuration: %#v", describeOpts)
	describeTargets, err := conn.DescribeScalableTargets(&describeOpts)
	if err != nil {
		// @TODO: We should probably send something else back if we're trying to access an unknown Resource ID
		// targetserr, ok := err.(awserr.Error)
		// if ok && targetserr.Code() == ""
		return nil, fmt.Errorf("Error retrieving Application AutoScaling Target: %s", err)
	}

	for idx, tgt := range describeTargets.ScalableTargets {
		if *tgt.ResourceId == resourceId && *tgt.ScalableDimension == dimension {
			return describeTargets.ScalableTargets[idx], nil
		}
	}

	return nil, nil
}
