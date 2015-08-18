package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIamPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamPolicyCreate,
		Read:   resourceAwsIamPolicyRead,
		Update: resourceAwsIamPolicyUpdate,
		Delete: resourceAwsIamPolicyDelete,

		Schema: map[string]*schema.Schema{
			"description": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
			"path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "/",
				ForceNew: true,
			},
			"policy": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsIamPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn
	name := d.Get("name").(string)

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

	return readIamPolicy(d, response.Policy)
}

func resourceAwsIamPolicyRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	request := &iam.GetPolicyInput{
		PolicyArn: aws.String(d.Id()),
	}

	response, err := iamconn.GetPolicy(request)
	if err != nil {
		if iamerr, ok := err.(awserr.Error); ok && iamerr.Code() == "NoSuchEntity" {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading IAM policy %s: %s", d.Id(), err)
	}

	return readIamPolicy(d, response.Policy)
}

func resourceAwsIamPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	if err := iamPolicyPruneVersions(d.Id(), iamconn); err != nil {
		return err
	}

	if !d.HasChange("policy") {
		return nil
	}
	request := &iam.CreatePolicyVersionInput{
		PolicyArn:      aws.String(d.Id()),
		PolicyDocument: aws.String(d.Get("policy").(string)),
		SetAsDefault:   aws.Bool(true),
	}

	if _, err := iamconn.CreatePolicyVersion(request); err != nil {
		return fmt.Errorf("Error updating IAM policy %s: %s", d.Id(), err)
	}
	return nil
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
	if err != nil {
		if iamerr, ok := err.(awserr.Error); ok && iamerr.Code() == "NoSuchEntity" {
			return nil
		}
		return fmt.Errorf("Error reading IAM policy %s: %#v", d.Id(), err)
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

func readIamPolicy(d *schema.ResourceData, policy *iam.Policy) error {
	d.SetId(*policy.Arn)
	if policy.Description != nil {
		// the description isn't present in the response to CreatePolicy.
		if err := d.Set("description", *policy.Description); err != nil {
			return err
		}
	}
	if err := d.Set("path", *policy.Path); err != nil {
		return err
	}
	if err := d.Set("name", *policy.PolicyName); err != nil {
		return err
	}
	if err := d.Set("arn", *policy.Arn); err != nil {
		return err
	}
	// TODO: set policy

	return nil
}
