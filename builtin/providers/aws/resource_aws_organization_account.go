package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"log"
	"regexp"
	"time"
)

func resourceAwsOrganizationAccount() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsOrganizationAccountCreate,
		Read:   resourceAwsOrganizationAccountRead,
		Update: resourceAwsOrganizationAccountUpdate,
		Delete: resourceAwsOrganizationAccountDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"joined_method": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"joined_timestamp": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringLenBetween(1, 50),
			},
			"email": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateAwsOrganizationAccountEmail,
			},
			"iam_user_access_to_billing": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"ALLOW", "DENY"}, true),
			},
			"role_name": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateAwsOrganizationAccountRoleName,
			},
		},
	}
}

func resourceAwsOrganizationAccountCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).orgsconn

	// Create the account
	createOpts := &organizations.CreateAccountInput{
		AccountName: aws.String(d.Get("name").(string)),
		Email:       aws.String(d.Get("email").(string)),
	}
	if role, ok := d.GetOk("role_name"); ok {
		createOpts.RoleName = aws.String(role.(string))
	}

	if iam_user, ok := d.GetOk("iam_user_access_to_billing"); ok {
		createOpts.IamUserAccessToBilling = aws.String(iam_user.(string))
	}

	log.Printf("[DEBUG] Account create config: %#v", createOpts)

	var err error
	var resp *organizations.CreateAccountOutput
	err = resource.Retry(4*time.Minute, func() *resource.RetryError {
		resp, err = conn.CreateAccount(createOpts)

		if err != nil {
			ec2err, ok := err.(awserr.Error)
			if !ok {
				return resource.NonRetryableError(err)
			}
			if ec2err.Code() == "FinalizingOrganizationException" {
				log.Printf("[DEBUG] Trying to create account again: %q", ec2err.Message())
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("Error creating account: %s", err)
	}
	log.Printf("[DEBUG] Account create response: %#v", resp)

	requestId := *resp.CreateAccountStatus.Id

	// Wait for the account to become available
	log.Printf("[DEBUG] Waiting for account request (%s) to succeed", requestId)

	stateConf := &resource.StateChangeConf{
		Pending: []string{"IN_PROGRESS"},
		Target:  []string{"SUCCEEDED"},
		Refresh: resourceAwsOrganizationAccountStateRefreshFunc(conn, requestId),
		Timeout: 5 * time.Minute,
	}
	stateResp, stateErr := stateConf.WaitForState()
	if stateErr != nil {
		return fmt.Errorf(
			"Error waiting for account request (%s) to become available: %s",
			requestId, stateErr)
	}

	// Store the ID
	accountId := stateResp.(*organizations.CreateAccountStatus).AccountId
	d.SetId(*accountId)

	log.Printf("[INFO] Account ID: %s", d.Id())

	return resourceAwsOrganizationAccountUpdate(d, meta)
}

func resourceAwsOrganizationAccountRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).orgsconn
	describeOpts := &organizations.DescribeAccountInput{
		AccountId: aws.String(d.Id()),
	}
	resp, err := conn.DescribeAccount(describeOpts)
	if err != nil {
		if orgerr, ok := err.(awserr.Error); ok && orgerr.Code() == "AccountNotFoundException" {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("arn", resp.Account.Arn)
	d.Set("joined_method", resp.Account.JoinedMethod)
	d.Set("joined_timestamp", resp.Account.JoinedTimestamp)
	d.Set("name", resp.Account.Name)
	d.Set("status", resp.Account.Status)
	return nil
}

func resourceAwsOrganizationAccountUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceAwsOrganizationAccountRead(d, meta)
}

func resourceAwsOrganizationAccountDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[WARN] Organization member accounts must be deleted from the web console.")
	d.SetId("")
	return nil
}

// resourceAwsOrganizationAccountStateRefreshFunc returns a resource.StateRefreshFunc
// that is used to watch a CreateAccount request
func resourceAwsOrganizationAccountStateRefreshFunc(conn *organizations.Organizations, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		opts := &organizations.DescribeCreateAccountStatusInput{
			CreateAccountRequestId: aws.String(id),
		}
		resp, err := conn.DescribeCreateAccountStatus(opts)
		if err != nil {
			if orgerr, ok := err.(awserr.Error); ok && orgerr.Code() == "CreateAccountStatusNotFoundException" {
				resp = nil
			} else {
				log.Printf("Error on OrganizationAccountStateRefresh: %s", err)
				return nil, "", err
			}
		}

		if resp == nil {
			// Sometimes AWS just has consistency issues and doesn't see
			// our instance yet. Return an empty state.
			return nil, "", nil
		}

		accountStatus := resp.CreateAccountStatus
		if *accountStatus.State == "FAILED" {
			return nil, *accountStatus.State, fmt.Errorf(*accountStatus.FailureReason)
		}
		return accountStatus, *accountStatus.State, nil
	}
}

func validateAwsOrganizationAccountEmail(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q must be a valid email address", value))
	}

	if len(value) < 6 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be less than 6 characters", value))
	}

	if len(value) > 64 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be greater than 64 characters", value))
	}

	return
}

func validateAwsOrganizationAccountRoleName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[\w+=,.@-]{1,64}$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q must consist of uppercase letters, lowercase letters, digits with no spaces, and any of the following characters: =,.@-", value))
	}

	return
}
