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

func resourceAwsDataSyncLocationEfs() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDataSyncLocationEfsCreate,
		Read:   resourceAwsDataSyncLocationEfsRead,
		Update: resourceAwsDataSyncLocationEfsUpdate,
		Delete: resourceAwsDataSyncLocationEfsDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"ec2_config": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"security_group_arns": {
							Type:     schema.TypeSet,
							Required: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"subnet_arn": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validation.NoZeroValues,
						},
					},
				},
			},
			"efs_file_system_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"subdirectory": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "/",
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

func resourceAwsDataSyncLocationEfsCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datasyncconn

	input := &datasync.CreateLocationEfsInput{
		Ec2Config:        expandDataSyncEc2Config(d.Get("ec2_config").([]interface{})),
		EfsFilesystemArn: aws.String(d.Get("efs_file_system_arn").(string)),
		Subdirectory:     aws.String(d.Get("subdirectory").(string)),
		Tags:             expandDataSyncTagListEntry(d.Get("tags").(map[string]interface{})),
	}

	log.Printf("[DEBUG] Creating DataSync Location EFS: %s", input)
	output, err := conn.CreateLocationEfs(input)
	if err != nil {
		return fmt.Errorf("error creating DataSync Location EFS: %s", err)
	}

	d.SetId(aws.StringValue(output.LocationArn))

	return resourceAwsDataSyncLocationEfsRead(d, meta)
}

func resourceAwsDataSyncLocationEfsRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datasyncconn

	input := &datasync.DescribeLocationEfsInput{
		LocationArn: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Reading DataSync Location EFS: %s", input)
	output, err := conn.DescribeLocationEfs(input)

	if isAWSErr(err, "InvalidRequestException", "not found") {
		log.Printf("[WARN] DataSync Location EFS %q not found - removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading DataSync Location EFS (%s): %s", d.Id(), err)
	}

	tagsInput := &datasync.ListTagsForResourceInput{
		ResourceArn: output.LocationArn,
	}

	log.Printf("[DEBUG] Reading DataSync Location EFS tags: %s", tagsInput)
	tagsOutput, err := conn.ListTagsForResource(tagsInput)

	if err != nil {
		return fmt.Errorf("error reading DataSync Location EFS (%s) tags: %s", d.Id(), err)
	}

	subdirectory, err := dataSyncParseLocationURI(aws.StringValue(output.LocationUri))

	if err != nil {
		return fmt.Errorf("error parsing Location EFS (%s) URI (%s): %s", d.Id(), aws.StringValue(output.LocationUri), err)
	}

	d.Set("arn", output.LocationArn)

	if err := d.Set("ec2_config", flattenDataSyncEc2Config(output.Ec2Config)); err != nil {
		return fmt.Errorf("error setting ec2_config: %s", err)
	}

	d.Set("subdirectory", subdirectory)

	if err := d.Set("tags", flattenDataSyncTagListEntry(tagsOutput.Tags)); err != nil {
		return fmt.Errorf("error setting tags: %s", err)
	}

	d.Set("uri", output.LocationUri)

	return nil
}

func resourceAwsDataSyncLocationEfsUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datasyncconn

	if d.HasChange("tags") {
		oldRaw, newRaw := d.GetChange("tags")
		createTags, removeTags := dataSyncTagsDiff(expandDataSyncTagListEntry(oldRaw.(map[string]interface{})), expandDataSyncTagListEntry(newRaw.(map[string]interface{})))

		if len(removeTags) > 0 {
			input := &datasync.UntagResourceInput{
				Keys:        dataSyncTagsKeys(removeTags),
				ResourceArn: aws.String(d.Id()),
			}

			log.Printf("[DEBUG] Untagging DataSync Location EFS: %s", input)
			if _, err := conn.UntagResource(input); err != nil {
				return fmt.Errorf("error untagging DataSync Location EFS (%s): %s", d.Id(), err)
			}
		}

		if len(createTags) > 0 {
			input := &datasync.TagResourceInput{
				ResourceArn: aws.String(d.Id()),
				Tags:        createTags,
			}

			log.Printf("[DEBUG] Tagging DataSync Location EFS: %s", input)
			if _, err := conn.TagResource(input); err != nil {
				return fmt.Errorf("error tagging DataSync Location EFS (%s): %s", d.Id(), err)
			}
		}
	}

	return resourceAwsDataSyncLocationEfsRead(d, meta)
}

func resourceAwsDataSyncLocationEfsDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datasyncconn

	input := &datasync.DeleteLocationInput{
		LocationArn: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting DataSync Location EFS: %s", input)
	_, err := conn.DeleteLocation(input)

	if isAWSErr(err, "InvalidRequestException", "not found") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting DataSync Location EFS (%s): %s", d.Id(), err)
	}

	return nil
}
