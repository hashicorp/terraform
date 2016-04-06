package aws

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

func resourceAwsCloudFormationStack() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudFormationStackCreate,
		Read:   resourceAwsCloudFormationStackRead,
		Update: resourceAwsCloudFormationStackUpdate,
		Delete: resourceAwsCloudFormationStackDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"template_body": &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				Computed:  true,
				StateFunc: normalizeJson,
			},
			"template_url": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"capabilities": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"disable_rollback": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"notification_arns": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"on_failure": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"parameters": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
			},
			"outputs": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
			},
			"policy_body": &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				Computed:  true,
				StateFunc: normalizeJson,
			},
			"policy_url": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"timeout_in_minutes": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"tags": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsCloudFormationStackCreate(d *schema.ResourceData, meta interface{}) error {
	retryTimeout := int64(30)
	conn := meta.(*AWSClient).cfconn

	input := cloudformation.CreateStackInput{
		StackName: aws.String(d.Get("name").(string)),
	}
	if v, ok := d.GetOk("template_body"); ok {
		input.TemplateBody = aws.String(normalizeJson(v.(string)))
	}
	if v, ok := d.GetOk("template_url"); ok {
		input.TemplateURL = aws.String(v.(string))
	}
	if v, ok := d.GetOk("capabilities"); ok {
		input.Capabilities = expandStringList(v.(*schema.Set).List())
	}
	if v, ok := d.GetOk("disable_rollback"); ok {
		input.DisableRollback = aws.Bool(v.(bool))
	}
	if v, ok := d.GetOk("notification_arns"); ok {
		input.NotificationARNs = expandStringList(v.(*schema.Set).List())
	}
	if v, ok := d.GetOk("on_failure"); ok {
		input.OnFailure = aws.String(v.(string))
	}
	if v, ok := d.GetOk("parameters"); ok {
		input.Parameters = expandCloudFormationParameters(v.(map[string]interface{}))
	}
	if v, ok := d.GetOk("policy_body"); ok {
		input.StackPolicyBody = aws.String(normalizeJson(v.(string)))
	}
	if v, ok := d.GetOk("policy_url"); ok {
		input.StackPolicyURL = aws.String(v.(string))
	}
	if v, ok := d.GetOk("tags"); ok {
		input.Tags = expandCloudFormationTags(v.(map[string]interface{}))
	}
	if v, ok := d.GetOk("timeout_in_minutes"); ok {
		m := int64(v.(int))
		input.TimeoutInMinutes = aws.Int64(m)
		if m > retryTimeout {
			retryTimeout = m + 5
			log.Printf("[DEBUG] CloudFormation timeout: %d", retryTimeout)
		}
	}

	log.Printf("[DEBUG] Creating CloudFormation Stack: %s", input)
	resp, err := conn.CreateStack(&input)
	if err != nil {
		return fmt.Errorf("Creating CloudFormation stack failed: %s", err.Error())
	}

	d.SetId(*resp.StackId)

	wait := resource.StateChangeConf{
		Pending:    []string{"CREATE_IN_PROGRESS", "ROLLBACK_IN_PROGRESS", "ROLLBACK_COMPLETE"},
		Target:     []string{"CREATE_COMPLETE"},
		Timeout:    time.Duration(retryTimeout) * time.Minute,
		MinTimeout: 5 * time.Second,
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.DescribeStacks(&cloudformation.DescribeStacksInput{
				StackName: aws.String(d.Get("name").(string)),
			})
			status := *resp.Stacks[0].StackStatus
			log.Printf("[DEBUG] Current CloudFormation stack status: %q", status)

			if status == "ROLLBACK_COMPLETE" {
				stack := resp.Stacks[0]
				failures, err := getCloudFormationFailures(stack.StackName, *stack.CreationTime, conn)
				if err != nil {
					return resp, "", fmt.Errorf(
						"Failed getting details about rollback: %q", err.Error())
				}

				return resp, "", fmt.Errorf("ROLLBACK_COMPLETE:\n%q", failures)
			}
			return resp, status, err
		},
	}

	_, err = wait.WaitForState()
	if err != nil {
		return err
	}

	log.Printf("[INFO] CloudFormation Stack %q created", d.Get("name").(string))

	return resourceAwsCloudFormationStackRead(d, meta)
}

func resourceAwsCloudFormationStackRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cfconn
	stackName := d.Get("name").(string)

	input := &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	}
	resp, err := conn.DescribeStacks(input)
	if err != nil {
		return err
	}

	stacks := resp.Stacks
	if len(stacks) < 1 {
		log.Printf("[DEBUG] Removing CloudFormation stack %s as it's already gone", d.Id())
		d.SetId("")
		return nil
	}
	for _, s := range stacks {
		if *s.StackId == d.Id() && *s.StackStatus == "DELETE_COMPLETE" {
			log.Printf("[DEBUG] Removing CloudFormation stack %s"+
				" as it has been already deleted", d.Id())
			d.SetId("")
			return nil
		}
	}

	tInput := cloudformation.GetTemplateInput{
		StackName: aws.String(stackName),
	}
	out, err := conn.GetTemplate(&tInput)
	if err != nil {
		return err
	}

	d.Set("template_body", normalizeJson(*out.TemplateBody))

	stack := stacks[0]
	log.Printf("[DEBUG] Received CloudFormation stack: %s", stack)

	d.Set("name", stack.StackName)
	d.Set("arn", stack.StackId)

	if stack.TimeoutInMinutes != nil {
		d.Set("timeout_in_minutes", int(*stack.TimeoutInMinutes))
	}
	if stack.Description != nil {
		d.Set("description", stack.Description)
	}
	if stack.DisableRollback != nil {
		d.Set("disable_rollback", stack.DisableRollback)
	}
	if len(stack.NotificationARNs) > 0 {
		err = d.Set("notification_arns", schema.NewSet(schema.HashString, flattenStringList(stack.NotificationARNs)))
		if err != nil {
			return err
		}
	}

	originalParams := d.Get("parameters").(map[string]interface{})
	err = d.Set("parameters", flattenCloudFormationParameters(stack.Parameters, originalParams))
	if err != nil {
		return err
	}

	err = d.Set("tags", flattenCloudFormationTags(stack.Tags))
	if err != nil {
		return err
	}

	err = d.Set("outputs", flattenCloudFormationOutputs(stack.Outputs))
	if err != nil {
		return err
	}

	if len(stack.Capabilities) > 0 {
		err = d.Set("capabilities", schema.NewSet(schema.HashString, flattenStringList(stack.Capabilities)))
		if err != nil {
			return err
		}
	}

	return nil
}

func resourceAwsCloudFormationStackUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cfconn

	input := &cloudformation.UpdateStackInput{
		StackName: aws.String(d.Get("name").(string)),
	}

	// Either TemplateBody, TemplateURL or UsePreviousTemplate are required
	if v, ok := d.GetOk("template_url"); ok {
		input.TemplateURL = aws.String(v.(string))
	}
	if v, ok := d.GetOk("template_body"); ok && input.TemplateURL == nil {
		input.TemplateBody = aws.String(normalizeJson(v.(string)))
	}

	// Capabilities must be present whether they are changed or not
	if v, ok := d.GetOk("capabilities"); ok {
		input.Capabilities = expandStringList(v.(*schema.Set).List())
	}

	if d.HasChange("notification_arns") {
		input.NotificationARNs = expandStringList(d.Get("notification_arns").(*schema.Set).List())
	}

	// Parameters must be present whether they are changed or not
	if v, ok := d.GetOk("parameters"); ok {
		input.Parameters = expandCloudFormationParameters(v.(map[string]interface{}))
	}

	if d.HasChange("policy_body") {
		input.StackPolicyBody = aws.String(normalizeJson(d.Get("policy_body").(string)))
	}
	if d.HasChange("policy_url") {
		input.StackPolicyURL = aws.String(d.Get("policy_url").(string))
	}

	log.Printf("[DEBUG] Updating CloudFormation stack: %s", input)
	stack, err := conn.UpdateStack(input)
	if err != nil {
		return err
	}

	lastUpdatedTime, err := getLastCfEventTimestamp(d.Get("name").(string), conn)
	if err != nil {
		return err
	}

	wait := resource.StateChangeConf{
		Pending: []string{
			"UPDATE_COMPLETE_CLEANUP_IN_PROGRESS",
			"UPDATE_IN_PROGRESS",
			"UPDATE_ROLLBACK_IN_PROGRESS",
			"UPDATE_ROLLBACK_COMPLETE_CLEANUP_IN_PROGRESS",
			"UPDATE_ROLLBACK_COMPLETE",
		},
		Target:     []string{"UPDATE_COMPLETE"},
		Timeout:    15 * time.Minute,
		MinTimeout: 5 * time.Second,
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.DescribeStacks(&cloudformation.DescribeStacksInput{
				StackName: aws.String(d.Get("name").(string)),
			})
			stack := resp.Stacks[0]
			status := *stack.StackStatus
			log.Printf("[DEBUG] Current CloudFormation stack status: %q", status)

			if status == "UPDATE_ROLLBACK_COMPLETE" {
				failures, err := getCloudFormationFailures(stack.StackName, *lastUpdatedTime, conn)
				if err != nil {
					return resp, "", fmt.Errorf(
						"Failed getting details about rollback: %q", err.Error())
				}

				return resp, "", fmt.Errorf(
					"UPDATE_ROLLBACK_COMPLETE:\n%q", failures)
			}

			return resp, status, err
		},
	}

	_, err = wait.WaitForState()
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] CloudFormation stack %q has been updated", *stack.StackId)

	return resourceAwsCloudFormationStackRead(d, meta)
}

func resourceAwsCloudFormationStackDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cfconn

	input := &cloudformation.DeleteStackInput{
		StackName: aws.String(d.Get("name").(string)),
	}
	log.Printf("[DEBUG] Deleting CloudFormation stack %s", input)
	_, err := conn.DeleteStack(input)
	if err != nil {
		awsErr, ok := err.(awserr.Error)
		if !ok {
			return err
		}

		if awsErr.Code() == "ValidationError" {
			// Ignore stack which has been already deleted
			return nil
		}
		return err
	}

	wait := resource.StateChangeConf{
		Pending:    []string{"DELETE_IN_PROGRESS", "ROLLBACK_IN_PROGRESS"},
		Target:     []string{"DELETE_COMPLETE"},
		Timeout:    30 * time.Minute,
		MinTimeout: 5 * time.Second,
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.DescribeStacks(&cloudformation.DescribeStacksInput{
				StackName: aws.String(d.Get("name").(string)),
			})

			if err != nil {
				awsErr, ok := err.(awserr.Error)
				if !ok {
					return resp, "DELETE_FAILED", err
				}

				log.Printf("[DEBUG] Error when deleting CloudFormation stack: %s: %s",
					awsErr.Code(), awsErr.Message())

				if awsErr.Code() == "ValidationError" {
					return resp, "DELETE_COMPLETE", nil
				}
			}

			if len(resp.Stacks) == 0 {
				log.Printf("[DEBUG] CloudFormation stack %q is already gone", d.Get("name"))
				return resp, "DELETE_COMPLETE", nil
			}

			status := *resp.Stacks[0].StackStatus
			log.Printf("[DEBUG] Current CloudFormation stack status: %q", status)

			return resp, status, err
		},
	}

	_, err = wait.WaitForState()
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] CloudFormation stack %q has been deleted", d.Id())

	d.SetId("")

	return nil
}

// getLastCfEventTimestamp takes the first event in a list
// of events ordered from the newest to the oldest
// and extracts timestamp from it
// LastUpdatedTime only provides last >successful< updated time
func getLastCfEventTimestamp(stackName string, conn *cloudformation.CloudFormation) (
	*time.Time, error) {
	output, err := conn.DescribeStackEvents(&cloudformation.DescribeStackEventsInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		return nil, err
	}

	return output.StackEvents[0].Timestamp, nil
}

// getCloudFormationFailures returns ResourceStatusReason(s)
// of events that should be failures based on regexp match of status
func getCloudFormationFailures(stackName *string, afterTime time.Time,
	conn *cloudformation.CloudFormation) ([]string, error) {
	var failures []string
	// Only catching failures from last 100 events
	// Some extra iteration logic via NextToken could be added
	// but in reality it's nearly impossible to generate >100
	// events by a single stack update
	events, err := conn.DescribeStackEvents(&cloudformation.DescribeStackEventsInput{
		StackName: stackName,
	})

	if err != nil {
		return nil, err
	}

	failRe := regexp.MustCompile("_FAILED$")
	rollbackRe := regexp.MustCompile("^ROLLBACK_")

	for _, e := range events.StackEvents {
		if (failRe.MatchString(*e.ResourceStatus) || rollbackRe.MatchString(*e.ResourceStatus)) &&
			e.Timestamp.After(afterTime) && e.ResourceStatusReason != nil {
			failures = append(failures, *e.ResourceStatusReason)
		}
	}

	return failures, nil
}
