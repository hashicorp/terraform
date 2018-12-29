package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsCloudFrontOriginAccessIdentity() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudFrontOriginAccessIdentityCreate,
		Read:   resourceAwsCloudFrontOriginAccessIdentityRead,
		Update: resourceAwsCloudFrontOriginAccessIdentityUpdate,
		Delete: resourceAwsCloudFrontOriginAccessIdentityDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"comment": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"caller_reference": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cloudfront_access_identity_path": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"etag": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"iam_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"s3_canonical_user_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsCloudFrontOriginAccessIdentityCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudfrontconn
	params := &cloudfront.CreateCloudFrontOriginAccessIdentityInput{
		CloudFrontOriginAccessIdentityConfig: expandOriginAccessIdentityConfig(d),
	}

	resp, err := conn.CreateCloudFrontOriginAccessIdentity(params)
	if err != nil {
		return err
	}
	d.SetId(*resp.CloudFrontOriginAccessIdentity.Id)
	return resourceAwsCloudFrontOriginAccessIdentityRead(d, meta)
}

func resourceAwsCloudFrontOriginAccessIdentityRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudfrontconn
	params := &cloudfront.GetCloudFrontOriginAccessIdentityInput{
		Id: aws.String(d.Id()),
	}

	resp, err := conn.GetCloudFrontOriginAccessIdentity(params)
	if err != nil {
		return err
	}

	// Update attributes from DistributionConfig
	flattenOriginAccessIdentityConfig(d, resp.CloudFrontOriginAccessIdentity.CloudFrontOriginAccessIdentityConfig)
	// Update other attributes outside of DistributionConfig
	d.SetId(*resp.CloudFrontOriginAccessIdentity.Id)
	d.Set("etag", resp.ETag)
	d.Set("s3_canonical_user_id", resp.CloudFrontOriginAccessIdentity.S3CanonicalUserId)
	d.Set("cloudfront_access_identity_path", fmt.Sprintf("origin-access-identity/cloudfront/%s", *resp.CloudFrontOriginAccessIdentity.Id))
	iamArn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Service:   "iam",
		AccountID: "cloudfront",
		Resource:  fmt.Sprintf("user/CloudFront Origin Access Identity %s", *resp.CloudFrontOriginAccessIdentity.Id),
	}.String()
	d.Set("iam_arn", iamArn)
	return nil
}

func resourceAwsCloudFrontOriginAccessIdentityUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudfrontconn
	params := &cloudfront.UpdateCloudFrontOriginAccessIdentityInput{
		Id:                                   aws.String(d.Id()),
		CloudFrontOriginAccessIdentityConfig: expandOriginAccessIdentityConfig(d),
		IfMatch:                              aws.String(d.Get("etag").(string)),
	}
	_, err := conn.UpdateCloudFrontOriginAccessIdentity(params)
	if err != nil {
		return err
	}

	return resourceAwsCloudFrontOriginAccessIdentityRead(d, meta)
}

func resourceAwsCloudFrontOriginAccessIdentityDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudfrontconn
	params := &cloudfront.DeleteCloudFrontOriginAccessIdentityInput{
		Id:      aws.String(d.Id()),
		IfMatch: aws.String(d.Get("etag").(string)),
	}

	_, err := conn.DeleteCloudFrontOriginAccessIdentity(params)

	return err
}

func expandOriginAccessIdentityConfig(d *schema.ResourceData) *cloudfront.OriginAccessIdentityConfig {
	originAccessIdentityConfig := &cloudfront.OriginAccessIdentityConfig{
		Comment: aws.String(d.Get("comment").(string)),
	}
	// This sets CallerReference if it's still pending computation (ie: new resource)
	if v, ok := d.GetOk("caller_reference"); !ok {
		originAccessIdentityConfig.CallerReference = aws.String(time.Now().Format(time.RFC3339Nano))
	} else {
		originAccessIdentityConfig.CallerReference = aws.String(v.(string))
	}
	return originAccessIdentityConfig
}

func flattenOriginAccessIdentityConfig(d *schema.ResourceData, originAccessIdentityConfig *cloudfront.OriginAccessIdentityConfig) {
	if originAccessIdentityConfig.Comment != nil {
		d.Set("comment", originAccessIdentityConfig.Comment)
	}
	d.Set("caller_reference", originAccessIdentityConfig.CallerReference)
}
