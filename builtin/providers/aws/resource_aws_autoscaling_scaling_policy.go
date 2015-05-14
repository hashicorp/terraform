package aws

import (
	"fmt"
	"log"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsAutoscalingScalingPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAutoscalingScalingPolicyCreate,
		Read:   resourceAwsAutoscalingScalingPolicyRead,
		Update: resourceAwsAutoscalingScalingPolicyUpdate,
		Delete: resourceAwsAutoscalingScalingPolicyDelete,

		Schema: map[string]*schema.Schema{
			"adjustment_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"autoscaling_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"cooldown": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"min_adjustment_step": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"policy_arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"policy_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"scaling_adjustment": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
		},
	}
}

func resourceAwsAutoscalingScalingPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	params := getAwsAutoscalingPutScalingPolicyInput(d)

	log.Printf("[DEBUG] AutoScaling PutScalingPolicy: %#v", params)
	resp, err := autoscalingconn.PutScalingPolicy(&params)
	if err != nil {
		return fmt.Errorf("Error putting scaling policy: %s", err)
	}

	d.Set("policy_arn", resp.PolicyARN)
	d.SetId(d.Get("policy_name").(string))
	log.Printf("[INFO] AutoScaling Scaling PolicyARN: %s", d.Get("policy_arn").(string))

	return resourceAwsAutoscalingScalingPolicyRead(d, meta)
}

func resourceAwsAutoscalingScalingPolicyRead(d *schema.ResourceData, meta interface{}) error {
	p, err := getAwsAutoscalingScalingPolicy(d, meta)
	if err != nil {
		return err
	}
	if p == nil {
		d.SetId("")
		return nil
	}

	log.Printf("[DEBUG] Read Scaling Policy: ASG: %s, SP: %s, Obj: %#v", d.Get("autoscaling_group_name"), d.Get("policy_name"), p)

	d.Set("adjustment_type", p.AdjustmentType)
	d.Set("autoscaling_group_name", p.AutoScalingGroupName)
	d.Set("cooldown", p.Cooldown)
	d.Set("min_adjustment_step", p.MinAdjustmentStep)
	d.Set("policy_arn", p.PolicyARN)
	d.Set("policy_name", p.PolicyName)
	d.Set("scaling_adjustment", p.ScalingAdjustment)

	return nil
}

func resourceAwsAutoscalingScalingPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	params := getAwsAutoscalingPutScalingPolicyInput(d)

	log.Printf("[DEBUG] Autoscaling Update Scaling Policy: %#v", params)
	_, err := autoscalingconn.PutScalingPolicy(&params)
	if err != nil {
		return err
	}

	return resourceAwsAutoscalingScalingPolicyRead(d, meta)
}

func resourceAwsAutoscalingScalingPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn
	p, err := getAwsAutoscalingScalingPolicy(d, meta)
	if err != nil {
		return err
	}
	if p == nil {
		return nil
	}

	params := autoscaling.DeletePolicyInput{
		AutoScalingGroupName: aws.String(d.Get("autoscaling_group_name").(string)),
		PolicyName:           aws.String(d.Get("policy_name").(string)),
	}
	if _, err := autoscalingconn.DeletePolicy(&params); err != nil {
		return fmt.Errorf("Autoscaling Scaling Policy: %s ", err)
	}

	d.SetId("")
	return nil
}

// PutScalingPolicy seems to require all params to be resent, so create and update can share this common function
func getAwsAutoscalingPutScalingPolicyInput(d *schema.ResourceData) autoscaling.PutScalingPolicyInput {
	var params = autoscaling.PutScalingPolicyInput{
		AutoScalingGroupName: aws.String(d.Get("autoscaling_group_name").(string)),
		PolicyName:           aws.String(d.Get("policy_name").(string)),
	}

	if v, ok := d.GetOk("adjustment_type"); ok {
		params.AdjustmentType = aws.String(v.(string))
	}

	if v, ok := d.GetOk("cooldown"); ok {
		params.Cooldown = aws.Long(int64(v.(int)))
	}

	if v, ok := d.GetOk("scaling_adjustment"); ok {
		params.ScalingAdjustment = aws.Long(int64(v.(int)))
	}

	if v, ok := d.GetOk("min_adjustment_step"); ok {
		params.MinAdjustmentStep = aws.Long(int64(v.(int)))
	}

	return params
}

func getAwsAutoscalingScalingPolicy(d *schema.ResourceData, meta interface{}) (*autoscaling.ScalingPolicy, error) {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	params := autoscaling.DescribePoliciesInput{
		AutoScalingGroupName: aws.String(d.Get("autoscaling_group_name").(string)),
		PolicyNames:          []*string{aws.String(d.Get("policy_name").(string))},
	}

	log.Printf("[DEBUG] AutoScaling Scaling Policy Describe Params: %#v", params)
	resp, err := autoscalingconn.DescribePolicies(&params)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving scaling policies: %s", err)
	}

	// find scaling policy
	policy_name := d.Get("policy_name")
	for idx, sp := range resp.ScalingPolicies {
		if *sp.PolicyName == policy_name {
			return resp.ScalingPolicies[idx], nil
		}
	}

	// policy not found
	return nil, nil
}
