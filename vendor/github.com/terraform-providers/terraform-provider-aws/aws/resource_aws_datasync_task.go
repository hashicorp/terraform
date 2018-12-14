package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/datasync"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsDataSyncTask() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDataSyncTaskCreate,
		Read:   resourceAwsDataSyncTaskRead,
		Update: resourceAwsDataSyncTaskUpdate,
		Delete: resourceAwsDataSyncTaskDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cloudwatch_log_group_arn": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"destination_location_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"options": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				// Ignore missing configuration block
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if old == "1" && new == "0" {
						return true
					}
					return false
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"atime": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  datasync.AtimeBestEffort,
							ValidateFunc: validation.StringInSlice([]string{
								datasync.AtimeBestEffort,
								datasync.AtimeNone,
							}, false),
						},
						"bytes_per_second": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      -1,
							ValidateFunc: validation.IntAtLeast(-1),
						},
						"gid": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  datasync.GidIntValue,
							ValidateFunc: validation.StringInSlice([]string{
								datasync.GidBoth,
								datasync.GidIntValue,
								datasync.GidName,
								datasync.GidNone,
							}, false),
						},
						"mtime": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  datasync.MtimePreserve,
							ValidateFunc: validation.StringInSlice([]string{
								datasync.MtimeNone,
								datasync.MtimePreserve,
							}, false),
						},
						"posix_permissions": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  datasync.PosixPermissionsPreserve,
							ValidateFunc: validation.StringInSlice([]string{
								datasync.PosixPermissionsBestEffort,
								datasync.PosixPermissionsNone,
								datasync.PosixPermissionsPreserve,
							}, false),
						},
						"preserve_deleted_files": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  datasync.PreserveDeletedFilesPreserve,
							ValidateFunc: validation.StringInSlice([]string{
								datasync.PreserveDeletedFilesPreserve,
								datasync.PreserveDeletedFilesRemove,
							}, false),
						},
						"preserve_devices": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  datasync.PreserveDevicesNone,
							ValidateFunc: validation.StringInSlice([]string{
								datasync.PreserveDevicesNone,
								datasync.PreserveDevicesPreserve,
							}, false),
						},
						"uid": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  datasync.UidIntValue,
							ValidateFunc: validation.StringInSlice([]string{
								datasync.UidBoth,
								datasync.UidIntValue,
								datasync.UidName,
								datasync.UidNone,
							}, false),
						},
						"verify_mode": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  datasync.VerifyModePointInTimeConsistent,
							ValidateFunc: validation.StringInSlice([]string{
								datasync.VerifyModeNone,
								datasync.VerifyModePointInTimeConsistent,
							}, false),
						},
					},
				},
			},
			"source_location_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"tags": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceAwsDataSyncTaskCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datasyncconn

	input := &datasync.CreateTaskInput{
		DestinationLocationArn: aws.String(d.Get("destination_location_arn").(string)),
		Options:                expandDataSyncOptions(d.Get("options").([]interface{})),
		SourceLocationArn:      aws.String(d.Get("source_location_arn").(string)),
		Tags:                   expandDataSyncTagListEntry(d.Get("tags").(map[string]interface{})),
	}

	if v, ok := d.GetOk("cloudwatch_log_group_arn"); ok {
		input.CloudWatchLogGroupArn = aws.String(v.(string))
	}

	if v, ok := d.GetOk("name"); ok {
		input.Name = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Creating DataSync Task: %s", input)
	output, err := conn.CreateTask(input)
	if err != nil {
		return fmt.Errorf("error creating DataSync Task: %s", err)
	}

	d.SetId(aws.StringValue(output.TaskArn))

	// Task creation can take a few minutes
	err = resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		output, err := conn.DescribeTask(&datasync.DescribeTaskInput{
			TaskArn: aws.String(d.Id()),
		})

		if isAWSErr(err, "InvalidRequestException", "not found") {
			return resource.RetryableError(err)
		}

		if err != nil {
			return resource.NonRetryableError(err)
		}

		if aws.StringValue(output.Status) == datasync.TaskStatusAvailable || aws.StringValue(output.Status) == datasync.TaskStatusRunning {
			return nil
		}

		err = fmt.Errorf("waiting for DataSync Task (%s) creation: last status (%s), error code (%s), error detail: %s",
			d.Id(), aws.StringValue(output.Status), aws.StringValue(output.ErrorCode), aws.StringValue(output.ErrorDetail))

		if aws.StringValue(output.Status) == datasync.TaskStatusCreating {
			return resource.RetryableError(err)
		}

		return resource.NonRetryableError(err)
	})
	if err != nil {
		return fmt.Errorf("error waiting for DataSync Task (%s) creation: %s", d.Id(), err)
	}

	return resourceAwsDataSyncTaskRead(d, meta)
}

func resourceAwsDataSyncTaskRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datasyncconn

	input := &datasync.DescribeTaskInput{
		TaskArn: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Reading DataSync Task: %s", input)
	output, err := conn.DescribeTask(input)

	if isAWSErr(err, "InvalidRequestException", "not found") {
		log.Printf("[WARN] DataSync Task %q not found - removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading DataSync Task (%s): %s", d.Id(), err)
	}

	tagsInput := &datasync.ListTagsForResourceInput{
		ResourceArn: output.TaskArn,
	}

	log.Printf("[DEBUG] Reading DataSync Task tags: %s", tagsInput)
	tagsOutput, err := conn.ListTagsForResource(tagsInput)

	if err != nil {
		return fmt.Errorf("error reading DataSync Task (%s) tags: %s", d.Id(), err)
	}

	d.Set("arn", output.TaskArn)
	d.Set("cloudwatch_log_group_arn", output.CloudWatchLogGroupArn)
	d.Set("destination_location_arn", output.DestinationLocationArn)
	d.Set("name", output.Name)

	if err := d.Set("options", flattenDataSyncOptions(output.Options)); err != nil {
		return fmt.Errorf("error setting options: %s", err)
	}

	d.Set("source_location_arn", output.SourceLocationArn)

	if err := d.Set("tags", flattenDataSyncTagListEntry(tagsOutput.Tags)); err != nil {
		return fmt.Errorf("error setting tags: %s", err)
	}

	return nil
}

func resourceAwsDataSyncTaskUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datasyncconn

	if d.HasChange("options") || d.HasChange("name") {
		input := &datasync.UpdateTaskInput{
			Options: expandDataSyncOptions(d.Get("options").([]interface{})),
			Name:    aws.String(d.Get("name").(string)),
			TaskArn: aws.String(d.Id()),
		}

		log.Printf("[DEBUG] Updating DataSync Task: %s", input)
		if _, err := conn.UpdateTask(input); err != nil {
			return fmt.Errorf("error creating DataSync Task: %s", err)
		}
	}

	if d.HasChange("tags") {
		oldRaw, newRaw := d.GetChange("tags")
		createTags, removeTags := dataSyncTagsDiff(expandDataSyncTagListEntry(oldRaw.(map[string]interface{})), expandDataSyncTagListEntry(newRaw.(map[string]interface{})))

		if len(removeTags) > 0 {
			input := &datasync.UntagResourceInput{
				Keys:        dataSyncTagsKeys(removeTags),
				ResourceArn: aws.String(d.Id()),
			}

			log.Printf("[DEBUG] Untagging DataSync Task: %s", input)
			if _, err := conn.UntagResource(input); err != nil {
				return fmt.Errorf("error untagging DataSync Task (%s): %s", d.Id(), err)
			}
		}

		if len(createTags) > 0 {
			input := &datasync.TagResourceInput{
				ResourceArn: aws.String(d.Id()),
				Tags:        createTags,
			}

			log.Printf("[DEBUG] Tagging DataSync Task: %s", input)
			if _, err := conn.TagResource(input); err != nil {
				return fmt.Errorf("error tagging DataSync Task (%s): %s", d.Id(), err)
			}
		}
	}

	return resourceAwsDataSyncTaskRead(d, meta)
}

func resourceAwsDataSyncTaskDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).datasyncconn

	input := &datasync.DeleteTaskInput{
		TaskArn: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting DataSync Task: %s", input)
	_, err := conn.DeleteTask(input)

	if isAWSErr(err, "InvalidRequestException", "not found") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting DataSync Task (%s): %s", d.Id(), err)
	}

	return nil
}
