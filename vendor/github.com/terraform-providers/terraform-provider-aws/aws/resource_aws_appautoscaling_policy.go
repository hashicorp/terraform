package aws

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/applicationautoscaling"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
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
			"step_scaling_policy_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"adjustment_type": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"cooldown": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"metric_aggregation_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"min_adjustment_magnitude": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"step_adjustment": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"metric_interval_lower_bound": &schema.Schema{
										Type:     schema.TypeFloat,
										Optional: true,
										Default:  -1,
									},
									"metric_interval_upper_bound": &schema.Schema{
										Type:     schema.TypeFloat,
										Optional: true,
										Default:  -1,
									},
									"scaling_adjustment": &schema.Schema{
										Type:     schema.TypeInt,
										Required: true,
									},
								},
							},
						},
					},
				},
			},
			"alarms": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"adjustment_type": &schema.Schema{
				Type:       schema.TypeString,
				Optional:   true,
				Deprecated: "Use step_scaling_policy_configuration -> adjustment_type instead",
			},
			"cooldown": &schema.Schema{
				Type:       schema.TypeInt,
				Optional:   true,
				Deprecated: "Use step_scaling_policy_configuration -> cooldown instead",
			},
			"metric_aggregation_type": &schema.Schema{
				Type:       schema.TypeString,
				Optional:   true,
				Deprecated: "Use step_scaling_policy_configuration -> metric_aggregation_type instead",
			},
			"min_adjustment_magnitude": &schema.Schema{
				Type:       schema.TypeInt,
				Optional:   true,
				Deprecated: "Use step_scaling_policy_configuration -> min_adjustment_magnitude instead",
			},
			"step_adjustment": &schema.Schema{
				Type:       schema.TypeSet,
				Optional:   true,
				Deprecated: "Use step_scaling_policy_configuration -> step_adjustment instead",
				Set:        resourceAwsAppautoscalingAdjustmentHash,
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
			},
			"target_tracking_scaling_policy_configuration": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"customized_metric_specification": &schema.Schema{
							Type:          schema.TypeList,
							MaxItems:      1,
							Optional:      true,
							ConflictsWith: []string{"target_tracking_scaling_policy_configuration.0.predefined_metric_specification"},
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"dimensions": &schema.Schema{
										Type:     schema.TypeSet,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"name": {
													Type:     schema.TypeString,
													Required: true,
												},
												"value": {
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
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateAppautoscalingCustomizedMetricSpecificationStatistic,
									},
									"unit": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"predefined_metric_specification": &schema.Schema{
							Type:          schema.TypeList,
							MaxItems:      1,
							Optional:      true,
							ConflictsWith: []string{"target_tracking_scaling_policy_configuration.0.customized_metric_specification"},
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"predefined_metric_type": &schema.Schema{
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateAppautoscalingPredefinedMetricSpecification,
									},
									"resource_label": &schema.Schema{
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validateAppautoscalingPredefinedResourceLabel,
									},
								},
							},
						},
						"disable_scale_in": &schema.Schema{
							Type:     schema.TypeBool,
							Default:  false,
							Optional: true,
						},
						"scale_in_cooldown": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"scale_out_cooldown": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"target_value": &schema.Schema{
							Type:     schema.TypeFloat,
							Required: true,
						},
					},
				},
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
	var resp *applicationautoscaling.PutScalingPolicyOutput
	err = resource.Retry(1*time.Minute, func() *resource.RetryError {
		var err error
		resp, err = conn.PutScalingPolicy(&params)
		if err != nil {
			if isAWSErr(err, "FailedResourceAccessException", "Rate exceeded") {
				return resource.RetryableError(err)
			}
			if isAWSErr(err, "FailedResourceAccessException", "is not authorized to perform") {
				return resource.RetryableError(err)
			}
			if isAWSErr(err, "FailedResourceAccessException", "token included in the request is invalid") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(fmt.Errorf("Error putting scaling policy: %s", err))
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Failed to create scaling policy: %s", err)
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
		log.Printf("[WARN] Application AutoScaling Policy (%s) not found, removing from state", d.Id())
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
	d.Set("step_scaling_policy_configuration", flattenStepScalingPolicyConfiguration(p.StepScalingPolicyConfiguration))
	d.Set("target_tracking_scaling_policy_configuration",
		flattenTargetTrackingScalingPolicyConfiguration(p.TargetTrackingScalingPolicyConfiguration))

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
		return fmt.Errorf("Failed to update scaling policy: %s", err)
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
		return fmt.Errorf("Failed to delete autoscaling policy: %s", err)
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
			case float64:
				if bound >= 0 {
					a.MetricIntervalLowerBound = aws.Float64(bound)
				}
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
			case float64:
				if bound >= 0 {
					a.MetricIntervalUpperBound = aws.Float64(bound)
				}
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

func expandAppautoscalingCustomizedMetricSpecification(configured []interface{}) *applicationautoscaling.CustomizedMetricSpecification {
	spec := &applicationautoscaling.CustomizedMetricSpecification{}

	for _, raw := range configured {
		data := raw.(map[string]interface{})
		if v, ok := data["metric_name"]; ok {
			spec.MetricName = aws.String(v.(string))
		}

		if v, ok := data["namespace"]; ok {
			spec.Namespace = aws.String(v.(string))
		}

		if v, ok := data["unit"].(string); ok && v != "" {
			spec.Unit = aws.String(v)
		}

		if v, ok := data["statistic"]; ok {
			spec.Statistic = aws.String(v.(string))
		}

		if s, ok := data["dimensions"].(*schema.Set); ok && s.Len() > 0 {
			dimensions := make([]*applicationautoscaling.MetricDimension, s.Len(), s.Len())
			for i, d := range s.List() {
				dimension := d.(map[string]interface{})
				dimensions[i] = &applicationautoscaling.MetricDimension{
					Name:  aws.String(dimension["name"].(string)),
					Value: aws.String(dimension["value"].(string)),
				}
			}
			spec.Dimensions = dimensions
		}
	}
	return spec
}

func expandAppautoscalingPredefinedMetricSpecification(configured []interface{}) *applicationautoscaling.PredefinedMetricSpecification {
	spec := &applicationautoscaling.PredefinedMetricSpecification{}

	for _, raw := range configured {
		data := raw.(map[string]interface{})

		if v, ok := data["predefined_metric_type"]; ok {
			spec.PredefinedMetricType = aws.String(v.(string))
		}

		if v, ok := data["resource_label"].(string); ok && v != "" {
			spec.ResourceLabel = aws.String(v)
		}
	}
	return spec
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

	// Deprecated fields
	// TODO: Remove in next major version
	at, atOk := d.GetOk("adjustment_type")
	cd, cdOk := d.GetOk("cooldown")
	mat, matOk := d.GetOk("metric_aggregation_type")
	mam, mamOk := d.GetOk("min_adjustment_magnitude")
	sa, saOk := d.GetOk("step_adjustment")
	if atOk || cdOk || matOk || mamOk || saOk {
		cfg := &applicationautoscaling.StepScalingPolicyConfiguration{}

		if atOk {
			cfg.AdjustmentType = aws.String(at.(string))
		}

		if cdOk {
			cfg.Cooldown = aws.Int64(int64(cd.(int)))
		}

		if matOk {
			cfg.MetricAggregationType = aws.String(mat.(string))
		}

		if saOk {
			steps, err := expandAppautoscalingStepAdjustments(sa.(*schema.Set).List())
			if err != nil {
				return params, fmt.Errorf("metric_interval_lower_bound and metric_interval_upper_bound must be strings!")
			}
			cfg.StepAdjustments = steps
		}

		if mamOk {
			cfg.MinAdjustmentMagnitude = aws.Int64(int64(mam.(int)))
		}

		params.StepScalingPolicyConfiguration = cfg
	}

	if v, ok := d.GetOk("step_scaling_policy_configuration"); ok {
		params.StepScalingPolicyConfiguration = expandStepScalingPolicyConfiguration(v.([]interface{}))
	}

	if l, ok := d.GetOk("target_tracking_scaling_policy_configuration"); ok {
		v := l.([]interface{})
		if len(v) < 1 {
			return params, fmt.Errorf("Empty target_tracking_scaling_policy_configuration block")
		}
		ttspCfg := v[0].(map[string]interface{})
		cfg := &applicationautoscaling.TargetTrackingScalingPolicyConfiguration{
			TargetValue: aws.Float64(ttspCfg["target_value"].(float64)),
		}

		if v, ok := ttspCfg["scale_in_cooldown"]; ok {
			cfg.ScaleInCooldown = aws.Int64(int64(v.(int)))
		}

		if v, ok := ttspCfg["scale_out_cooldown"]; ok {
			cfg.ScaleOutCooldown = aws.Int64(int64(v.(int)))
		}

		if v, ok := ttspCfg["disable_scale_in"]; ok {
			cfg.DisableScaleIn = aws.Bool(v.(bool))
		}

		if v, ok := ttspCfg["customized_metric_specification"].([]interface{}); ok && len(v) > 0 {
			cfg.CustomizedMetricSpecification = expandAppautoscalingCustomizedMetricSpecification(v)
		}

		if v, ok := ttspCfg["predefined_metric_specification"].([]interface{}); ok && len(v) > 0 {
			cfg.PredefinedMetricSpecification = expandAppautoscalingPredefinedMetricSpecification(v)
		}

		params.TargetTrackingScalingPolicyConfiguration = cfg
	}

	return params, nil
}

func getAwsAppautoscalingPolicy(d *schema.ResourceData, meta interface{}) (*applicationautoscaling.ScalingPolicy, error) {
	conn := meta.(*AWSClient).appautoscalingconn

	params := applicationautoscaling.DescribeScalingPoliciesInput{
		PolicyNames:       []*string{aws.String(d.Get("name").(string))},
		ResourceId:        aws.String(d.Get("resource_id").(string)),
		ScalableDimension: aws.String(d.Get("scalable_dimension").(string)),
		ServiceNamespace:  aws.String(d.Get("service_namespace").(string)),
	}

	log.Printf("[DEBUG] Application AutoScaling Policy Describe Params: %#v", params)
	resp, err := conn.DescribeScalingPolicies(&params)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving scaling policies: %s", err)
	}
	if len(resp.ScalingPolicies) == 0 {
		return nil, nil
	}

	return resp.ScalingPolicies[0], nil
}

func expandStepScalingPolicyConfiguration(cfg []interface{}) *applicationautoscaling.StepScalingPolicyConfiguration {
	if len(cfg) < 1 {
		return nil
	}

	out := &applicationautoscaling.StepScalingPolicyConfiguration{}

	m := cfg[0].(map[string]interface{})
	if v, ok := m["adjustment_type"]; ok {
		out.AdjustmentType = aws.String(v.(string))
	}
	if v, ok := m["cooldown"]; ok {
		out.Cooldown = aws.Int64(int64(v.(int)))
	}
	if v, ok := m["metric_aggregation_type"]; ok {
		out.MetricAggregationType = aws.String(v.(string))
	}
	if v, ok := m["min_adjustment_magnitude"].(int); ok && v > 0 {
		out.MinAdjustmentMagnitude = aws.Int64(int64(v))
	}
	if v, ok := m["step_adjustment"].(*schema.Set); ok && v.Len() > 0 {
		out.StepAdjustments, _ = expandAppautoscalingStepAdjustments(v.List())
	}

	return out
}

func flattenStepScalingPolicyConfiguration(cfg *applicationautoscaling.StepScalingPolicyConfiguration) []interface{} {
	if cfg == nil {
		return []interface{}{}
	}

	m := make(map[string]interface{}, 0)

	if cfg.AdjustmentType != nil {
		m["adjustment_type"] = *cfg.AdjustmentType
	}
	if cfg.Cooldown != nil {
		m["cooldown"] = *cfg.Cooldown
	}
	if cfg.MetricAggregationType != nil {
		m["metric_aggregation_type"] = *cfg.MetricAggregationType
	}
	if cfg.MinAdjustmentMagnitude != nil {
		m["min_adjustment_magnitude"] = *cfg.MinAdjustmentMagnitude
	}
	if cfg.StepAdjustments != nil {
		m["step_adjustment"] = flattenAppautoscalingStepAdjustments(cfg.StepAdjustments)
	}

	return []interface{}{m}
}

func flattenAppautoscalingStepAdjustments(adjs []*applicationautoscaling.StepAdjustment) []interface{} {
	out := make([]interface{}, len(adjs), len(adjs))

	for i, adj := range adjs {
		m := make(map[string]interface{}, 0)

		m["scaling_adjustment"] = *adj.ScalingAdjustment

		if adj.MetricIntervalLowerBound != nil {
			m["metric_interval_lower_bound"] = *adj.MetricIntervalLowerBound
		}
		if adj.MetricIntervalUpperBound != nil {
			m["metric_interval_upper_bound"] = *adj.MetricIntervalUpperBound
		}

		out[i] = m
	}

	return out
}

func flattenTargetTrackingScalingPolicyConfiguration(cfg *applicationautoscaling.TargetTrackingScalingPolicyConfiguration) []interface{} {
	if cfg == nil {
		return []interface{}{}
	}

	m := make(map[string]interface{}, 0)
	m["target_value"] = *cfg.TargetValue

	if cfg.DisableScaleIn != nil {
		m["disable_scale_in"] = *cfg.DisableScaleIn
	}
	if cfg.ScaleInCooldown != nil {
		m["scale_in_cooldown"] = *cfg.ScaleInCooldown
	}
	if cfg.ScaleOutCooldown != nil {
		m["scale_out_cooldown"] = *cfg.ScaleOutCooldown
	}
	if cfg.CustomizedMetricSpecification != nil {
		m["customized_metric_specification"] = flattenCustomizedMetricSpecification(cfg.CustomizedMetricSpecification)
	}
	if cfg.PredefinedMetricSpecification != nil {
		m["predefined_metric_specification"] = flattenPredefinedMetricSpecification(cfg.PredefinedMetricSpecification)
	}

	return []interface{}{m}
}

func flattenCustomizedMetricSpecification(cfg *applicationautoscaling.CustomizedMetricSpecification) []interface{} {
	if cfg == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"metric_name": *cfg.MetricName,
		"namespace":   *cfg.Namespace,
		"statistic":   *cfg.Statistic,
	}

	if len(cfg.Dimensions) > 0 {
		m["dimensions"] = flattenMetricDimensions(cfg.Dimensions)
	}

	if cfg.Unit != nil {
		m["unit"] = *cfg.Unit
	}
	return []interface{}{m}
}

func flattenMetricDimensions(ds []*applicationautoscaling.MetricDimension) []interface{} {
	l := make([]interface{}, len(ds), len(ds))
	for i, d := range ds {
		l[i] = map[string]interface{}{
			"name":  *d.Name,
			"value": *d.Value,
		}
	}
	return l
}

func flattenPredefinedMetricSpecification(cfg *applicationautoscaling.PredefinedMetricSpecification) []interface{} {
	if cfg == nil {
		return []interface{}{}
	}
	m := map[string]interface{}{
		"predefined_metric_type": *cfg.PredefinedMetricType,
	}
	if cfg.ResourceLabel != nil {
		m["resource_label"] = *cfg.ResourceLabel
	}
	return []interface{}{m}
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
