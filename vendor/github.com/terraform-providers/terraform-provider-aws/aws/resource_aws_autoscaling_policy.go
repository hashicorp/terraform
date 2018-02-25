package aws

import (
	"bytes"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsAutoscalingPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAutoscalingPolicyCreate,
		Read:   resourceAwsAutoscalingPolicyRead,
		Update: resourceAwsAutoscalingPolicyUpdate,
		Delete: resourceAwsAutoscalingPolicyDelete,

		Schema: map[string]*schema.Schema{
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"adjustment_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"autoscaling_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"policy_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "SimpleScaling", // preserve AWS's default to make validation easier.
			},
			"cooldown": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"estimated_instance_warmup": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"metric_aggregation_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"min_adjustment_magnitude": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(1),
			},
			"min_adjustment_step": &schema.Schema{
				Type:          schema.TypeInt,
				Optional:      true,
				Deprecated:    "Use min_adjustment_magnitude instead, otherwise you may see a perpetual diff on this resource.",
				ConflictsWith: []string{"min_adjustment_magnitude"},
			},
			"scaling_adjustment": &schema.Schema{
				Type:          schema.TypeInt,
				Optional:      true,
				ConflictsWith: []string{"step_adjustment"},
			},
			"step_adjustment": &schema.Schema{
				Type:          schema.TypeSet,
				Optional:      true,
				ConflictsWith: []string{"scaling_adjustment"},
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
				Set: resourceAwsAutoscalingScalingAdjustmentHash,
			},
			"target_tracking_configuration": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"predefined_metric_specification": &schema.Schema{
							Type:          schema.TypeList,
							Optional:      true,
							MaxItems:      1,
							ConflictsWith: []string{"target_tracking_configuration.0.customized_metric_specification"},
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"predefined_metric_type": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"resource_label": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"customized_metric_specification": &schema.Schema{
							Type:          schema.TypeList,
							Optional:      true,
							MaxItems:      1,
							ConflictsWith: []string{"target_tracking_configuration.0.predefined_metric_specification"},
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"metric_dimension": &schema.Schema{
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"name": &schema.Schema{
													Type:     schema.TypeString,
													Required: true,
												},
												"value": &schema.Schema{
													Type:     schema.TypeString,
													Required: true,
												},
											},
										},
									},
									"metric_name": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"namespace": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"statistic": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"unit": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"target_value": &schema.Schema{
							Type:     schema.TypeFloat,
							Required: true,
						},
						"disable_scale_in": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},
		},
	}
}

func resourceAwsAutoscalingPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	params, err := getAwsAutoscalingPutScalingPolicyInput(d)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] AutoScaling PutScalingPolicy: %#v", params)
	resp, err := autoscalingconn.PutScalingPolicy(&params)
	if err != nil {
		return fmt.Errorf("Error putting scaling policy: %s", err)
	}

	d.Set("arn", resp.PolicyARN)
	d.SetId(d.Get("name").(string))
	log.Printf("[INFO] AutoScaling Scaling PolicyARN: %s", d.Get("arn").(string))

	return resourceAwsAutoscalingPolicyRead(d, meta)
}

func resourceAwsAutoscalingPolicyRead(d *schema.ResourceData, meta interface{}) error {
	p, err := getAwsAutoscalingPolicy(d, meta)
	if err != nil {
		return err
	}
	if p == nil {
		log.Printf("[WARN] Autoscaling Policy (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	log.Printf("[DEBUG] Read Scaling Policy: ASG: %s, SP: %s, Obj: %s", d.Get("autoscaling_group_name"), d.Get("name"), p)

	d.Set("adjustment_type", p.AdjustmentType)
	d.Set("autoscaling_group_name", p.AutoScalingGroupName)
	d.Set("cooldown", p.Cooldown)
	d.Set("estimated_instance_warmup", p.EstimatedInstanceWarmup)
	d.Set("metric_aggregation_type", p.MetricAggregationType)
	d.Set("policy_type", p.PolicyType)
	if p.MinAdjustmentMagnitude != nil {
		d.Set("min_adjustment_magnitude", p.MinAdjustmentMagnitude)
		d.Set("min_adjustment_step", 0)
	} else {
		d.Set("min_adjustment_step", p.MinAdjustmentStep)
	}
	d.Set("arn", p.PolicyARN)
	d.Set("name", p.PolicyName)
	d.Set("scaling_adjustment", p.ScalingAdjustment)
	d.Set("step_adjustment", flattenStepAdjustments(p.StepAdjustments))
	d.Set("target_tracking_configuration", flattenTargetTrackingConfiguration(p.TargetTrackingConfiguration))

	return nil
}

func resourceAwsAutoscalingPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	params, inputErr := getAwsAutoscalingPutScalingPolicyInput(d)
	if inputErr != nil {
		return inputErr
	}

	log.Printf("[DEBUG] Autoscaling Update Scaling Policy: %#v", params)
	_, err := autoscalingconn.PutScalingPolicy(&params)
	if err != nil {
		return err
	}

	return resourceAwsAutoscalingPolicyRead(d, meta)
}

func resourceAwsAutoscalingPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn
	p, err := getAwsAutoscalingPolicy(d, meta)
	if err != nil {
		return err
	}
	if p == nil {
		return nil
	}

	params := autoscaling.DeletePolicyInput{
		AutoScalingGroupName: aws.String(d.Get("autoscaling_group_name").(string)),
		PolicyName:           aws.String(d.Get("name").(string)),
	}
	log.Printf("[DEBUG] Deleting Autoscaling Policy opts: %s", params)
	if _, err := autoscalingconn.DeletePolicy(&params); err != nil {
		return fmt.Errorf("Autoscaling Scaling Policy: %s ", err)
	}

	d.SetId("")
	return nil
}

// PutScalingPolicy can safely resend all parameters without destroying the
// resource, so create and update can share this common function. It will error
// if certain mutually exclusive values are set.
func getAwsAutoscalingPutScalingPolicyInput(d *schema.ResourceData) (autoscaling.PutScalingPolicyInput, error) {
	var params = autoscaling.PutScalingPolicyInput{
		AutoScalingGroupName: aws.String(d.Get("autoscaling_group_name").(string)),
		PolicyName:           aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("adjustment_type"); ok {
		params.AdjustmentType = aws.String(v.(string))
	}

	if v, ok := d.GetOkExists("cooldown"); ok {
		params.Cooldown = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOkExists("estimated_instance_warmup"); ok {
		params.EstimatedInstanceWarmup = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("metric_aggregation_type"); ok {
		params.MetricAggregationType = aws.String(v.(string))
	}

	if v, ok := d.GetOk("policy_type"); ok {
		params.PolicyType = aws.String(v.(string))
	}

	//if policy_type=="SimpleScaling" then scaling_adjustment is required and 0 is allowed
	if v, ok := d.GetOkExists("scaling_adjustment"); ok || *params.PolicyType == "SimpleScaling" {
		params.ScalingAdjustment = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("step_adjustment"); ok {
		steps, err := expandStepAdjustments(v.(*schema.Set).List())
		if err != nil {
			return params, fmt.Errorf("metric_interval_lower_bound and metric_interval_upper_bound must be strings!")
		}
		params.StepAdjustments = steps
	}

	if v, ok := d.GetOkExists("min_adjustment_magnitude"); ok {
		// params.MinAdjustmentMagnitude = aws.Int64(int64(d.Get("min_adjustment_magnitude").(int)))
		params.MinAdjustmentMagnitude = aws.Int64(int64(v.(int)))
	} else if v, ok := d.GetOkExists("min_adjustment_step"); ok {
		// params.MinAdjustmentStep = aws.Int64(int64(d.Get("min_adjustment_step").(int)))
		params.MinAdjustmentStep = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("target_tracking_configuration"); ok {
		params.TargetTrackingConfiguration = expandTargetTrackingConfiguration(v.([]interface{}))
	}

	// Validate our final input to confirm it won't error when sent to AWS.
	// First, SimpleScaling policy types...
	if *params.PolicyType == "SimpleScaling" && params.StepAdjustments != nil {
		return params, fmt.Errorf("SimpleScaling policy types cannot use step_adjustments!")
	}
	if *params.PolicyType == "SimpleScaling" && params.MetricAggregationType != nil {
		return params, fmt.Errorf("SimpleScaling policy types cannot use metric_aggregation_type!")
	}
	if *params.PolicyType == "SimpleScaling" && params.EstimatedInstanceWarmup != nil {
		return params, fmt.Errorf("SimpleScaling policy types cannot use estimated_instance_warmup!")
	}
	if *params.PolicyType == "SimpleScaling" && params.TargetTrackingConfiguration != nil {
		return params, fmt.Errorf("SimpleScaling policy types cannot use target_tracking_configuration!")
	}

	// Second, StepScaling policy types...
	if *params.PolicyType == "StepScaling" && params.ScalingAdjustment != nil {
		return params, fmt.Errorf("StepScaling policy types cannot use scaling_adjustment!")
	}
	if *params.PolicyType == "StepScaling" && params.Cooldown != nil {
		return params, fmt.Errorf("StepScaling policy types cannot use cooldown!")
	}
	if *params.PolicyType == "StepScaling" && params.TargetTrackingConfiguration != nil {
		return params, fmt.Errorf("StepScaling policy types cannot use target_tracking_configuration!")
	}

	// Third, TargetTrackingScaling policy types...
	if *params.PolicyType == "TargetTrackingScaling" && params.AdjustmentType != nil {
		return params, fmt.Errorf("TargetTrackingScaling policy types cannot use adjustment_type!")
	}
	if *params.PolicyType == "TargetTrackingScaling" && params.Cooldown != nil {
		return params, fmt.Errorf("TargetTrackingScaling policy types cannot use cooldown!")
	}
	if *params.PolicyType == "TargetTrackingScaling" && params.MetricAggregationType != nil {
		return params, fmt.Errorf("TargetTrackingScaling policy types cannot use metric_aggregation_type!")
	}
	if *params.PolicyType == "TargetTrackingScaling" && params.MinAdjustmentMagnitude != nil {
		return params, fmt.Errorf("TargetTrackingScaling policy types cannot use min_adjustment_magnitude!")
	}
	if *params.PolicyType == "TargetTrackingScaling" && params.ScalingAdjustment != nil {
		return params, fmt.Errorf("TargetTrackingScaling policy types cannot use scaling_adjustment!")
	}
	if *params.PolicyType == "TargetTrackingScaling" && params.StepAdjustments != nil {
		return params, fmt.Errorf("TargetTrackingScaling policy types cannot use step_adjustments!")
	}

	return params, nil
}

func getAwsAutoscalingPolicy(d *schema.ResourceData, meta interface{}) (*autoscaling.ScalingPolicy, error) {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	params := autoscaling.DescribePoliciesInput{
		AutoScalingGroupName: aws.String(d.Get("autoscaling_group_name").(string)),
		PolicyNames:          []*string{aws.String(d.Get("name").(string))},
	}

	log.Printf("[DEBUG] AutoScaling Scaling Policy Describe Params: %#v", params)
	resp, err := autoscalingconn.DescribePolicies(&params)
	if err != nil {
		//A ValidationError here can mean that either the Policy is missing OR the Autoscaling Group is missing
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "ValidationError" {
			log.Printf("[WARN] Autoscaling Policy (%s) not found, removing from state", d.Id())
			d.SetId("")

			return nil, nil
		}
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

func resourceAwsAutoscalingScalingAdjustmentHash(v interface{}) int {
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

func expandTargetTrackingConfiguration(configs []interface{}) *autoscaling.TargetTrackingConfiguration {
	if len(configs) < 1 {
		return nil
	}

	config := configs[0].(map[string]interface{})

	result := &autoscaling.TargetTrackingConfiguration{}

	result.TargetValue = aws.Float64(config["target_value"].(float64))
	if v, ok := config["disable_scale_in"]; ok {
		result.DisableScaleIn = aws.Bool(v.(bool))
	}
	if v, ok := config["predefined_metric_specification"]; ok && len(v.([]interface{})) > 0 {
		spec := v.([]interface{})[0].(map[string]interface{})
		predSpec := &autoscaling.PredefinedMetricSpecification{
			PredefinedMetricType: aws.String(spec["predefined_metric_type"].(string)),
		}
		if val, ok := spec["resource_label"]; ok && val.(string) != "" {
			predSpec.ResourceLabel = aws.String(val.(string))
		}
		result.PredefinedMetricSpecification = predSpec
	}
	if v, ok := config["customized_metric_specification"]; ok && len(v.([]interface{})) > 0 {
		spec := v.([]interface{})[0].(map[string]interface{})
		customSpec := &autoscaling.CustomizedMetricSpecification{
			Namespace:  aws.String(spec["namespace"].(string)),
			MetricName: aws.String(spec["metric_name"].(string)),
			Statistic:  aws.String(spec["statistic"].(string)),
		}
		if val, ok := spec["unit"]; ok {
			customSpec.Unit = aws.String(val.(string))
		}
		if val, ok := spec["metric_dimension"]; ok {
			dims := val.([]interface{})
			metDimList := make([]*autoscaling.MetricDimension, len(dims))
			for i := range metDimList {
				dim := dims[i].(map[string]interface{})
				md := &autoscaling.MetricDimension{
					Name:  aws.String(dim["name"].(string)),
					Value: aws.String(dim["value"].(string)),
				}
				metDimList[i] = md
			}
			customSpec.Dimensions = metDimList
		}
		result.CustomizedMetricSpecification = customSpec
	}
	return result
}

func flattenTargetTrackingConfiguration(config *autoscaling.TargetTrackingConfiguration) []interface{} {
	if config == nil {
		return []interface{}{}
	}

	result := map[string]interface{}{}
	result["disable_scale_in"] = *config.DisableScaleIn
	result["target_value"] = *config.TargetValue
	if config.PredefinedMetricSpecification != nil {
		spec := map[string]interface{}{}
		spec["predefined_metric_type"] = *config.PredefinedMetricSpecification.PredefinedMetricType
		if config.PredefinedMetricSpecification.ResourceLabel != nil {
			spec["resource_label"] = *config.PredefinedMetricSpecification.ResourceLabel
		}
		result["predefined_metric_specification"] = []map[string]interface{}{spec}
	}
	if config.CustomizedMetricSpecification != nil {
		spec := map[string]interface{}{}
		spec["metric_name"] = *config.CustomizedMetricSpecification.MetricName
		spec["namespace"] = *config.CustomizedMetricSpecification.Namespace
		spec["statistic"] = *config.CustomizedMetricSpecification.Statistic
		if config.CustomizedMetricSpecification.Unit != nil {
			spec["unit"] = *config.CustomizedMetricSpecification.Unit
		}
		if config.CustomizedMetricSpecification.Dimensions != nil {
			dimSpec := make([]interface{}, len(config.CustomizedMetricSpecification.Dimensions))
			for i := range dimSpec {
				dim := map[string]interface{}{}
				rawDim := config.CustomizedMetricSpecification.Dimensions[i]
				dim["name"] = *rawDim.Name
				dim["value"] = *rawDim.Value
				dimSpec[i] = dim
			}
			spec["metric_dimension"] = dimSpec
		}
		result["customized_metric_specification"] = []map[string]interface{}{spec}
	}
	return []interface{}{result}
}
