package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/datasync"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsDataSyncLocationS3() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDataSyncLocationS3Create,
		Read:   resourceAwsDataSyncLocationS3Read,
		Update: resourceAwsDataSyncLocationS3Update,
		Delete: resourceAwsDataSyncLocationS3Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"s3_bucket_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"s3_config": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bucket_access_role_arn": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validation.NoZeroValues,
						},
					},
				},
			},
			"subdirectory": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				// Ignore missing trailing slash
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if new == "/" {
						return false
					}
					if strings.TrimSuffix(old, "/") == strings.TrimSuffix(new, "/") {
						return true
					}
					return false
				},
			},
			"tags": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsDataSyncLocationS3Create(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datasyncconn

	input := &datasync.CreateLocationS3Input{
		S3BucketArn:  aws.String(d.Get("s3_bucket_arn").(string)),
		S3Config:     expandDataSyncS3Config(d.Get("s3_config").([]interface{})),
		Subdirectory: aws.String(d.Get("subdirectory").(string)),
		Tags:         expandDataSyncTagListEntry(d.Get("tags").(map[string]interface{})),
	}

	log.Printf("[DEBUG] Creating DataSync Location S3: %s", input)
	output, err := conn.CreateLocationS3(input)
	if err != nil {
		return fmt.Errorf("error creating DataSync Location S3: %s", err)
	}

	d.SetId(aws.StringValue(output.LocationArn))

	return resourceAwsDataSyncLocationS3Read(d, meta)
}

func resourceAwsDataSyncLocationS3Read(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datasyncconn

	input := &datasync.DescribeLocationS3Input{
		LocationArn: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Reading DataSync Location S3: %s", input)
	output, err := conn.DescribeLocationS3(input)

	if isAWSErr(err, "InvalidRequestException", "not found") {
		log.Printf("[WARN] DataSync Location S3 %q not found - removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading DataSync Location S3 (%s): %s", d.Id(), err)
	}

	tagsInput := &datasync.ListTagsForResourceInput{
		ResourceArn: output.LocationArn,
	}

	log.Printf("[DEBUG] Reading DataSync Location S3 tags: %s", tagsInput)
	tagsOutput, err := conn.ListTagsForResource(tagsInput)

	if err != nil {
		return fmt.Errorf("error reading DataSync Location S3 (%s) tags: %s", d.Id(), err)
	}

	subdirectory, err := dataSyncParseLocationURI(aws.StringValue(output.LocationUri))

	if err != nil {
		return fmt.Errorf("error parsing Location S3 (%s) URI (%s): %s", d.Id(), aws.StringValue(output.LocationUri), err)
	}

	d.Set("arn", output.LocationArn)

	if err := d.Set("s3_config", flattenDataSyncS3Config(output.S3Config)); err != nil {
		return fmt.Errorf("error setting s3_config: %s", err)
	}

	d.Set("subdirectory", subdirectory)

	if err := d.Set("tags", flattenDataSyncTagListEntry(tagsOutput.Tags)); err != nil {
		return fmt.Errorf("error setting tags: %s", err)
	}

	d.Set("uri", output.LocationUri)

	return nil
}

func resourceAwsDataSyncLocationS3Update(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datasyncconn

	if d.HasChange("tags") {
		oldRaw, newRaw := d.GetChange("tags")
		createTags, removeTags := dataSyncTagsDiff(expandDataSyncTagListEntry(oldRaw.(map[string]interface{})), expandDataSyncTagListEntry(newRaw.(map[string]interface{})))

		if len(removeTags) > 0 {
			input := &datasync.UntagResourceInput{
				Keys:        dataSyncTagsKeys(removeTags),
				ResourceArn: aws.String(d.Id()),
			}

			log.Printf("[DEBUG] Untagging DataSync Location S3: %s", input)
			if _, err := conn.UntagResource(input); err != nil {
				return fmt.Errorf("error untagging DataSync Location S3 (%s): %s", d.Id(), err)
			}
		}

		if len(createTags) > 0 {
			input := &datasync.TagResourceInput{
				ResourceArn: aws.String(d.Id()),
				Tags:        createTags,
			}

			log.Printf("[DEBUG] Tagging DataSync Location S3: %s", input)
			if _, err := conn.TagResource(input); err != nil {
				return fmt.Errorf("error tagging DataSync Location S3 (%s): %s", d.Id(), err)
			}
		}
	}

	return resourceAwsDataSyncLocationS3Read(d, meta)
}

func resourceAwsDataSyncLocationS3Delete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datasyncconn

	input := &datasync.DeleteLocationInput{
		LocationArn: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting DataSync Location S3: %s", input)
	_, err := conn.DeleteLocation(input)

	if isAWSErr(err, "InvalidRequestException", "not found") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting DataSync Location S3 (%s): %s", d.Id(), err)
	}

	return nil
}
