package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/glue"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsGlueTrigger() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsGlueTriggerCreate,
		Read:   resourceAwsGlueTriggerRead,
		Update: resourceAwsGlueTriggerUpdate,
		Delete: resourceAwsGlueTriggerDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"actions": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"arguments": {
							Type:     schema.TypeMap,
							Optional: true,
						},
						"job_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"timeout": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"predicate": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"conditions": {
							Type:     schema.TypeList,
							Required: true,
							MinItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"job_name": {
										Type:     schema.TypeString,
										Required: true,
									},
									"logical_operator": {
										Type:     schema.TypeString,
										Optional: true,
										Default:  glue.LogicalOperatorEquals,
										ValidateFunc: validation.StringInSlice([]string{
											glue.LogicalOperatorEquals,
										}, false),
									},
									"state": {
										Type:     schema.TypeString,
										Required: true,
										ValidateFunc: validation.StringInSlice([]string{
											glue.JobRunStateFailed,
											glue.JobRunStateStopped,
											glue.JobRunStateSucceeded,
											glue.JobRunStateTimeout,
										}, false),
									},
								},
							},
						},
						"logical": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  glue.LogicalAnd,
							ValidateFunc: validation.StringInSlice([]string{
								glue.LogicalAnd,
								glue.LogicalAny,
							}, false),
						},
					},
				},
			},
			"schedule": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					glue.TriggerTypeConditional,
					glue.TriggerTypeOnDemand,
					glue.TriggerTypeScheduled,
				}, false),
			},
		},
	}
}

func resourceAwsGlueTriggerCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).glueconn
	name := d.Get("name").(string)
	triggerType := d.Get("type").(string)

	input := &glue.CreateTriggerInput{
		Actions: expandGlueActions(d.Get("actions").([]interface{})),
		Name:    aws.String(name),
		Type:    aws.String(triggerType),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("predicate"); ok {
		input.Predicate = expandGluePredicate(v.([]interface{}))
	}

	if v, ok := d.GetOk("schedule"); ok {
		input.Schedule = aws.String(v.(string))
	}

	if d.Get("enabled").(bool) && triggerType != glue.TriggerTypeOnDemand {
		input.StartOnCreation = aws.Bool(true)
	}

	log.Printf("[DEBUG] Creating Glue Trigger: %s", input)
	_, err := conn.CreateTrigger(input)
	if err != nil {
		return fmt.Errorf("error creating Glue Trigger (%s): %s", name, err)
	}

	d.SetId(name)

	log.Printf("[DEBUG] Waiting for Glue Trigger (%s) to create", d.Id())
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			glue.TriggerStateActivating,
			glue.TriggerStateCreating,
		},
		Target: []string{
			glue.TriggerStateActivated,
			glue.TriggerStateCreated,
		},
		Refresh: resourceAwsGlueTriggerRefreshFunc(conn, d.Id()),
		Timeout: d.Timeout(schema.TimeoutCreate),
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("error waiting for Glue Trigger (%s) to create", d.Id())
	}

	return resourceAwsGlueTriggerRead(d, meta)
}

func resourceAwsGlueTriggerRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).glueconn

	input := &glue.GetTriggerInput{
		Name: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Reading Glue Trigger: %s", input)
	output, err := conn.GetTrigger(input)
	if err != nil {
		if isAWSErr(err, glue.ErrCodeEntityNotFoundException, "") {
			log.Printf("[WARN] Glue Trigger (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading Glue Trigger (%s): %s", d.Id(), err)
	}

	trigger := output.Trigger
	if trigger == nil {
		log.Printf("[WARN] Glue Trigger (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err := d.Set("actions", flattenGlueActions(trigger.Actions)); err != nil {
		return fmt.Errorf("error setting actions: %s", err)
	}

	d.Set("description", trigger.Description)

	var enabled bool
	state := aws.StringValue(trigger.State)

	if aws.StringValue(trigger.Type) == glue.TriggerTypeOnDemand {
		enabled = (state == glue.TriggerStateCreated || state == glue.TriggerStateCreating)
	} else {
		enabled = (state == glue.TriggerStateActivated || state == glue.TriggerStateActivating)
	}
	d.Set("enabled", enabled)

	if err := d.Set("predicate", flattenGluePredicate(trigger.Predicate)); err != nil {
		return fmt.Errorf("error setting predicate: %s", err)
	}

	d.Set("name", trigger.Name)
	d.Set("schedule", trigger.Schedule)
	d.Set("type", trigger.Type)

	return nil
}

func resourceAwsGlueTriggerUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).glueconn

	if d.HasChange("actions") ||
		d.HasChange("description") ||
		d.HasChange("predicate") ||
		d.HasChange("schedule") {
		triggerUpdate := &glue.TriggerUpdate{
			Actions: expandGlueActions(d.Get("actions").([]interface{})),
		}

		if v, ok := d.GetOk("description"); ok {
			triggerUpdate.Description = aws.String(v.(string))
		}

		if v, ok := d.GetOk("predicate"); ok {
			triggerUpdate.Predicate = expandGluePredicate(v.([]interface{}))
		}

		if v, ok := d.GetOk("schedule"); ok {
			triggerUpdate.Schedule = aws.String(v.(string))
		}
		input := &glue.UpdateTriggerInput{
			Name:          aws.String(d.Id()),
			TriggerUpdate: triggerUpdate,
		}

		log.Printf("[DEBUG] Updating Glue Trigger: %s", input)
		_, err := conn.UpdateTrigger(input)
		if err != nil {
			return fmt.Errorf("error updating Glue Trigger (%s): %s", d.Id(), err)
		}
	}

	if d.HasChange("enabled") {
		if d.Get("enabled").(bool) {
			input := &glue.StartTriggerInput{
				Name: aws.String(d.Id()),
			}

			log.Printf("[DEBUG] Starting Glue Trigger: %s", input)
			_, err := conn.StartTrigger(input)
			if err != nil {
				return fmt.Errorf("error starting Glue Trigger (%s): %s", d.Id(), err)
			}
		} else {
			input := &glue.StopTriggerInput{
				Name: aws.String(d.Id()),
			}

			log.Printf("[DEBUG] Stopping Glue Trigger: %s", input)
			_, err := conn.StopTrigger(input)
			if err != nil {
				return fmt.Errorf("error stopping Glue Trigger (%s): %s", d.Id(), err)
			}
		}
	}

	return nil
}

func resourceAwsGlueTriggerDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).glueconn

	log.Printf("[DEBUG] Deleting Glue Trigger: %s", d.Id())
	err := deleteGlueTrigger(conn, d.Id())
	if err != nil {
		return fmt.Errorf("error deleting Glue Trigger (%s): %s", d.Id(), err)
	}

	log.Printf("[DEBUG] Waiting for Glue Trigger (%s) to delete", d.Id())
	stateConf := &resource.StateChangeConf{
		Pending: []string{glue.TriggerStateDeleting},
		Target:  []string{""},
		Refresh: resourceAwsGlueTriggerRefreshFunc(conn, d.Id()),
		Timeout: d.Timeout(schema.TimeoutDelete),
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		if isAWSErr(err, glue.ErrCodeEntityNotFoundException, "") {
			return nil
		}
		return fmt.Errorf("error waiting for Glue Trigger (%s) to delete", d.Id())
	}

	return nil
}

func resourceAwsGlueTriggerRefreshFunc(conn *glue.Glue, name string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := conn.GetTrigger(&glue.GetTriggerInput{
			Name: aws.String(name),
		})
		if err != nil {
			return output, "", err
		}

		if output.Trigger == nil {
			return output, "", nil
		}

		return output, aws.StringValue(output.Trigger.State), nil
	}
}

func deleteGlueTrigger(conn *glue.Glue, Name string) error {
	input := &glue.DeleteTriggerInput{
		Name: aws.String(Name),
	}

	_, err := conn.DeleteTrigger(input)
	if err != nil {
		if isAWSErr(err, glue.ErrCodeEntityNotFoundException, "") {
			return nil
		}
		return err
	}

	return nil
}

func expandGlueActions(l []interface{}) []*glue.Action {
	actions := []*glue.Action{}

	for _, mRaw := range l {
		m := mRaw.(map[string]interface{})

		action := &glue.Action{
			JobName: aws.String(m["job_name"].(string)),
		}

		argumentsMap := make(map[string]string)
		for k, v := range m["arguments"].(map[string]interface{}) {
			argumentsMap[k] = v.(string)
		}
		action.Arguments = aws.StringMap(argumentsMap)

		if v, ok := m["timeout"]; ok && v.(int) > 0 {
			action.Timeout = aws.Int64(int64(v.(int)))
		}

		actions = append(actions, action)
	}

	return actions
}

func expandGlueConditions(l []interface{}) []*glue.Condition {
	conditions := []*glue.Condition{}

	for _, mRaw := range l {
		m := mRaw.(map[string]interface{})

		condition := &glue.Condition{
			JobName:         aws.String(m["job_name"].(string)),
			LogicalOperator: aws.String(m["logical_operator"].(string)),
			State:           aws.String(m["state"].(string)),
		}

		conditions = append(conditions, condition)
	}

	return conditions
}

func expandGluePredicate(l []interface{}) *glue.Predicate {
	m := l[0].(map[string]interface{})

	predicate := &glue.Predicate{
		Conditions: expandGlueConditions(m["conditions"].([]interface{})),
	}

	if v, ok := m["logical"]; ok && v.(string) != "" {
		predicate.Logical = aws.String(v.(string))
	}

	return predicate
}

func flattenGlueActions(actions []*glue.Action) []interface{} {
	l := []interface{}{}

	for _, action := range actions {
		m := map[string]interface{}{
			"arguments": aws.StringValueMap(action.Arguments),
			"job_name":  aws.StringValue(action.JobName),
			"timeout":   int(aws.Int64Value(action.Timeout)),
		}
		l = append(l, m)
	}

	return l
}

func flattenGlueConditions(conditions []*glue.Condition) []interface{} {
	l := []interface{}{}

	for _, condition := range conditions {
		m := map[string]interface{}{
			"job_name":         aws.StringValue(condition.JobName),
			"logical_operator": aws.StringValue(condition.LogicalOperator),
			"state":            aws.StringValue(condition.State),
		}
		l = append(l, m)
	}

	return l
}

func flattenGluePredicate(predicate *glue.Predicate) []map[string]interface{} {
	if predicate == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"conditions": flattenGlueConditions(predicate.Conditions),
		"logical":    aws.StringValue(predicate.Logical),
	}

	return []map[string]interface{}{m}
}
