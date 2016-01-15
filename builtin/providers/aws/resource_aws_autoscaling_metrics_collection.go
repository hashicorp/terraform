package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsAutoscalingMetric() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAutoscalingMetricCreate,
		Read:   resourceAwsAutoscalingMetricRead,
		Update: resourceAwsAutoscalingMetricUpdate,
		Delete: resourceAwsAutoscalingMetricDelete,

		Schema: map[string]*schema.Schema{
			"autoscaling_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"metrics": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"granularity": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsAutoscalingMetricCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn

	asgName := d.Get("autoscaling_group_name").(string)
	props := &autoscaling.EnableMetricsCollectionInput{
		AutoScalingGroupName: aws.String(asgName),
		Granularity:          aws.String(d.Get("granularity").(string)),
		Metrics:              expandStringList(d.Get("metrics").(*schema.Set).List()),
	}

	log.Printf("[INFO] Enabling metrics collection for the ASG: %s", asgName)
	_, err := conn.EnableMetricsCollection(props)
	if err != nil {
		return err
	}

	d.SetId(d.Get("autoscaling_group_name").(string))

	return resourceAwsAutoscalingMetricRead(d, meta)
}

func resourceAwsAutoscalingMetricRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn

	g, err := getAwsAutoscalingGroup(d.Get("autoscaling_group_name").(string), conn)
	if err != nil {
		return err
	}
	if g == nil {
		return nil
	}

	if g.EnabledMetrics != nil && len(g.EnabledMetrics) > 0 {
		if err := d.Set("metrics", flattenAsgEnabledMetrics(g.EnabledMetrics)); err != nil {
			log.Printf("[WARN] Error setting metrics for (%s): %s", d.Id(), err)
		}
		d.Set("granularity", g.EnabledMetrics[0].Granularity)
	}

	return nil
}

func resourceAwsAutoscalingMetricUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn

	if d.HasChange("metrics") {
		o, n := d.GetChange("metrics")
		if o == nil {
			o = new(schema.Set)
		}
		if n == nil {
			n = new(schema.Set)
		}

		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		disableMetrics := os.Difference(ns)
		if disableMetrics.Len() != 0 {
			props := &autoscaling.DisableMetricsCollectionInput{
				AutoScalingGroupName: aws.String(d.Id()),
				Metrics:              expandStringList(disableMetrics.List()),
			}

			_, err := conn.DisableMetricsCollection(props)
			if err != nil {
				return fmt.Errorf("Failure to Disable metrics collection types for ASG %s: %s", d.Id(), err)
			}
		}

		enabledMetrics := ns.Difference(os)
		if enabledMetrics.Len() != 0 {
			props := &autoscaling.EnableMetricsCollectionInput{
				AutoScalingGroupName: aws.String(d.Id()),
				Metrics:              expandStringList(enabledMetrics.List()),
			}

			_, err := conn.EnableMetricsCollection(props)
			if err != nil {
				return fmt.Errorf("Failure to Enable metrics collection types for ASG %s: %s", d.Id(), err)
			}
		}
	}

	return resourceAwsAutoscalingMetricRead(d, meta)
}

func resourceAwsAutoscalingMetricDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).autoscalingconn

	props := &autoscaling.DisableMetricsCollectionInput{
		AutoScalingGroupName: aws.String(d.Id()),
	}

	log.Printf("[INFO] Disabling ALL metrics collection for the ASG: %s", d.Id())
	_, err := conn.DisableMetricsCollection(props)
	if err != nil {
		return err
	}

	return nil
}
