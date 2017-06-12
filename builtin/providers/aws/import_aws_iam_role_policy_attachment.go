package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIamRolePolicyAttachmentImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	conn := meta.(*AWSClient).iamconn
	roleName := d.Id()

	resp, err := conn.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return nil, err
	}

	results := []*schema.ResourceData{}

	for _, policy := range resp.AttachedPolicies {
		id := fmt.Sprintf("%s-%s", roleName, *policy.PolicyArn)

		attachment := resourceAwsIamRolePolicyAttachment()
		ad := attachment.Data(nil)
		ad.SetId(id)
		ad.SetType("aws_iam_role_policy_attachment")
		ad.Set("role", roleName)
		ad.Set("policy_arn", policy.PolicyArn)

		results = append(results, ad)
	}

	return results, nil
}
