package aws

import (
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsAlbListenerRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAlbListenerRuleCreate,
		Read:   resourceAwsAlbListenerRuleRead,
		Update: resourceAwsAlbListenerRuleUpdate,
		Delete: resourceAwsAlbListenerRuleDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"listener_arn": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"priority": {
				Type:         schema.TypeInt,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateAwsAlbListenerRulePriority,
			},
			"action": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"target_group_arn": {
							Type:     schema.TypeString,
							Required: true,
						},
						"type": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateAwsAlbListenerActionType,
						},
					},
				},
			},
			"condition": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"field": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateAwsListenerRuleField,
						},
						"values": {
							Type:     schema.TypeList,
							MaxItems: 1,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsAlbListenerRuleCreate(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn

	params := &elbv2.CreateRuleInput{
		ListenerArn: aws.String(d.Get("listener_arn").(string)),
		Priority:    aws.Int64(int64(d.Get("priority").(int))),
	}

	actions := d.Get("action").([]interface{})
	params.Actions = make([]*elbv2.Action, len(actions))
	for i, action := range actions {
		actionMap := action.(map[string]interface{})
		params.Actions[i] = &elbv2.Action{
			TargetGroupArn: aws.String(actionMap["target_group_arn"].(string)),
			Type:           aws.String(actionMap["type"].(string)),
		}
	}

	conditions := d.Get("condition").([]interface{})
	params.Conditions = make([]*elbv2.RuleCondition, len(conditions))
	for i, condition := range conditions {
		conditionMap := condition.(map[string]interface{})
		values := conditionMap["values"].([]interface{})
		params.Conditions[i] = &elbv2.RuleCondition{
			Field:  aws.String(conditionMap["field"].(string)),
			Values: make([]*string, len(values)),
		}
		for j, value := range values {
			params.Conditions[i].Values[j] = aws.String(value.(string))
		}
	}

	resp, err := elbconn.CreateRule(params)
	if err != nil {
		return errwrap.Wrapf("Error creating ALB Listener Rule: {{err}}", err)
	}

	if len(resp.Rules) == 0 {
		return errors.New("Error creating ALB Listener Rule: no rules returned in response")
	}

	d.SetId(*resp.Rules[0].RuleArn)

	return resourceAwsAlbListenerRuleRead(d, meta)
}

func resourceAwsAlbListenerRuleRead(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn

	resp, err := elbconn.DescribeRules(&elbv2.DescribeRulesInput{
		RuleArns: []*string{aws.String(d.Id())},
	})
	if err != nil {
		if isRuleNotFound(err) {
			log.Printf("[WARN] DescribeRules - removing %s from state", d.Id())
			d.SetId("")
			return nil
		}
		return errwrap.Wrapf(fmt.Sprintf("Error retrieving Rules for listener %s: {{err}}", d.Id()), err)
	}

	if len(resp.Rules) != 1 {
		return fmt.Errorf("Error retrieving Rule %q", d.Id())
	}

	rule := resp.Rules[0]

	d.Set("arn", rule.RuleArn)
	// Rules are evaluated in priority order, from the lowest value to the highest value. The default rule has the lowest priority.
	if *rule.Priority == "default" {
		d.Set("priority", 99999)
	} else {
		if priority, err := strconv.Atoi(*rule.Priority); err != nil {
			return errwrap.Wrapf("Cannot convert rule priority %q to int: {{err}}", err)
		} else {
			d.Set("priority", priority)
		}
	}

	actions := make([]interface{}, len(rule.Actions))
	for i, action := range rule.Actions {
		actionMap := make(map[string]interface{})
		actionMap["target_group_arn"] = *action.TargetGroupArn
		actionMap["type"] = *action.Type
		actions[i] = actionMap
	}
	d.Set("action", actions)

	conditions := make([]interface{}, len(rule.Conditions))
	for i, condition := range rule.Conditions {
		conditionMap := make(map[string]interface{})
		conditionMap["field"] = *condition.Field
		conditionValues := make([]string, len(condition.Values))
		for k, value := range condition.Values {
			conditionValues[k] = *value
		}
		conditionMap["values"] = conditionValues
		conditions[i] = conditionMap
	}
	d.Set("condition", conditions)

	return nil
}

func resourceAwsAlbListenerRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn

	d.Partial(true)

	if d.HasChange("priority") {
		params := &elbv2.SetRulePrioritiesInput{
			RulePriorities: []*elbv2.RulePriorityPair{
				{
					RuleArn:  aws.String(d.Id()),
					Priority: aws.Int64(int64(d.Get("priority").(int))),
				},
			},
		}

		_, err := elbconn.SetRulePriorities(params)
		if err != nil {
			return err
		}

		d.SetPartial("priority")
	}

	requestUpdate := false
	params := &elbv2.ModifyRuleInput{
		RuleArn: aws.String(d.Id()),
	}

	if d.HasChange("action") {
		actions := d.Get("action").([]interface{})
		params.Actions = make([]*elbv2.Action, len(actions))
		for i, action := range actions {
			actionMap := action.(map[string]interface{})
			params.Actions[i] = &elbv2.Action{
				TargetGroupArn: aws.String(actionMap["target_group_arn"].(string)),
				Type:           aws.String(actionMap["type"].(string)),
			}
		}
		requestUpdate = true
		d.SetPartial("action")
	}

	if d.HasChange("condition") {
		conditions := d.Get("condition").([]interface{})
		params.Conditions = make([]*elbv2.RuleCondition, len(conditions))
		for i, condition := range conditions {
			conditionMap := condition.(map[string]interface{})
			values := conditionMap["values"].([]interface{})
			params.Conditions[i] = &elbv2.RuleCondition{
				Field:  aws.String(conditionMap["field"].(string)),
				Values: make([]*string, len(values)),
			}
			for j, value := range values {
				params.Conditions[i].Values[j] = aws.String(value.(string))
			}
		}
		requestUpdate = true
		d.SetPartial("condition")
	}

	if requestUpdate {
		resp, err := elbconn.ModifyRule(params)
		if err != nil {
			return errwrap.Wrapf("Error modifying ALB Listener Rule: {{err}}", err)
		}

		if len(resp.Rules) == 0 {
			return errors.New("Error modifying creating ALB Listener Rule: no rules returned in response")
		}
	}

	d.Partial(false)

	return resourceAwsAlbListenerRuleRead(d, meta)
}

func resourceAwsAlbListenerRuleDelete(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn

	_, err := elbconn.DeleteRule(&elbv2.DeleteRuleInput{
		RuleArn: aws.String(d.Id()),
	})
	if err != nil && !isRuleNotFound(err) {
		return errwrap.Wrapf("Error deleting ALB Listener Rule: {{err}}", err)
	}
	return nil
}

func validateAwsAlbListenerRulePriority(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	if value < 1 || value > 99999 {
		errors = append(errors, fmt.Errorf("%q must be in the range 1-99999", k))
	}
	return
}

func validateAwsListenerRuleField(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) > 64 {
		errors = append(errors, fmt.Errorf("%q must be a maximum of 64 characters", k))
	}
	return
}

func isRuleNotFound(err error) bool {
	elberr, ok := err.(awserr.Error)
	return ok && elberr.Code() == "RuleNotFound"
}
