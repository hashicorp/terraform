package aws

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/mitchellh/go-homedir"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/s3"
)

func resourceAwsS3BucketObject() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsS3BucketObjectCreate,
		Read:   resourceAwsS3BucketObjectRead,
		Update: resourceAwsS3BucketObjectUpdate,
		Delete: resourceAwsS3BucketObjectDelete,

		CustomizeDiff: resourceAwsS3BucketObjectCustomizeDiff,

		Schema: map[string]*schema.Schema{
			"bucket": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"key": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"acl": {
				Type:     schema.TypeString,
				Default:  s3.ObjectCannedACLPrivate,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					s3.ObjectCannedACLPrivate,
					s3.ObjectCannedACLPublicRead,
					s3.ObjectCannedACLPublicReadWrite,
					s3.ObjectCannedACLAuthenticatedRead,
					s3.ObjectCannedACLAwsExecRead,
					s3.ObjectCannedACLBucketOwnerRead,
					s3.ObjectCannedACLBucketOwnerFullControl,
				}, false),
			},

			"cache_control": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"content_disposition": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"content_encoding": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"content_language": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"content_type": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"source": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"content", "content_base64"},
			},

			"content": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"source", "content_base64"},
			},

			"content_base64": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"source", "content"},
			},

			"storage_class": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ValidateFunc: validation.StringInSlice([]string{
					s3.ObjectStorageClassStandard,
					s3.ObjectStorageClassReducedRedundancy,
					s3.ObjectStorageClassGlacier,
					s3.ObjectStorageClassStandardIa,
					s3.ObjectStorageClassOnezoneIa,
					s3.ObjectStorageClassIntelligentTiering,
				}, false),
			},

			"server_side_encryption": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					s3.ServerSideEncryptionAes256,
					s3.ServerSideEncryptionAwsKms,
				}, false),
				Computed: true,
			},

			"kms_key_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateArn,
			},

			"etag": {
				Type: schema.TypeString,
				// This will conflict with SSE-C and SSE-KMS encryption and multi-part upload
				// if/when it's actually implemented. The Etag then won't match raw-file MD5.
				// See http://docs.aws.amazon.com/AmazonS3/latest/API/RESTCommonResponseHeaders.html
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"kms_key_id", "server_side_encryption"},
			},

			"version_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),

			"website_redirect": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceAwsS3BucketObjectPut(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn

	restricted := meta.(*AWSClient).IsChinaCloud()

	var body io.ReadSeeker

	if v, ok := d.GetOk("source"); ok {
		source := v.(string)
		path, err := homedir.Expand(source)
		if err != nil {
			return fmt.Errorf("Error expanding homedir in source (%s): %s", source, err)
		}
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("Error opening S3 bucket object source (%s): %s", path, err)
		}

		body = file
		defer func() {
			err := file.Close()
			if err != nil {
				log.Printf("[WARN] Error closing S3 bucket object source (%s): %s", path, err)
			}
		}()
	} else if v, ok := d.GetOk("content"); ok {
		content := v.(string)
		body = bytes.NewReader([]byte(content))
	} else if v, ok := d.GetOk("content_base64"); ok {
		content := v.(string)
		// We can't do streaming decoding here (with base64.NewDecoder) because
		// the AWS SDK requires an io.ReadSeeker but a base64 decoder can't seek.
		contentRaw, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			return fmt.Errorf("error decoding content_base64: %s", err)
		}
		body = bytes.NewReader(contentRaw)
	} else {
		return fmt.Errorf("Must specify \"source\", \"content\", or \"content_base64\" field")
	}

	bucket := d.Get("bucket").(string)
	key := d.Get("key").(string)

	putInput := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		ACL:    aws.String(d.Get("acl").(string)),
		Body:   body,
	}

	if v, ok := d.GetOk("storage_class"); ok {
		putInput.StorageClass = aws.String(v.(string))
	}

	if v, ok := d.GetOk("cache_control"); ok {
		putInput.CacheControl = aws.String(v.(string))
	}

	if v, ok := d.GetOk("content_type"); ok {
		putInput.ContentType = aws.String(v.(string))
	}

	if v, ok := d.GetOk("content_encoding"); ok {
		putInput.ContentEncoding = aws.String(v.(string))
	}

	if v, ok := d.GetOk("content_language"); ok {
		putInput.ContentLanguage = aws.String(v.(string))
	}

	if v, ok := d.GetOk("content_disposition"); ok {
		putInput.ContentDisposition = aws.String(v.(string))
	}

	if v, ok := d.GetOk("server_side_encryption"); ok {
		putInput.ServerSideEncryption = aws.String(v.(string))
	}

	if v, ok := d.GetOk("kms_key_id"); ok {
		putInput.SSEKMSKeyId = aws.String(v.(string))
		putInput.ServerSideEncryption = aws.String(s3.ServerSideEncryptionAwsKms)
	}

	if v, ok := d.GetOk("tags"); ok {
		if restricted {
			return fmt.Errorf("This region does not allow for tags on S3 objects")
		}

		// The tag-set must be encoded as URL Query parameters.
		values := url.Values{}
		for k, v := range v.(map[string]interface{}) {
			values.Add(k, v.(string))
		}
		putInput.Tagging = aws.String(values.Encode())
	}

	if v, ok := d.GetOk("website_redirect"); ok {
		putInput.WebsiteRedirectLocation = aws.String(v.(string))
	}

	if _, err := s3conn.PutObject(putInput); err != nil {
		return fmt.Errorf("Error putting object in S3 bucket (%s): %s", bucket, err)
	}

	d.SetId(key)
	return resourceAwsS3BucketObjectRead(d, meta)
}

func resourceAwsS3BucketObjectCreate(d *schema.ResourceData, meta interface{}) error {
	return resourceAwsS3BucketObjectPut(d, meta)
}

func resourceAwsS3BucketObjectRead(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn

	restricted := meta.(*AWSClient).IsChinaCloud()

	bucket := d.Get("bucket").(string)
	key := d.Get("key").(string)

	resp, err := s3conn.HeadObject(
		&s3.HeadObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})

	if err != nil {
		// If S3 returns a 404 Request Failure, mark the object as destroyed
		if awsErr, ok := err.(awserr.RequestFailure); ok && awsErr.StatusCode() == 404 {
			d.SetId("")
			log.Printf("[WARN] Error Reading Object (%s), object not found (HTTP status 404)", key)
			return nil
		}
		return err
	}
	log.Printf("[DEBUG] Reading S3 Bucket Object meta: %s", resp)

	d.Set("cache_control", resp.CacheControl)
	d.Set("content_disposition", resp.ContentDisposition)
	d.Set("content_encoding", resp.ContentEncoding)
	d.Set("content_language", resp.ContentLanguage)
	d.Set("content_type", resp.ContentType)
	d.Set("version_id", resp.VersionId)
	d.Set("server_side_encryption", resp.ServerSideEncryption)
	d.Set("website_redirect", resp.WebsiteRedirectLocation)

	// Only set non-default KMS key ID (one that doesn't match default)
	if resp.SSEKMSKeyId != nil {
		// retrieve S3 KMS Default Master Key
		kmsconn := meta.(*AWSClient).kmsconn
		kmsresp, err := kmsconn.DescribeKey(&kms.DescribeKeyInput{
			KeyId: aws.String("alias/aws/s3"),
		})
		if err != nil {
			return fmt.Errorf("Failed to describe default S3 KMS key (alias/aws/s3): %s", err)
		}

		if *resp.SSEKMSKeyId != *kmsresp.KeyMetadata.Arn {
			log.Printf("[DEBUG] S3 object is encrypted using a non-default KMS Key ID: %s", *resp.SSEKMSKeyId)
			d.Set("kms_key_id", resp.SSEKMSKeyId)
		}
	}
	// See https://forums.aws.amazon.com/thread.jspa?threadID=44003
	d.Set("etag", strings.Trim(aws.StringValue(resp.ETag), `"`))

	// The "STANDARD" (which is also the default) storage
	// class when set would not be included in the results.
	d.Set("storage_class", s3.StorageClassStandard)
	if resp.StorageClass != nil {
		d.Set("storage_class", resp.StorageClass)
	}

	if !restricted {
		if err := getTagsS3Object(s3conn, d); err != nil {
			return fmt.Errorf("error getting S3 object tags (bucket: %s, key: %s): %s", bucket, key, err)
		}
	}

	return nil
}

func resourceAwsS3BucketObjectUpdate(d *schema.ResourceData, meta interface{}) error {
	// Changes to any of these attributes requires creation of a new object version (if bucket is versioned):
	for _, key := range []string{
		"cache_control",
		"content_disposition",
		"content_encoding",
		"content_language",
		"content_type",
		"source",
		"content",
		"content_base64",
		"storage_class",
		"server_side_encryption",
		"kms_key_id",
		"etag",
		"website_redirect",
	} {
		if d.HasChange(key) {
			return resourceAwsS3BucketObjectPut(d, meta)
		}
	}

	conn := meta.(*AWSClient).s3conn

	if d.HasChange("acl") {
		_, err := conn.PutObjectAcl(&s3.PutObjectAclInput{
			Bucket: aws.String(d.Get("bucket").(string)),
			Key:    aws.String(d.Get("key").(string)),
			ACL:    aws.String(d.Get("acl").(string)),
		})
		if err != nil {
			return fmt.Errorf("error putting S3 object ACL: %s", err)
		}
	}

	if err := setTagsS3Object(conn, d); err != nil {
		return fmt.Errorf("error setting S3 object tags: %s", err)
	}

	return resourceAwsS3BucketObjectRead(d, meta)
}

func resourceAwsS3BucketObjectDelete(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn

	bucket := d.Get("bucket").(string)
	key := d.Get("key").(string)
	// We are effectively ignoring any leading '/' in the key name as aws.Config.DisableRestProtocolURICleaning is false
	// so we need to explicitly ignore any leading '/' in the s3.ListObjectVersions call.
	if strings.HasPrefix(key, "/") {
		key = key[1:]
	}

	if _, ok := d.GetOk("version_id"); ok {
		// Bucket is versioned, we need to delete all versions
		vInput := s3.ListObjectVersionsInput{
			Bucket: aws.String(bucket),
			Prefix: aws.String(key),
		}
		out, err := s3conn.ListObjectVersions(&vInput)
		if err != nil {
			return fmt.Errorf("Failed listing S3 object versions: %s", err)
		}

		for _, v := range out.Versions {
			input := s3.DeleteObjectInput{
				Bucket:    aws.String(bucket),
				Key:       aws.String(key),
				VersionId: v.VersionId,
			}
			_, err := s3conn.DeleteObject(&input)
			if err != nil {
				return fmt.Errorf("Error deleting S3 object version of %s:\n %s:\n %s",
					key, v, err)
			}
		}
	} else {
		// Just delete the object
		input := s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		}
		_, err := s3conn.DeleteObject(&input)
		if err != nil {
			return fmt.Errorf("Error deleting S3 bucket object: %s  Bucket: %q Object: %q", err, bucket, key)
		}
	}

	return nil
}

func resourceAwsS3BucketObjectCustomizeDiff(d *schema.ResourceDiff, meta interface{}) error {
	if d.HasChange("etag") {
		d.SetNewComputed("version_id")
	}

	return nil
}
