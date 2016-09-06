package aws

import (
	"bytes"
	"fmt"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/applicationautoscaling"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsAppautoscalingPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAppautoscalingPolicyCreate,
		Read:   resourceAwsAppautoscalingPolicyRead,
		Update: resourceAwsAppautoscalingPolicyUpdate,
		Delete: resourceAwsAppautoscalingPolicyDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					// https://github.com/boto/botocore/blob/9f322b1/botocore/data/autoscaling/2011-01-01/service-2.json#L1862-L1873
					value := v.(string)
					if len(value) > 255 {
						errors = append(errors, fmt.Errorf("%s cannot be longer than 255 characters", k))
					}
					return
				},
			},
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"policy_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "StepScaling",
			},
			"resource_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"scalable_dimension": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateAppautoscalingScalableDimension,
			},
			"service_namespace": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateAppautoscalingServiceNamespace,
			},
			"adjustment_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"cooldown": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"metric_aggregation_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"min_adjustment_magnitude": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"alarms": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"step_adjustment": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"metric_interval_lower_bound": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"metric_interval_upper_bound": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"scaling_adjustment": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
				Set: resourceAwsAppautoscalingAdjustmentHash,
			},
		},
	}
}

func resourceAwsAppautoscalingPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appautoscalingconn

	params, err := getAwsAppautoscalingPutScalingPolicyInput(d)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] ApplicationAutoScaling PutScalingPolicy: %#v", params)
	resp, err := conn.PutScalingPolicy(&params)
	if err != nil {
		return fmt.Errorf("Error putting scaling policy: %s", err)
	}

	d.Set("arn", resp.PolicyARN)
	d.SetId(d.Get("name").(string))
	log.Printf("[INFO] ApplicationAutoScaling scaling PolicyARN: %s", d.Get("arn").(string))

	return resourceAwsAppautoscalingPolicyRead(d, meta)
}

func resourceAwsAppautoscalingPolicyRead(d *schema.ResourceData, meta interface{}) error {
	p, err := getAwsAppautoscalingPolicy(d, meta)
	if err != nil {
		return err
	}
	if p == nil {
		d.SetId("")
		return nil
	}

	log.Printf("[DEBUG] Read ApplicationAutoScaling policy: %s, SP: %s, Obj: %s", d.Get("name"), d.Get("name"), p)

	d.Set("arn", p.PolicyARN)
	d.Set("name", p.PolicyName)
	d.Set("policy_type", p.PolicyType)
	d.Set("resource_id", p.ResourceId)
	d.Set("scalable_dimension", p.ScalableDimension)
	d.Set("service_namespace", p.ServiceNamespace)
	d.Set("alarms", p.Alarms)
	d.Set("step_scaling_policy_configuration", p.StepScalingPolicyConfiguration)

	return nil
}

func resourceAwsAppautoscalingPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appautoscalingconn

	params, inputErr := getAwsAppautoscalingPutScalingPolicyInput(d)
	if inputErr != nil {
		return inputErr
	}

	log.Printf("[DEBUG] Application Autoscaling Update Scaling Policy: %#v", params)
	_, err := conn.PutScalingPolicy(&params)
	if err != nil {
		return err
	}

	return resourceAwsAppautoscalingPolicyRead(d, meta)
}

func resourceAwsAppautoscalingPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appautoscalingconn
	p, err := getAwsAppautoscalingPolicy(d, meta)
	if err != nil {
		return fmt.Errorf("Error getting policy: %s", err)
	}
	if p == nil {
		return nil
	}

	params := applicationautoscaling.DeleteScalingPolicyInput{
		PolicyName:        aws.String(d.Get("name").(string)),
		ResourceId:        aws.String(d.Get("resource_id").(string)),
		ScalableDimension: aws.String(d.Get("scalable_dimension").(string)),
		ServiceNamespace:  aws.String(d.Get("service_namespace").(string)),
	}
	log.Printf("[DEBUG] Deleting Application AutoScaling Policy opts: %#v", params)
	if _, err := conn.DeleteScalingPolicy(&params); err != nil {
		return fmt.Errorf("Application AutoScaling Policy: %s", err)
	}

	d.SetId("")
	return nil
}

// Takes the result of flatmap.Expand for an array of step adjustments and
// returns a []*applicationautoscaling.StepAdjustment.
func expandAppautoscalingStepAdjustments(configured []interface{}) ([]*applicationautoscaling.StepAdjustment, error) {
	var adjustments []*applicationautoscaling.StepAdjustment

	// Loop over our configured step adjustments and create an array
	// of aws-sdk-go compatible objects. We're forced to convert strings
	// to floats here because there's no way to detect whether or not
	// an uninitialized, optional schema element is "0.0" deliberately.
	// With strings, we can test for "", which is definitely an empty
	// struct value.
	for _, raw := range configured {
		data := raw.(map[string]interface{})
		a := &applicationautoscaling.StepAdjustment{
			ScalingAdjustment: aws.Int64(int64(data["scaling_adjustment"].(int))),
		}
		if data["metric_interval_lower_bound"] != "" {
			bound := data["metric_interval_lower_bound"]
			switch bound := bound.(type) {
			case string:
				f, err := strconv.ParseFloat(bound, 64)
				if err != nil {
					return nil, fmt.Errorf(
						"metric_interval_lower_bound must be a float value represented as a string")
				}
				a.MetricIntervalLowerBound = aws.Float64(f)
			default:
				return nil, fmt.Errorf(
					"metric_interval_lower_bound isn't a string. This is a bug. Please file an issue.")
			}
		}
		if data["metric_interval_upper_bound"] != "" {
			bound := data["metric_interval_upper_bound"]
			switch bound := bound.(type) {
			case string:
				f, err := strconv.ParseFloat(bound, 64)
				if err != nil {
					return nil, fmt.Errorf(
						"metric_interval_upper_bound must be a float value represented as a string")
				}
				a.MetricIntervalUpperBound = aws.Float64(f)
			default:
				return nil, fmt.Errorf(
					"metric_interval_upper_bound isn't a string. This is a bug. Please file an issue.")
			}
		}
		adjustments = append(adjustments, a)
	}

	return adjustments, nil
}

func getAwsAppautoscalingPutScalingPolicyInput(d *schema.ResourceData) (applicationautoscaling.PutScalingPolicyInput, error) {
	var params = applicationautoscaling.PutScalingPolicyInput{
		PolicyName: aws.String(d.Get("name").(string)),
		ResourceId: aws.String(d.Get("resource_id").(string)),
	}

	if v, ok := d.GetOk("policy_type"); ok {
		params.PolicyType = aws.String(v.(string))
	}

	if v, ok := d.GetOk("service_namespace"); ok {
		params.ServiceNamespace = aws.String(v.(string))
	}

	if v, ok := d.GetOk("scalable_dimension"); ok {
		params.ScalableDimension = aws.String(v.(string))
	}

	var adjustmentSteps []*applicationautoscaling.StepAdjustment
	if v, ok := d.GetOk("step_adjustment"); ok {
		steps, err := expandAppautoscalingStepAdjustments(v.(*schema.Set).List())
		if err != nil {
			return params, fmt.Errorf("metric_interval_lower_bound and metric_interval_upper_bound must be strings!")
		}
		adjustmentSteps = steps
	}

	// build StepScalingPolicyConfiguration
	params.StepScalingPolicyConfiguration = &applicationautoscaling.StepScalingPolicyConfiguration{
		AdjustmentType:        aws.String(d.Get("adjustment_type").(string)),
		Cooldown:              aws.Int64(int64(d.Get("cooldown").(int))),
		MetricAggregationType: aws.String(d.Get("metric_aggregation_type").(string)),
		StepAdjustments:       adjustmentSteps,
	}

	if v, ok := d.GetOk("min_adjustment_magnitude"); ok {
		params.StepScalingPolicyConfiguration.MinAdjustmentMagnitude = aws.Int64(int64(v.(int)))
	}

	return params, nil
}

func getAwsAppautoscalingPolicy(d *schema.ResourceData, meta interface{}) (*applicationautoscaling.ScalingPolicy, error) {
	conn := meta.(*AWSClient).appautoscalingconn

	params := applicationautoscaling.DescribeScalingPoliciesInput{
		PolicyNames:      []*string{aws.String(d.Get("name").(string))},
		ServiceNamespace: aws.String(d.Get("service_namespace").(string)),
	}

	log.Printf("[DEBUG] Application AutoScaling Policy Describe Params: %#v", params)
	resp, err := conn.DescribeScalingPolicies(&params)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving scaling policies: %s", err)
	}

	// find scaling policy
	name := d.Get("name")
	for idx, sp := range resp.ScalingPolicies {
		if *sp.PolicyName == name {
			return resp.ScalingPolicies[idx], nil
		}
	}

	// policy not found
	return nil, nil
}

func resourceAwsAppautoscalingAdjustmentHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	if v, ok := m["metric_interval_lower_bound"]; ok {
		buf.WriteString(fmt.Sprintf("%f-", v))
	}
	if v, ok := m["metric_interval_upper_bound"]; ok {
		buf.WriteString(fmt.Sprintf("%f-", v))
	}
	buf.WriteString(fmt.Sprintf("%d-", m["scaling_adjustment"].(int)))

	return hashcode.String(buf.String())
}
