package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/service/s3control"
	"github.com/hashicorp/terraform/helper/resource"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsS3BucketPublicAccessBlock() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsS3BucketPublicAccessBlockCreate,
		Read:   resourceAwsS3BucketPublicAccessBlockRead,
		Update: resourceAwsS3BucketPublicAccessBlockUpdate,
		Delete: resourceAwsS3BucketPublicAccessBlockDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"bucket": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"block_public_acls": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"block_public_policy": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"ignore_public_acls": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"restrict_public_buckets": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func resourceAwsS3BucketPublicAccessBlockCreate(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn
	bucket := d.Get("bucket").(string)

	input := &s3.PutPublicAccessBlockInput{
		Bucket: aws.String(bucket),
		PublicAccessBlockConfiguration: &s3.PublicAccessBlockConfiguration{
			BlockPublicAcls:       aws.Bool(d.Get("block_public_acls").(bool)),
			BlockPublicPolicy:     aws.Bool(d.Get("block_public_policy").(bool)),
			IgnorePublicAcls:      aws.Bool(d.Get("ignore_public_acls").(bool)),
			RestrictPublicBuckets: aws.Bool(d.Get("restrict_public_buckets").(bool)),
		},
	}

	log.Printf("[DEBUG] S3 bucket: %s, public access block: %v", bucket, input.PublicAccessBlockConfiguration)
	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, err := s3conn.PutPublicAccessBlock(input)

		if isAWSErr(err, s3.ErrCodeNoSuchBucket, "") {
			return resource.RetryableError(err)
		}

		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error creating public access block policy for S3 bucket (%s): %s", bucket, err)
	}

	d.SetId(bucket)
	return resourceAwsS3BucketPublicAccessBlockRead(d, meta)
}

func resourceAwsS3BucketPublicAccessBlockRead(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn

	input := &s3.GetPublicAccessBlockInput{
		Bucket: aws.String(d.Id()),
	}

	// Retry for eventual consistency on creation
	var output *s3.GetPublicAccessBlockOutput
	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		var err error
		output, err = s3conn.GetPublicAccessBlock(input)

		if d.IsNewResource() && (isAWSErr(err, s3control.ErrCodeNoSuchPublicAccessBlockConfiguration, "") ||
			isAWSErr(err, s3.ErrCodeNoSuchBucket, "")) {
			return resource.RetryableError(err)
		}

		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})

	if isAWSErr(err, s3control.ErrCodeNoSuchPublicAccessBlockConfiguration, "") {
		log.Printf("[WARN] S3 Bucket Public Access Block (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading S3 bucket Public Access Block: %s", err)
	}

	if output == nil || output.PublicAccessBlockConfiguration == nil {
		return fmt.Errorf("error reading S3 Bucket Public Access Block (%s): missing public access block configuration", d.Id())
	}

	d.Set("bucket", d.Id())
	d.Set("block_public_acls", output.PublicAccessBlockConfiguration.BlockPublicAcls)
	d.Set("block_public_policy", output.PublicAccessBlockConfiguration.BlockPublicPolicy)
	d.Set("ignore_public_acls", output.PublicAccessBlockConfiguration.IgnorePublicAcls)
	d.Set("restrict_public_buckets", output.PublicAccessBlockConfiguration.RestrictPublicBuckets)

	return nil
}

func resourceAwsS3BucketPublicAccessBlockUpdate(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn

	input := &s3.PutPublicAccessBlockInput{
		Bucket: aws.String(d.Id()),
		PublicAccessBlockConfiguration: &s3.PublicAccessBlockConfiguration{
			BlockPublicAcls:       aws.Bool(d.Get("block_public_acls").(bool)),
			BlockPublicPolicy:     aws.Bool(d.Get("block_public_policy").(bool)),
			IgnorePublicAcls:      aws.Bool(d.Get("ignore_public_acls").(bool)),
			RestrictPublicBuckets: aws.Bool(d.Get("restrict_public_buckets").(bool)),
		},
	}

	log.Printf("[DEBUG] Updating S3 bucket Public Access Block: %s", input)
	_, err := s3conn.PutPublicAccessBlock(input)
	if err != nil {
		return fmt.Errorf("error updating S3 Bucket Public Access Block (%s): %s", d.Id(), err)
	}

	// Workaround API eventual consistency issues. This type of logic should not normally be used.
	// We cannot reliably determine when the Read after Update might be properly updated.
	// Rather than introduce complicated retry logic, we presume that a lack of an update error
	// means our update succeeded with our expected values.
	d.Set("block_public_acls", input.PublicAccessBlockConfiguration.BlockPublicAcls)
	d.Set("block_public_policy", input.PublicAccessBlockConfiguration.BlockPublicPolicy)
	d.Set("ignore_public_acls", input.PublicAccessBlockConfiguration.IgnorePublicAcls)
	d.Set("restrict_public_buckets", input.PublicAccessBlockConfiguration.RestrictPublicBuckets)

	// Skip normal Read after Update due to eventual consistency issues
	return nil
}

func resourceAwsS3BucketPublicAccessBlockDelete(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn

	input := &s3.DeletePublicAccessBlockInput{
		Bucket: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] S3 bucket: %s, delete public access block", d.Id())
	_, err := s3conn.DeletePublicAccessBlock(input)

	if isAWSErr(err, s3control.ErrCodeNoSuchPublicAccessBlockConfiguration, "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting S3 Bucket Public Access Block (%s): %s", d.Id(), err)
	}

	return nil
}
