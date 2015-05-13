package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/s3"
	"github.com/awslabs/aws-sdk-go/aws/awsutil"
)

func resourceAwsS3BucketPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsS3BucketPolicyPut,
		Update: resourceAwsS3BucketPolicyPut,
		Read:   resourceAwsS3BucketPolicyRead,
		Delete: resourceAwsS3BucketPolicyDelete,

		Schema: map[string]*schema.Schema{
			"bucket": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"policy": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsS3BucketPolicyPut(d *schema.ResourceData, meta interface{}) error {

	s3conn := meta.(*AWSClient).s3conn
	bucket := d.Get("bucket").(string)
	policy := d.Get("policy").(string)
	name   := d.Get("name").(string)

	resp, err := s3conn.PutBucketPolicy(
		&s3.PutBucketPolicyInput{
			Bucket: aws.String(bucket),
			Policy: aws.String(policy),
		})

	log.Printf("[DEBUG] S3 bucket policy set (response): %s", awsutil.StringValue(resp));

	if err != nil {
		return fmt.Errorf("Error adding policy to S3 bucket: %s", err)
	}

	d.SetId(fmt.Sprintf("%s:%s", bucket, ))
	return nil
}

func resourceAwsS3BucketPolicyRead(d *schema.ResourceData, m interface{}) error {
	return nil
}

func resourceAwsS3BucketPolicyUpdate(d *schema.ResourceData, m interface{}) error {
	return nil
}

func resourceAwsS3BucketPolicyDelete(d *schema.ResourceData, m interface{}) error {
	return nil
}
