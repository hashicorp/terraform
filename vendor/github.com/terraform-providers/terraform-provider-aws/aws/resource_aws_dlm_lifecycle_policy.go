package aws

import (
	"fmt"
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dlm"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsDlmLifecyclePolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDlmLifecyclePolicyCreate,
		Read:   resourceAwsDlmLifecyclePolicyRead,
		Update: resourceAwsDlmLifecyclePolicyUpdate,
		Delete: resourceAwsDlmLifecyclePolicyDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"description": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^[0-9A-Za-z _-]+$"), "see https://docs.aws.amazon.com/cli/latest/reference/dlm/create-lifecycle-policy.html"),
				//	TODO: https://docs.aws.amazon.com/dlm/latest/APIReference/API_LifecyclePolicy.html#dlm-Type-LifecyclePolicy-Description says it has max length of 500 but doesn't mention the regex but SDK and CLI docs only mention the regex and not max length. Check this
			},
			"execution_role_arn": {
				// TODO: Make this not required and if it's not provided then use the default service role, creating it if necessary
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateArn,
			},
			"policy_details": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"resource_types": {
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"schedule": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"copy_tags": {
										Type:     schema.TypeBool,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"create_rule": {
										Type:     schema.TypeList,
										Required: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"interval": {
													Type:     schema.TypeInt,
													Required: true,
													ValidateFunc: validateIntegerInSlice([]int{
														12,
														24,
													}),
												},
												"interval_unit": {
													Type:     schema.TypeString,
													Optional: true,
													Default:  dlm.IntervalUnitValuesHours,
													ValidateFunc: validation.StringInSlice([]string{
														dlm.IntervalUnitValuesHours,
													}, false),
												},
												"times": {
													Type:     schema.TypeList,
													Optional: true,
													Computed: true,
													MaxItems: 1,
													Elem: &schema.Schema{
														Type:         schema.TypeString,
														ValidateFunc: validation.StringMatch(regexp.MustCompile("^([0-9]|0[0-9]|1[0-9]|2[0-3]):[0-5][0-9]$"), "see https://docs.aws.amazon.com/dlm/latest/APIReference/API_CreateRule.html#dlm-Type-CreateRule-Times"),
													},
												},
											},
										},
									},
									"name": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validation.StringLenBetween(0, 500),
									},
									"retain_rule": {
										Type:     schema.TypeList,
										Required: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"count": {
													Type:         schema.TypeInt,
													Required:     true,
													ValidateFunc: validation.IntBetween(1, 1000),
												},
											},
										},
									},
									"tags_to_add": {
										Type:     schema.TypeMap,
										Optional: true,
									},
								},
							},
						},
						"target_tags": {
							Type:     schema.TypeMap,
							Required: true,
						},
					},
				},
			},
			"state": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  dlm.SettablePolicyStateValuesEnabled,
				ValidateFunc: validation.StringInSlice([]string{
					dlm.SettablePolicyStateValuesDisabled,
					dlm.SettablePolicyStateValuesEnabled,
				}, false),
			},
		},
	}
}

func resourceAwsDlmLifecyclePolicyCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dlmconn

	input := dlm.CreateLifecyclePolicyInput{
		Description:      aws.String(d.Get("description").(string)),
		ExecutionRoleArn: aws.String(d.Get("execution_role_arn").(string)),
		PolicyDetails:    expandDlmPolicyDetails(d.Get("policy_details").([]interface{})),
		State:            aws.String(d.Get("state").(string)),
	}

	log.Printf("[INFO] Creating DLM lifecycle policy: %s", input)
	out, err := conn.CreateLifecyclePolicy(&input)
	if err != nil {
		return fmt.Errorf("error creating DLM Lifecycle Policy: %s", err)
	}

	d.SetId(*out.PolicyId)

	return resourceAwsDlmLifecyclePolicyRead(d, meta)
}

func resourceAwsDlmLifecyclePolicyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dlmconn

	log.Printf("[INFO] Reading DLM lifecycle policy: %s", d.Id())
	out, err := conn.GetLifecyclePolicy(&dlm.GetLifecyclePolicyInput{
		PolicyId: aws.String(d.Id()),
	})

	if isAWSErr(err, dlm.ErrCodeResourceNotFoundException, "") {
		log.Printf("[WARN] DLM Lifecycle Policy (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading DLM Lifecycle Policy (%s): %s", d.Id(), err)
	}

	d.Set("description", out.Policy.Description)
	d.Set("execution_role_arn", out.Policy.ExecutionRoleArn)
	d.Set("state", out.Policy.State)
	if err := d.Set("policy_details", flattenDlmPolicyDetails(out.Policy.PolicyDetails)); err != nil {
		return fmt.Errorf("error setting policy details %s", err)
	}

	return nil
}

func resourceAwsDlmLifecyclePolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dlmconn

	input := dlm.UpdateLifecyclePolicyInput{
		PolicyId: aws.String(d.Id()),
	}

	if d.HasChange("description") {
		input.Description = aws.String(d.Get("description").(string))
	}
	if d.HasChange("execution_role_arn") {
		input.ExecutionRoleArn = aws.String(d.Get("execution_role_arn").(string))
	}
	if d.HasChange("state") {
		input.State = aws.String(d.Get("state").(string))
	}
	if d.HasChange("policy_details") {
		input.PolicyDetails = expandDlmPolicyDetails(d.Get("policy_details").([]interface{}))
	}

	log.Printf("[INFO] Updating lifecycle policy %s", d.Id())
	_, err := conn.UpdateLifecyclePolicy(&input)
	if err != nil {
		return fmt.Errorf("error updating DLM Lifecycle Policy (%s): %s", d.Id(), err)
	}

	return resourceAwsDlmLifecyclePolicyRead(d, meta)
}

func resourceAwsDlmLifecyclePolicyDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dlmconn

	log.Printf("[INFO] Deleting DLM lifecycle policy: %s", d.Id())
	_, err := conn.DeleteLifecyclePolicy(&dlm.DeleteLifecyclePolicyInput{
		PolicyId: aws.String(d.Id()),
	})
	if err != nil {
		return fmt.Errorf("error deleting DLM Lifecycle Policy (%s): %s", d.Id(), err)
	}

	return nil
}

func expandDlmPolicyDetails(cfg []interface{}) *dlm.PolicyDetails {
	if len(cfg) == 0 || cfg[0] == nil {
		return nil
	}

	policyDetails := &dlm.PolicyDetails{}
	m := cfg[0].(map[string]interface{})
	if v, ok := m["resource_types"]; ok {
		policyDetails.ResourceTypes = expandStringList(v.([]interface{}))
	}
	if v, ok := m["schedule"]; ok {
		policyDetails.Schedules = expandDlmSchedules(v.([]interface{}))
	}
	if v, ok := m["target_tags"]; ok {
		policyDetails.TargetTags = expandDlmTags(v.(map[string]interface{}))
	}

	return policyDetails
}

func flattenDlmPolicyDetails(policyDetails *dlm.PolicyDetails) []map[string]interface{} {
	result := make(map[string]interface{}, 0)
	result["resource_types"] = flattenStringList(policyDetails.ResourceTypes)
	result["schedule"] = flattenDlmSchedules(policyDetails.Schedules)
	result["target_tags"] = flattenDlmTags(policyDetails.TargetTags)

	return []map[string]interface{}{result}
}

func expandDlmSchedules(cfg []interface{}) []*dlm.Schedule {
	schedules := make([]*dlm.Schedule, len(cfg))
	for i, c := range cfg {
		schedule := &dlm.Schedule{}
		m := c.(map[string]interface{})
		if v, ok := m["copy_tags"]; ok {
			schedule.CopyTags = aws.Bool(v.(bool))
		}
		if v, ok := m["create_rule"]; ok {
			schedule.CreateRule = expandDlmCreateRule(v.([]interface{}))
		}
		if v, ok := m["name"]; ok {
			schedule.Name = aws.String(v.(string))
		}
		if v, ok := m["retain_rule"]; ok {
			schedule.RetainRule = expandDlmRetainRule(v.([]interface{}))
		}
		if v, ok := m["tags_to_add"]; ok {
			schedule.TagsToAdd = expandDlmTags(v.(map[string]interface{}))
		}
		schedules[i] = schedule
	}

	return schedules
}

func flattenDlmSchedules(schedules []*dlm.Schedule) []map[string]interface{} {
	result := make([]map[string]interface{}, len(schedules))
	for i, s := range schedules {
		m := make(map[string]interface{})
		m["copy_tags"] = aws.BoolValue(s.CopyTags)
		m["create_rule"] = flattenDlmCreateRule(s.CreateRule)
		m["name"] = aws.StringValue(s.Name)
		m["retain_rule"] = flattenDlmRetainRule(s.RetainRule)
		m["tags_to_add"] = flattenDlmTags(s.TagsToAdd)
		result[i] = m
	}

	return result
}

func expandDlmCreateRule(cfg []interface{}) *dlm.CreateRule {
	if len(cfg) == 0 || cfg[0] == nil {
		return nil
	}
	c := cfg[0].(map[string]interface{})
	createRule := &dlm.CreateRule{
		Interval:     aws.Int64(int64(c["interval"].(int))),
		IntervalUnit: aws.String(c["interval_unit"].(string)),
	}
	if v, ok := c["times"]; ok {
		createRule.Times = expandStringList(v.([]interface{}))
	}

	return createRule
}

func flattenDlmCreateRule(createRule *dlm.CreateRule) []map[string]interface{} {
	if createRule == nil {
		return []map[string]interface{}{}
	}

	result := make(map[string]interface{})
	result["interval"] = aws.Int64Value(createRule.Interval)
	result["interval_unit"] = aws.StringValue(createRule.IntervalUnit)
	result["times"] = flattenStringList(createRule.Times)

	return []map[string]interface{}{result}
}

func expandDlmRetainRule(cfg []interface{}) *dlm.RetainRule {
	if len(cfg) == 0 || cfg[0] == nil {
		return nil
	}
	m := cfg[0].(map[string]interface{})
	return &dlm.RetainRule{
		Count: aws.Int64(int64(m["count"].(int))),
	}
}

func flattenDlmRetainRule(retainRule *dlm.RetainRule) []map[string]interface{} {
	result := make(map[string]interface{})
	result["count"] = aws.Int64Value(retainRule.Count)

	return []map[string]interface{}{result}
}

func expandDlmTags(m map[string]interface{}) []*dlm.Tag {
	var result []*dlm.Tag
	for k, v := range m {
		result = append(result, &dlm.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		})
	}

	return result
}

func flattenDlmTags(tags []*dlm.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range tags {
		result[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
	}

	return result
}
