package aws

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/mitchellh/go-homedir"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
)

func resourceAwsS3BucketObject() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsS3BucketObjectPut,
		Read:   resourceAwsS3BucketObjectRead,
		Update: resourceAwsS3BucketObjectPut,
		Delete: resourceAwsS3BucketObjectDelete,

		Schema: map[string]*schema.Schema{
			"bucket": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"cache_control": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"content_disposition": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"content_encoding": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"content_language": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"content_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"source": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"content"},
			},

			"content": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"source"},
			},

			"kms_key_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"etag": &schema.Schema{
				Type: schema.TypeString,
				// This will conflict with SSE-C and SSE-KMS encryption and multi-part upload
				// if/when it's actually implemented. The Etag then won't match raw-file MD5.
				// See http://docs.aws.amazon.com/AmazonS3/latest/API/RESTCommonResponseHeaders.html
				Optional: true,
				Computed: true,
			},

			"version_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsS3BucketObjectPut(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn

	bucket := d.Get("bucket").(string)
	key := d.Get("key").(string)
	var body io.ReadSeeker

	if v, ok := d.GetOk("source"); ok {
		source := v.(string)
		path, err := homedir.Expand(source)
		if err != nil {
			return fmt.Errorf("Error expanding homedir in source (%s): %s", source, err)
		}
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("Error opening S3 bucket object source (%s): %s", source, err)
		}

		body = file
	} else if v, ok := d.GetOk("content"); ok {
		content := v.(string)
		body = bytes.NewReader([]byte(content))
	} else {
		return fmt.Errorf("Must specify \"source\" or \"content\" field")
	}

	if _, ok := d.GetOk("kms_key_id"); ok {
		if _, ok := d.GetOk("etag"); ok {
			return fmt.Errorf("Unable to specify 'kms_key_id' and 'etag' together because 'etag' wouldn't equal the MD5 digest of the raw object data")
		}
	}

	putInput := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   body,
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

	if v, ok := d.GetOk("kms_key_id"); ok {
		putInput.SSEKMSKeyId = aws.String(v.(string))
		putInput.ServerSideEncryption = aws.String("aws:kms")
	}

	resp, err := s3conn.PutObject(putInput)
	if err != nil {
		return fmt.Errorf("Error putting object in S3 bucket (%s): %s", bucket, err)
	}

	// See https://forums.aws.amazon.com/thread.jspa?threadID=44003
	d.Set("etag", strings.Trim(*resp.ETag, `"`))

	d.Set("version_id", resp.VersionId)
	d.SetId(key)
	return resourceAwsS3BucketObjectRead(d, meta)
}

func resourceAwsS3BucketObjectRead(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn

	bucket := d.Get("bucket").(string)
	key := d.Get("key").(string)
	etag := d.Get("etag").(string)

	resp, err := s3conn.HeadObject(
		&s3.HeadObjectInput{
			Bucket:  aws.String(bucket),
			Key:     aws.String(key),
			IfMatch: aws.String(etag),
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

	d.Set("cache_control", resp.CacheControl)
	d.Set("content_disposition", resp.ContentDisposition)
	d.Set("content_encoding", resp.ContentEncoding)
	d.Set("content_language", resp.ContentLanguage)
	d.Set("content_type", resp.ContentType)
	d.Set("version_id", resp.VersionId)
	d.Set("kms_key_id", resp.SSEKMSKeyId)

	log.Printf("[DEBUG] Reading S3 Bucket Object meta: %s", resp)
	return nil
}

func resourceAwsS3BucketObjectDelete(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn

	bucket := d.Get("bucket").(string)
	key := d.Get("key").(string)

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
			return fmt.Errorf("Error deleting S3 bucket object: %s", err)
		}
	}

	return nil
}
