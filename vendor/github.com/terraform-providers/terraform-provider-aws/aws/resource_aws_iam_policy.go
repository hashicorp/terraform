package aws

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIamPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamPolicyCreate,
		Read:   resourceAwsIamPolicyRead,
		Update: resourceAwsIamPolicyUpdate,
		Delete: resourceAwsIamPolicyDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"description": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
			"path": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "/",
				ForceNew: true,
			},
			"policy": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateFunc:     validateIAMPolicyJson,
				DiffSuppressFunc: suppressEquivalentAwsPolicyDiffs,
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					// https://github.com/boto/botocore/blob/2485f5c/botocore/data/iam/2010-05-08/service-2.json#L8329-L8334
					value := v.(string)
					if len(value) > 128 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 128 characters", k))
					}
					if !regexp.MustCompile("^[\\w+=,.@-]*$").MatchString(value) {
						errors = append(errors, fmt.Errorf(
							"%q must match [\\w+=,.@-]", k))
					}
					return
				},
			},
			"name_prefix": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name"},
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					// https://github.com/boto/botocore/blob/2485f5c/botocore/data/iam/2010-05-08/service-2.json#L8329-L8334
					value := v.(string)
					if len(value) > 96 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 96 characters, name is limited to 128", k))
					}
					if !regexp.MustCompile("^[\\w+=,.@-]*$").MatchString(value) {
						errors = append(errors, fmt.Errorf(
							"%q must match [\\w+=,.@-]", k))
					}
					return
				},
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsIamPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	var name string
	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		name = resource.PrefixedUniqueId(v.(string))
	} else {
		name = resource.UniqueId()
	}

	request := &iam.CreatePolicyInput{
		Description:    aws.String(d.Get("description").(string)),
		Path:           aws.String(d.Get("path").(string)),
		PolicyDocument: aws.String(d.Get("policy").(string)),
		PolicyName:     aws.String(name),
	}

	response, err := iamconn.CreatePolicy(request)
	if err != nil {
		return fmt.Errorf("Error creating IAM policy %s: %s", name, err)
	}

	d.SetId(*response.Policy.Arn)

	return resourceAwsIamPolicyRead(d, meta)
}

func resourceAwsIamPolicyRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	getPolicyRequest := &iam.GetPolicyInput{
		PolicyArn: aws.String(d.Id()),
	}
	log.Printf("[DEBUG] Getting IAM Policy: %s", getPolicyRequest)

	// Handle IAM eventual consistency
	var getPolicyResponse *iam.GetPolicyOutput
	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		var err error
		getPolicyResponse, err = iamconn.GetPolicy(getPolicyRequest)

		if d.IsNewResource() && isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
			return resource.RetryableError(err)
		}

		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})

	if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
		log.Printf("[WARN] IAM Policy (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("Error reading IAM policy %s: %s", d.Id(), err)
	}

	if getPolicyResponse == nil || getPolicyResponse.Policy == nil {
		log.Printf("[WARN] IAM Policy (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("arn", getPolicyResponse.Policy.Arn)
	d.Set("description", getPolicyResponse.Policy.Description)
	d.Set("name", getPolicyResponse.Policy.PolicyName)
	d.Set("path", getPolicyResponse.Policy.Path)

	// Retrieve policy

	getPolicyVersionRequest := &iam.GetPolicyVersionInput{
		PolicyArn: aws.String(d.Id()),
		VersionId: getPolicyResponse.Policy.DefaultVersionId,
	}
	log.Printf("[DEBUG] Getting IAM Policy Version: %s", getPolicyVersionRequest)

	// Handle IAM eventual consistency
	var getPolicyVersionResponse *iam.GetPolicyVersionOutput
	err = resource.Retry(1*time.Minute, func() *resource.RetryError {
		var err error
		getPolicyVersionResponse, err = iamconn.GetPolicyVersion(getPolicyVersionRequest)

		if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
			return resource.RetryableError(err)
		}

		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})

	if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
		log.Printf("[WARN] IAM Policy (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("Error reading IAM policy version %s: %s", d.Id(), err)
	}

	policy := ""
	if getPolicyVersionResponse != nil && getPolicyVersionResponse.PolicyVersion != nil {
		var err error
		policy, err = url.QueryUnescape(aws.StringValue(getPolicyVersionResponse.PolicyVersion.Document))
		if err != nil {
			return fmt.Errorf("error parsing policy: %s", err)
		}
	}

	d.Set("policy", policy)

	return nil
}

func resourceAwsIamPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	if err := iamPolicyPruneVersions(d.Id(), iamconn); err != nil {
		return err
	}

	request := &iam.CreatePolicyVersionInput{
		PolicyArn:      aws.String(d.Id()),
		PolicyDocument: aws.String(d.Get("policy").(string)),
		SetAsDefault:   aws.Bool(true),
	}

	if _, err := iamconn.CreatePolicyVersion(request); err != nil {
		return fmt.Errorf("Error updating IAM policy %s: %s", d.Id(), err)
	}

	return resourceAwsIamPolicyRead(d, meta)
}

func resourceAwsIamPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	if err := iamPolicyDeleteNondefaultVersions(d.Id(), iamconn); err != nil {
		return err
	}

	request := &iam.DeletePolicyInput{
		PolicyArn: aws.String(d.Id()),
	}

	_, err := iamconn.DeletePolicy(request)
	if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("Error deleting IAM policy %s: %#v", d.Id(), err)
	}

	return nil
}

// iamPolicyPruneVersions deletes the oldest versions.
//
// Old versions are deleted until there are 4 or less remaining, which means at
// least one more can be created before hitting the maximum of 5.
//
// The default version is never deleted.

func iamPolicyPruneVersions(arn string, iamconn *iam.IAM) error {
	versions, err := iamPolicyListVersions(arn, iamconn)
	if err != nil {
		return err
	}
	if len(versions) < 5 {
		return nil
	}

	var oldestVersion *iam.PolicyVersion

	for _, version := range versions {
		if *version.IsDefaultVersion {
			continue
		}
		if oldestVersion == nil ||
			version.CreateDate.Before(*oldestVersion.CreateDate) {
			oldestVersion = version
		}
	}

	if err := iamPolicyDeleteVersion(arn, *oldestVersion.VersionId, iamconn); err != nil {
		return err
	}
	return nil
}

func iamPolicyDeleteNondefaultVersions(arn string, iamconn *iam.IAM) error {
	versions, err := iamPolicyListVersions(arn, iamconn)
	if err != nil {
		return err
	}

	for _, version := range versions {
		if *version.IsDefaultVersion {
			continue
		}
		if err := iamPolicyDeleteVersion(arn, *version.VersionId, iamconn); err != nil {
			return err
		}
	}

	return nil
}

func iamPolicyDeleteVersion(arn, versionID string, iamconn *iam.IAM) error {
	request := &iam.DeletePolicyVersionInput{
		PolicyArn: aws.String(arn),
		VersionId: aws.String(versionID),
	}

	_, err := iamconn.DeletePolicyVersion(request)
	if err != nil {
		return fmt.Errorf("Error deleting version %s from IAM policy %s: %s", versionID, arn, err)
	}
	return nil
}

func iamPolicyListVersions(arn string, iamconn *iam.IAM) ([]*iam.PolicyVersion, error) {
	request := &iam.ListPolicyVersionsInput{
		PolicyArn: aws.String(arn),
	}

	response, err := iamconn.ListPolicyVersions(request)
	if err != nil {
		return nil, fmt.Errorf("Error listing versions for IAM policy %s: %s", arn, err)
	}
	return response.Versions, nil
}
