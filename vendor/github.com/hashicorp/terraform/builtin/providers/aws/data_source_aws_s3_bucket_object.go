package aws

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsS3BucketObject() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsS3BucketObjectRead,

		Schema: map[string]*schema.Schema{
			"body": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"bucket": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"cache_control": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"content_disposition": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"content_encoding": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"content_language": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"content_length": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"content_type": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"etag": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"expiration": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"expires": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"last_modified": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"metadata": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
			},
			"range": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"server_side_encryption": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"sse_kms_key_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"storage_class": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"version_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"website_redirect_location": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchemaComputed(),
		},
	}
}

func dataSourceAwsS3BucketObjectRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).s3conn

	bucket := d.Get("bucket").(string)
	key := d.Get("key").(string)

	input := s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if v, ok := d.GetOk("range"); ok {
		input.Range = aws.String(v.(string))
	}
	if v, ok := d.GetOk("version_id"); ok {
		input.VersionId = aws.String(v.(string))
	}

	versionText := ""
	uniqueId := bucket + "/" + key
	if v, ok := d.GetOk("version_id"); ok {
		versionText = fmt.Sprintf(" of version %q", v.(string))
		uniqueId += "@" + v.(string)
	}

	log.Printf("[DEBUG] Reading S3 object: %s", input)
	out, err := conn.HeadObject(&input)
	if err != nil {
		return fmt.Errorf("Failed getting S3 object: %s Bucket: %q Object: %q", err, bucket, key)
	}
	if out.DeleteMarker != nil && *out.DeleteMarker == true {
		return fmt.Errorf("Requested S3 object %q%s has been deleted",
			bucket+key, versionText)
	}

	log.Printf("[DEBUG] Received S3 object: %s", out)

	d.SetId(uniqueId)

	d.Set("cache_control", out.CacheControl)
	d.Set("content_disposition", out.ContentDisposition)
	d.Set("content_encoding", out.ContentEncoding)
	d.Set("content_language", out.ContentLanguage)
	d.Set("content_length", out.ContentLength)
	d.Set("content_type", out.ContentType)
	// See https://forums.aws.amazon.com/thread.jspa?threadID=44003
	d.Set("etag", strings.Trim(*out.ETag, `"`))
	d.Set("expiration", out.Expiration)
	d.Set("expires", out.Expires)
	d.Set("last_modified", out.LastModified.Format(time.RFC1123))
	d.Set("metadata", pointersMapToStringList(out.Metadata))
	d.Set("server_side_encryption", out.ServerSideEncryption)
	d.Set("sse_kms_key_id", out.SSEKMSKeyId)
	d.Set("version_id", out.VersionId)
	d.Set("website_redirect_location", out.WebsiteRedirectLocation)

	// The "STANDARD" (which is also the default) storage
	// class when set would not be included in the results.
	d.Set("storage_class", s3.StorageClassStandard)
	if out.StorageClass != nil {
		d.Set("storage_class", out.StorageClass)
	}

	if isContentTypeAllowed(out.ContentType) {
		input := s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		}
		if v, ok := d.GetOk("range"); ok {
			input.Range = aws.String(v.(string))
		}
		if out.VersionId != nil {
			input.VersionId = out.VersionId
		}
		out, err := conn.GetObject(&input)
		if err != nil {
			return fmt.Errorf("Failed getting S3 object: %s", err)
		}

		buf := new(bytes.Buffer)
		bytesRead, err := buf.ReadFrom(out.Body)
		if err != nil {
			return fmt.Errorf("Failed reading content of S3 object (%s): %s",
				uniqueId, err)
		}
		log.Printf("[INFO] Saving %d bytes from S3 object %s", bytesRead, uniqueId)
		d.Set("body", buf.String())
	} else {
		contentType := ""
		if out.ContentType == nil {
			contentType = "<EMPTY>"
		} else {
			contentType = *out.ContentType
		}

		log.Printf("[INFO] Ignoring body of S3 object %s with Content-Type %q",
			uniqueId, contentType)
	}

	tagResp, err := conn.GetObjectTagging(
		&s3.GetObjectTaggingInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
	if err != nil {
		return err
	}
	d.Set("tags", tagsToMapS3(tagResp.TagSet))

	return nil
}

// This is to prevent potential issues w/ binary files
// and generally unprintable characters
// See https://github.com/hashicorp/terraform/pull/3858#issuecomment-156856738
func isContentTypeAllowed(contentType *string) bool {
	if contentType == nil {
		return false
	}

	allowedContentTypes := []*regexp.Regexp{
		regexp.MustCompile("^text/.+"),
		regexp.MustCompile("^application/json$"),
	}

	for _, r := range allowedContentTypes {
		if r.MatchString(*contentType) {
			return true
		}
	}

	return false
}
