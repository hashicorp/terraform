package aws

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	events "github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsCloudWatchEventPermission() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudWatchEventPermissionCreate,
		Read:   resourceAwsCloudWatchEventPermissionRead,
		Update: resourceAwsCloudWatchEventPermissionUpdate,
		Delete: resourceAwsCloudWatchEventPermissionDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"action": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "events:PutEvents",
				ValidateFunc: validateCloudWatchEventPermissionAction,
			},
			"principal": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateCloudWatchEventPermissionPrincipal,
			},
			"statement_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateCloudWatchEventPermissionStatementID,
			},
		},
	}
}

func resourceAwsCloudWatchEventPermissionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatcheventsconn

	statementID := d.Get("statement_id").(string)

	input := events.PutPermissionInput{
		Action:      aws.String(d.Get("action").(string)),
		Principal:   aws.String(d.Get("principal").(string)),
		StatementId: aws.String(statementID),
	}

	log.Printf("[DEBUG] Creating CloudWatch Events permission: %s", input)
	_, err := conn.PutPermission(&input)
	if err != nil {
		return fmt.Errorf("Creating CloudWatch Events permission failed: %s", err.Error())
	}

	d.SetId(statementID)

	return resourceAwsCloudWatchEventPermissionRead(d, meta)
}

// See also: https://docs.aws.amazon.com/AmazonCloudWatchEvents/latest/APIReference/API_DescribeEventBus.html
func resourceAwsCloudWatchEventPermissionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatcheventsconn
	input := events.DescribeEventBusInput{}
	var policyDoc CloudWatchEventPermissionPolicyDoc
	var policyStatement *CloudWatchEventPermissionPolicyStatement

	// Especially with concurrent PutPermission calls there can be a slight delay
	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		log.Printf("[DEBUG] Reading CloudWatch Events bus: %s", input)
		debo, err := conn.DescribeEventBus(&input)
		if err != nil {
			return resource.NonRetryableError(fmt.Errorf("Reading CloudWatch Events permission '%s' failed: %s", d.Id(), err.Error()))
		}

		if debo.Policy == nil {
			return resource.RetryableError(fmt.Errorf("CloudWatch Events permission %q not found", d.Id()))
		}

		err = json.Unmarshal([]byte(*debo.Policy), &policyDoc)
		if err != nil {
			return resource.NonRetryableError(fmt.Errorf("Reading CloudWatch Events permission '%s' failed: %s", d.Id(), err.Error()))
		}

		policyStatement, err = findCloudWatchEventPermissionPolicyStatementByID(&policyDoc, d.Id())
		return resource.RetryableError(err)
	})
	if err != nil {
		// Missing statement inside valid policy
		if nfErr, ok := err.(*resource.NotFoundError); ok {
			log.Printf("[WARN] %s", nfErr)
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("action", policyStatement.Action)

	principalString, ok := policyStatement.Principal.(string)
	if ok && (principalString == "*") {
		d.Set("principal", "*")
	} else {
		principalMap := policyStatement.Principal.(map[string]interface{})
		policyARN, err := arn.Parse(principalMap["AWS"].(string))
		if err != nil {
			return fmt.Errorf("Reading CloudWatch Events permission '%s' failed: %s", d.Id(), err.Error())
		}
		d.Set("principal", policyARN.AccountID)
	}
	d.Set("statement_id", policyStatement.Sid)

	return nil
}

func resourceAwsCloudWatchEventPermissionUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatcheventsconn

	input := events.PutPermissionInput{
		Action:      aws.String(d.Get("action").(string)),
		Principal:   aws.String(d.Get("principal").(string)),
		StatementId: aws.String(d.Get("statement_id").(string)),
	}

	log.Printf("[DEBUG] Update CloudWatch Events permission: %s", input)
	_, err := conn.PutPermission(&input)
	if err != nil {
		return fmt.Errorf("Updating CloudWatch Events permission '%s' failed: %s", d.Id(), err.Error())
	}

	return resourceAwsCloudWatchEventPermissionRead(d, meta)
}

func resourceAwsCloudWatchEventPermissionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatcheventsconn
	input := events.RemovePermissionInput{
		StatementId: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Delete CloudWatch Events permission: %s", input)
	_, err := conn.RemovePermission(&input)
	if err != nil {
		return fmt.Errorf("Deleting CloudWatch Events permission '%s' failed: %s", d.Id(), err.Error())
	}
	return nil
}

// https://docs.aws.amazon.com/AmazonCloudWatchEvents/latest/APIReference/API_PutPermission.html#API_PutPermission_RequestParameters
func validateCloudWatchEventPermissionAction(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)
	if (len(value) < 1) || (len(value) > 64) {
		es = append(es, fmt.Errorf("%q must be between 1 and 64 characters", k))
	}

	if !regexp.MustCompile(`^events:[a-zA-Z]+$`).MatchString(value) {
		es = append(es, fmt.Errorf("%q must be: events: followed by one or more alphabetic characters", k))
	}
	return
}

// https://docs.aws.amazon.com/AmazonCloudWatchEvents/latest/APIReference/API_PutPermission.html#API_PutPermission_RequestParameters
func validateCloudWatchEventPermissionPrincipal(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)
	if !regexp.MustCompile(`^(\d{12}|\*)$`).MatchString(value) {
		es = append(es, fmt.Errorf("%q must be * or a 12 digit AWS account ID", k))
	}
	return
}

// https://docs.aws.amazon.com/AmazonCloudWatchEvents/latest/APIReference/API_PutPermission.html#API_PutPermission_RequestParameters
func validateCloudWatchEventPermissionStatementID(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)
	if (len(value) < 1) || (len(value) > 64) {
		es = append(es, fmt.Errorf("%q must be between 1 and 64 characters", k))
	}

	if !regexp.MustCompile(`^[a-zA-Z0-9-_]+$`).MatchString(value) {
		es = append(es, fmt.Errorf("%q must be one or more alphanumeric, hyphen, or underscore characters", k))
	}
	return
}

// CloudWatchEventPermissionPolicyDoc represents the Policy attribute of DescribeEventBus
// See also: https://docs.aws.amazon.com/AmazonCloudWatchEvents/latest/APIReference/API_DescribeEventBus.html
type CloudWatchEventPermissionPolicyDoc struct {
	Version    string
	ID         string                                     `json:"Id,omitempty"`
	Statements []CloudWatchEventPermissionPolicyStatement `json:"Statement"`
}

// CloudWatchEventPermissionPolicyStatement represents the Statement attribute of CloudWatchEventPermissionPolicyDoc
// See also: https://docs.aws.amazon.com/AmazonCloudWatchEvents/latest/APIReference/API_DescribeEventBus.html
type CloudWatchEventPermissionPolicyStatement struct {
	Sid       string
	Effect    string
	Action    string
	Principal interface{} // "*" or {"AWS": "arn:aws:iam::111111111111:root"}
	Resource  string
}

func findCloudWatchEventPermissionPolicyStatementByID(policy *CloudWatchEventPermissionPolicyDoc, id string) (
	*CloudWatchEventPermissionPolicyStatement, error) {

	log.Printf("[DEBUG] Received %d statements in CloudWatch Events permission policy: %s", len(policy.Statements), policy.Statements)
	for _, statement := range policy.Statements {
		if statement.Sid == id {
			return &statement, nil
		}
	}

	return nil, &resource.NotFoundError{
		LastRequest:  id,
		LastResponse: policy,
		Message:      fmt.Sprintf("Failed to find statement %q in CloudWatch Events permission policy:\n%s", id, policy.Statements),
	}
}
