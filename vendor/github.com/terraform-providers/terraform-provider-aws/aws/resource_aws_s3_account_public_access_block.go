package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3control"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsS3AccountPublicAccessBlock() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsS3AccountPublicAccessBlockCreate,
		Read:   resourceAwsS3AccountPublicAccessBlockRead,
		Update: resourceAwsS3AccountPublicAccessBlockUpdate,
		Delete: resourceAwsS3AccountPublicAccessBlockDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"account_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateAwsAccountId,
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

func resourceAwsS3AccountPublicAccessBlockCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).s3controlconn

	accountID := meta.(*AWSClient).accountid
	if v, ok := d.GetOk("account_id"); ok {
		accountID = v.(string)
	}

	input := &s3control.PutPublicAccessBlockInput{
		AccountId: aws.String(accountID),
		PublicAccessBlockConfiguration: &s3control.PublicAccessBlockConfiguration{
			BlockPublicAcls:       aws.Bool(d.Get("block_public_acls").(bool)),
			BlockPublicPolicy:     aws.Bool(d.Get("block_public_policy").(bool)),
			IgnorePublicAcls:      aws.Bool(d.Get("ignore_public_acls").(bool)),
			RestrictPublicBuckets: aws.Bool(d.Get("restrict_public_buckets").(bool)),
		},
	}

	log.Printf("[DEBUG] Creating S3 Account Public Access Block: %s", input)
	_, err := conn.PutPublicAccessBlock(input)
	if err != nil {
		return fmt.Errorf("error creating S3 Account Public Access Block: %s", err)
	}

	d.SetId(accountID)

	return resourceAwsS3AccountPublicAccessBlockRead(d, meta)
}

func resourceAwsS3AccountPublicAccessBlockRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).s3controlconn

	input := &s3control.GetPublicAccessBlockInput{
		AccountId: aws.String(d.Id()),
	}

	// Retry for eventual consistency on creation
	var output *s3control.GetPublicAccessBlockOutput
	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		var err error
		output, err = conn.GetPublicAccessBlock(input)

		if d.IsNewResource() && isAWSErr(err, s3control.ErrCodeNoSuchPublicAccessBlockConfiguration, "") {
			return resource.RetryableError(err)
		}

		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})

	if isAWSErr(err, s3control.ErrCodeNoSuchPublicAccessBlockConfiguration, "") {
		log.Printf("[WARN] S3 Account Public Access Block (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading S3 Account Public Access Block: %s", err)
	}

	if output == nil || output.PublicAccessBlockConfiguration == nil {
		return fmt.Errorf("error reading S3 Account Public Access Block (%s): missing public access block configuration", d.Id())
	}

	d.Set("account_id", d.Id())
	d.Set("block_public_acls", output.PublicAccessBlockConfiguration.BlockPublicAcls)
	d.Set("block_public_policy", output.PublicAccessBlockConfiguration.BlockPublicPolicy)
	d.Set("ignore_public_acls", output.PublicAccessBlockConfiguration.IgnorePublicAcls)
	d.Set("restrict_public_buckets", output.PublicAccessBlockConfiguration.RestrictPublicBuckets)

	return nil
}

func resourceAwsS3AccountPublicAccessBlockUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).s3controlconn

	input := &s3control.PutPublicAccessBlockInput{
		AccountId: aws.String(d.Id()),
		PublicAccessBlockConfiguration: &s3control.PublicAccessBlockConfiguration{
			BlockPublicAcls:       aws.Bool(d.Get("block_public_acls").(bool)),
			BlockPublicPolicy:     aws.Bool(d.Get("block_public_policy").(bool)),
			IgnorePublicAcls:      aws.Bool(d.Get("ignore_public_acls").(bool)),
			RestrictPublicBuckets: aws.Bool(d.Get("restrict_public_buckets").(bool)),
		},
	}

	log.Printf("[DEBUG] Updating S3 Account Public Access Block: %s", input)
	_, err := conn.PutPublicAccessBlock(input)
	if err != nil {
		return fmt.Errorf("error updating S3 Account Public Access Block (%s): %s", d.Id(), err)
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

func resourceAwsS3AccountPublicAccessBlockDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).s3controlconn

	input := &s3control.DeletePublicAccessBlockInput{
		AccountId: aws.String(d.Id()),
	}

	_, err := conn.DeletePublicAccessBlock(input)

	if isAWSErr(err, s3control.ErrCodeNoSuchPublicAccessBlockConfiguration, "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting S3 Account Public Access Block (%s): %s", d.Id(), err)
	}

	return nil
}
