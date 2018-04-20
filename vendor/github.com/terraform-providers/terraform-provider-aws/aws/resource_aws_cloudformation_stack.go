package aws

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/structure"
)

func resourceAwsCloudFormationStack() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudFormationStackCreate,
		Read:   resourceAwsCloudFormationStackRead,
		Update: resourceAwsCloudFormationStackUpdate,
		Delete: resourceAwsCloudFormationStackDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"template_body": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateCloudFormationTemplate,
				StateFunc: func(v interface{}) string {
					template, _ := normalizeCloudFormationTemplate(v)
					return template
				},
			},
			"template_url": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"capabilities": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"disable_rollback": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"notification_arns": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"on_failure": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"parameters": {
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
			},
			"outputs": {
				Type:     schema.TypeMap,
				Computed: true,
			},
			"policy_body": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateJsonString,
				StateFunc: func(v interface{}) string {
					json, _ := structure.NormalizeJsonString(v)
					return json
				},
			},
			"policy_url": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"timeout_in_minutes": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"tags": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"iam_role_arn": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceAwsCloudFormationStackCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cfconn

	input := cloudformation.CreateStackInput{
		StackName: aws.String(d.Get("name").(string)),
	}
	if v, ok := d.GetOk("template_body"); ok {
		template, err := normalizeCloudFormationTemplate(v)
		if err != nil {
			return errwrap.Wrapf("template body contains an invalid JSON or YAML: {{err}}", err)
		}
		input.TemplateBody = aws.String(template)
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
		policy, err := structure.NormalizeJsonString(v)
		if err != nil {
			return errwrap.Wrapf("policy body contains an invalid JSON: {{err}}", err)
		}
		input.StackPolicyBody = aws.String(policy)
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
	}
	if v, ok := d.GetOk("iam_role_arn"); ok {
		input.RoleARN = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Creating CloudFormation Stack: %s", input)
	resp, err := conn.CreateStack(&input)
	if err != nil {
		return fmt.Errorf("Creating CloudFormation stack failed: %s", err.Error())
	}

	d.SetId(*resp.StackId)
	var lastStatus string

	wait := resource.StateChangeConf{
		Pending: []string{
			"CREATE_IN_PROGRESS",
			"DELETE_IN_PROGRESS",
			"ROLLBACK_IN_PROGRESS",
		},
		Target: []string{
			"CREATE_COMPLETE",
			"CREATE_FAILED",
			"DELETE_COMPLETE",
			"DELETE_FAILED",
			"ROLLBACK_COMPLETE",
			"ROLLBACK_FAILED",
		},
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 1 * time.Second,
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.DescribeStacks(&cloudformation.DescribeStacksInput{
				StackName: aws.String(d.Id()),
			})
			if err != nil {
				log.Printf("[ERROR] Failed to describe stacks: %s", err)
				return nil, "", err
			}
			if len(resp.Stacks) == 0 {
				// This shouldn't happen unless CloudFormation is inconsistent
				// See https://github.com/hashicorp/terraform/issues/5487
				log.Printf("[WARN] CloudFormation stack %q not found.\nresponse: %q",
					d.Id(), resp)
				return resp, "", fmt.Errorf(
					"CloudFormation stack %q vanished unexpectedly during creation.\n"+
						"Unless you knowingly manually deleted the stack "+
						"please report this as bug at https://github.com/hashicorp/terraform/issues\n"+
						"along with the config & Terraform version & the details below:\n"+
						"Full API response: %s\n",
					d.Id(), resp)
			}

			status := *resp.Stacks[0].StackStatus
			lastStatus = status
			log.Printf("[DEBUG] Current CloudFormation stack status: %q", status)

			return resp, status, err
		},
	}

	_, err = wait.WaitForState()
	if err != nil {
		return err
	}

	if lastStatus == "ROLLBACK_COMPLETE" || lastStatus == "ROLLBACK_FAILED" {
		reasons, err := getCloudFormationRollbackReasons(d.Id(), nil, conn)
		if err != nil {
			return fmt.Errorf("Failed getting rollback reasons: %q", err.Error())
		}

		return fmt.Errorf("%s: %q", lastStatus, reasons)
	}
	if lastStatus == "DELETE_COMPLETE" || lastStatus == "DELETE_FAILED" {
		reasons, err := getCloudFormationDeletionReasons(d.Id(), conn)
		if err != nil {
			return fmt.Errorf("Failed getting deletion reasons: %q", err.Error())
		}

		d.SetId("")
		return fmt.Errorf("%s: %q", lastStatus, reasons)
	}
	if lastStatus == "CREATE_FAILED" {
		reasons, err := getCloudFormationFailures(d.Id(), conn)
		if err != nil {
			return fmt.Errorf("Failed getting failure reasons: %q", err.Error())
		}
		return fmt.Errorf("%s: %q", lastStatus, reasons)
	}

	log.Printf("[INFO] CloudFormation Stack %q created", d.Id())

	return resourceAwsCloudFormationStackRead(d, meta)
}

func resourceAwsCloudFormationStackRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cfconn

	input := &cloudformation.DescribeStacksInput{
		StackName: aws.String(d.Id()),
	}
	resp, err := conn.DescribeStacks(input)
	if err != nil {
		awsErr, ok := err.(awserr.Error)
		// ValidationError: Stack with id % does not exist
		if ok && awsErr.Code() == "ValidationError" {
			log.Printf("[WARN] Removing CloudFormation stack %s as it's already gone", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	stacks := resp.Stacks
	if len(stacks) < 1 {
		log.Printf("[WARN] Removing CloudFormation stack %s as it's already gone", d.Id())
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
		StackName: aws.String(d.Id()),
	}
	out, err := conn.GetTemplate(&tInput)
	if err != nil {
		return err
	}

	template, err := normalizeCloudFormationTemplate(*out.TemplateBody)
	if err != nil {
		return errwrap.Wrapf("template body contains an invalid JSON or YAML: {{err}}", err)
	}
	d.Set("template_body", template)

	stack := stacks[0]
	log.Printf("[DEBUG] Received CloudFormation stack: %s", stack)

	d.Set("name", stack.StackName)
	d.Set("iam_role_arn", stack.RoleARN)

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
		StackName: aws.String(d.Id()),
	}

	// Either TemplateBody, TemplateURL or UsePreviousTemplate are required
	if v, ok := d.GetOk("template_url"); ok {
		input.TemplateURL = aws.String(v.(string))
	}
	if v, ok := d.GetOk("template_body"); ok && input.TemplateURL == nil {
		template, err := normalizeCloudFormationTemplate(v)
		if err != nil {
			return errwrap.Wrapf("template body contains an invalid JSON or YAML: {{err}}", err)
		}
		input.TemplateBody = aws.String(template)
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

	if v, ok := d.GetOk("tags"); ok {
		input.Tags = expandCloudFormationTags(v.(map[string]interface{}))
	}

	if d.HasChange("policy_body") {
		policy, err := structure.NormalizeJsonString(d.Get("policy_body"))
		if err != nil {
			return errwrap.Wrapf("policy body contains an invalid JSON: {{err}}", err)
		}
		input.StackPolicyBody = aws.String(policy)
	}
	if d.HasChange("policy_url") {
		input.StackPolicyURL = aws.String(d.Get("policy_url").(string))
	}

	if d.HasChange("iam_role_arn") {
		input.RoleARN = aws.String(d.Get("iam_role_arn").(string))
	}

	log.Printf("[DEBUG] Updating CloudFormation stack: %s", input)
	_, err := conn.UpdateStack(input)
	if err != nil {
		awsErr, ok := err.(awserr.Error)
		// ValidationError: No updates are to be performed.
		if !ok ||
			awsErr.Code() != "ValidationError" ||
			awsErr.Message() != "No updates are to be performed." {
			return err
		}

		log.Printf("[DEBUG] Current CloudFormation stack has no updates")
	}

	lastUpdatedTime, err := getLastCfEventTimestamp(d.Id(), conn)
	if err != nil {
		return err
	}

	var lastStatus string
	var stackId string
	wait := resource.StateChangeConf{
		Pending: []string{
			"UPDATE_COMPLETE_CLEANUP_IN_PROGRESS",
			"UPDATE_IN_PROGRESS",
			"UPDATE_ROLLBACK_IN_PROGRESS",
			"UPDATE_ROLLBACK_COMPLETE_CLEANUP_IN_PROGRESS",
		},
		Target: []string{
			"CREATE_COMPLETE", // If no stack update was performed
			"UPDATE_COMPLETE",
			"UPDATE_ROLLBACK_COMPLETE",
			"UPDATE_ROLLBACK_FAILED",
		},
		Timeout:    d.Timeout(schema.TimeoutUpdate),
		MinTimeout: 5 * time.Second,
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.DescribeStacks(&cloudformation.DescribeStacksInput{
				StackName: aws.String(d.Id()),
			})
			if err != nil {
				log.Printf("[ERROR] Failed to describe stacks: %s", err)
				return nil, "", err
			}

			stackId = aws.StringValue(resp.Stacks[0].StackId)

			status := *resp.Stacks[0].StackStatus
			lastStatus = status
			log.Printf("[DEBUG] Current CloudFormation stack status: %q", status)

			return resp, status, err
		},
	}

	_, err = wait.WaitForState()
	if err != nil {
		return err
	}

	if lastStatus == "UPDATE_ROLLBACK_COMPLETE" || lastStatus == "UPDATE_ROLLBACK_FAILED" {
		reasons, err := getCloudFormationRollbackReasons(stackId, lastUpdatedTime, conn)
		if err != nil {
			return fmt.Errorf("Failed getting details about rollback: %q", err.Error())
		}

		return fmt.Errorf("%s: %q", lastStatus, reasons)
	}

	log.Printf("[DEBUG] CloudFormation stack %q has been updated", stackId)

	return resourceAwsCloudFormationStackRead(d, meta)
}

func resourceAwsCloudFormationStackDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cfconn

	input := &cloudformation.DeleteStackInput{
		StackName: aws.String(d.Id()),
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
	var lastStatus string
	wait := resource.StateChangeConf{
		Pending: []string{
			"DELETE_IN_PROGRESS",
			"ROLLBACK_IN_PROGRESS",
		},
		Target: []string{
			"DELETE_COMPLETE",
			"DELETE_FAILED",
		},
		Timeout:    d.Timeout(schema.TimeoutDelete),
		MinTimeout: 5 * time.Second,
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.DescribeStacks(&cloudformation.DescribeStacksInput{
				StackName: aws.String(d.Id()),
			})
			if err != nil {
				awsErr, ok := err.(awserr.Error)
				if !ok {
					return nil, "", err
				}

				log.Printf("[DEBUG] Error when deleting CloudFormation stack: %s: %s",
					awsErr.Code(), awsErr.Message())

				// ValidationError: Stack with id % does not exist
				if awsErr.Code() == "ValidationError" {
					return resp, "DELETE_COMPLETE", nil
				}
				return nil, "", err
			}

			if len(resp.Stacks) == 0 {
				log.Printf("[DEBUG] CloudFormation stack %q is already gone", d.Id())
				return resp, "DELETE_COMPLETE", nil
			}

			status := *resp.Stacks[0].StackStatus
			lastStatus = status
			log.Printf("[DEBUG] Current CloudFormation stack status: %q", status)

			return resp, status, err
		},
	}

	_, err = wait.WaitForState()
	if err != nil {
		return err
	}

	if lastStatus == "DELETE_FAILED" {
		reasons, err := getCloudFormationFailures(d.Id(), conn)
		if err != nil {
			return fmt.Errorf("Failed getting reasons of failure: %q", err.Error())
		}

		return fmt.Errorf("%s: %q", lastStatus, reasons)
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

func getCloudFormationRollbackReasons(stackId string, afterTime *time.Time, conn *cloudformation.CloudFormation) ([]string, error) {
	var failures []string

	err := conn.DescribeStackEventsPages(&cloudformation.DescribeStackEventsInput{
		StackName: aws.String(stackId),
	}, func(page *cloudformation.DescribeStackEventsOutput, lastPage bool) bool {
		for _, e := range page.StackEvents {
			if afterTime != nil && !e.Timestamp.After(*afterTime) {
				continue
			}

			if cfStackEventIsFailure(e) || cfStackEventIsRollback(e) {
				failures = append(failures, *e.ResourceStatusReason)
			}
		}
		return !lastPage
	})

	return failures, err
}

func getCloudFormationDeletionReasons(stackId string, conn *cloudformation.CloudFormation) ([]string, error) {
	var failures []string

	err := conn.DescribeStackEventsPages(&cloudformation.DescribeStackEventsInput{
		StackName: aws.String(stackId),
	}, func(page *cloudformation.DescribeStackEventsOutput, lastPage bool) bool {
		for _, e := range page.StackEvents {
			if cfStackEventIsFailure(e) || cfStackEventIsStackDeletion(e) {
				failures = append(failures, *e.ResourceStatusReason)
			}
		}
		return !lastPage
	})

	return failures, err
}

func getCloudFormationFailures(stackId string, conn *cloudformation.CloudFormation) ([]string, error) {
	var failures []string

	err := conn.DescribeStackEventsPages(&cloudformation.DescribeStackEventsInput{
		StackName: aws.String(stackId),
	}, func(page *cloudformation.DescribeStackEventsOutput, lastPage bool) bool {
		for _, e := range page.StackEvents {
			if cfStackEventIsFailure(e) {
				failures = append(failures, *e.ResourceStatusReason)
			}
		}
		return !lastPage
	})

	return failures, err
}

func cfStackEventIsFailure(event *cloudformation.StackEvent) bool {
	failRe := regexp.MustCompile("_FAILED$")
	return failRe.MatchString(*event.ResourceStatus) && event.ResourceStatusReason != nil
}

func cfStackEventIsRollback(event *cloudformation.StackEvent) bool {
	rollbackRe := regexp.MustCompile("^ROLLBACK_")
	return rollbackRe.MatchString(*event.ResourceStatus) && event.ResourceStatusReason != nil
}

func cfStackEventIsStackDeletion(event *cloudformation.StackEvent) bool {
	return *event.ResourceStatus == "DELETE_IN_PROGRESS" &&
		*event.ResourceType == "AWS::CloudFormation::Stack" &&
		event.ResourceStatusReason != nil
}
