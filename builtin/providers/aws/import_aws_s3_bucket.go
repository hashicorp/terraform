package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsS3BucketImportState(
	d *schema.ResourceData,
	meta interface{}) ([]*schema.ResourceData, error) {

	results := make([]*schema.ResourceData, 1, 1)
	results[0] = d

	conn := meta.(*AWSClient).s3conn
	pol, err := conn.GetBucketPolicy(&s3.GetBucketPolicyInput{
		Bucket: aws.String(d.Id()),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NoSuchBucketPolicy" {
			// Bucket without policy
			return results, nil
		}
		return nil, errwrap.Wrapf("Error importing AWS S3 bucket policy: {{err}}", err)
	}

	policy := resourceAwsS3BucketPolicy()
	pData := policy.Data(nil)
	pData.SetId(d.Id())
	pData.SetType("aws_s3_bucket_policy")
	pData.Set("bucket", d.Id())
	pData.Set("policy", pol)
	results = append(results, pData)

	return results, nil
}
