package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsS3BucketPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsS3BucketPolicyPut,
		Read:   resourceAwsS3BucketPolicyRead,
		Update: resourceAwsS3BucketPolicyPut,
		Delete: resourceAwsS3BucketPolicyDelete,

		Schema: map[string]*schema.Schema{
			"bucket": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"policy": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateFunc:     validation.ValidateJsonString,
				DiffSuppressFunc: suppressEquivalentAwsPolicyDiffs,
			},
		},
	}
}

func resourceAwsS3BucketPolicyPut(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn

	bucket := d.Get("bucket").(string)
	policy := d.Get("policy").(string)

	log.Printf("[DEBUG] S3 bucket: %s, put policy: %s", bucket, policy)

	params := &s3.PutBucketPolicyInput{
		Bucket: aws.String(bucket),
		Policy: aws.String(policy),
	}

	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		if _, err := s3conn.PutBucketPolicy(params); err != nil {
			if awserr, ok := err.(awserr.Error); ok {
				if awserr.Code() == "MalformedPolicy" {
					return resource.RetryableError(awserr)
				}
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("Error putting S3 policy: %s", err)
	}

	d.SetId(bucket)

	return nil
}

func resourceAwsS3BucketPolicyRead(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn

	log.Printf("[DEBUG] S3 bucket policy, read for bucket: %s", d.Id())
	pol, err := s3conn.GetBucketPolicy(&s3.GetBucketPolicyInput{
		Bucket: aws.String(d.Id()),
	})

	v := ""
	if err == nil && pol.Policy != nil {
		v = *pol.Policy
	}
	if err := d.Set("policy", v); err != nil {
		return err
	}

	return nil
}

func resourceAwsS3BucketPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn

	bucket := d.Get("bucket").(string)

	log.Printf("[DEBUG] S3 bucket: %s, delete policy", bucket)
	_, err := s3conn.DeleteBucketPolicy(&s3.DeleteBucketPolicyInput{
		Bucket: aws.String(bucket),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NoSuchBucket" {
			return nil
		}
		return fmt.Errorf("Error deleting S3 policy: %s", err)
	}

	return nil
}
