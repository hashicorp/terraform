package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsSfnActivity() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSfnActivityCreate,
		Read:   resourceAwsSfnActivityRead,
		Update: resourceAwsSfnActivityUpdate,
		Delete: resourceAwsSfnActivityDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(0, 80),
			},

			"creation_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsSfnActivityCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sfnconn
	log.Print("[DEBUG] Creating Step Function Activity")

	params := &sfn.CreateActivityInput{
		Name: aws.String(d.Get("name").(string)),
	}

	activity, err := conn.CreateActivity(params)
	if err != nil {
		return fmt.Errorf("Error creating Step Function Activity: %s", err)
	}

	d.SetId(*activity.ActivityArn)

	if v, ok := d.GetOk("tags"); ok {
		input := &sfn.TagResourceInput{
			ResourceArn: aws.String(d.Id()),
			Tags:        tagsFromMapSfn(v.(map[string]interface{})),
		}
		log.Printf("[DEBUG] Tagging SFN Activity: %s", input)
		_, err := conn.TagResource(input)
		if err != nil {
			return fmt.Errorf("error tagging SFN Activity (%s): %s", d.Id(), input)
		}
	}
	return resourceAwsSfnActivityRead(d, meta)
}

func resourceAwsSfnActivityUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sfnconn

	if d.HasChange("tags") {
		oldTagsRaw, newTagsRaw := d.GetChange("tags")
		oldTagsMap := oldTagsRaw.(map[string]interface{})
		newTagsMap := newTagsRaw.(map[string]interface{})
		createTags, removeTags := diffTagsSfn(tagsFromMapSfn(oldTagsMap), tagsFromMapSfn(newTagsMap))

		if len(removeTags) > 0 {
			removeTagKeys := make([]*string, len(removeTags))
			for i, removeTag := range removeTags {
				removeTagKeys[i] = removeTag.Key
			}

			input := &sfn.UntagResourceInput{
				ResourceArn: aws.String(d.Id()),
				TagKeys:     removeTagKeys,
			}

			log.Printf("[DEBUG] Untagging State Function Activity: %s", input)
			if _, err := conn.UntagResource(input); err != nil {
				return fmt.Errorf("error untagging State Function Activity (%s): %s", d.Id(), err)
			}
		}

		if len(createTags) > 0 {
			input := &sfn.TagResourceInput{
				ResourceArn: aws.String(d.Id()),
				Tags:        createTags,
			}

			log.Printf("[DEBUG] Tagging State Function Activity: %s", input)
			if _, err := conn.TagResource(input); err != nil {
				return fmt.Errorf("error tagging State Function Activity (%s): %s", d.Id(), err)
			}
		}
	}

	return resourceAwsSfnActivityRead(d, meta)
}

func resourceAwsSfnActivityRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sfnconn
	log.Printf("[DEBUG] Reading Step Function Activity: %s", d.Id())

	sm, err := conn.DescribeActivity(&sfn.DescribeActivityInput{
		ActivityArn: aws.String(d.Id()),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "ActivityDoesNotExist" {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", sm.Name)

	if err := d.Set("creation_date", sm.CreationDate.Format(time.RFC3339)); err != nil {
		log.Printf("[DEBUG] Error setting creation_date: %s", err)
	}

	tagsResp, err := conn.ListTagsForResource(
		&sfn.ListTagsForResourceInput{
			ResourceArn: aws.String(d.Id()),
		},
	)

	if err != nil {
		return fmt.Errorf("error listing SFN Activity (%s) tags: %s", d.Id(), err)
	}

	if err := d.Set("tags", tagsToMapSfn(tagsResp.Tags)); err != nil {
		return fmt.Errorf("error setting tags: %s", err)
	}

	return nil
}

func resourceAwsSfnActivityDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sfnconn
	log.Printf("[DEBUG] Deleting Step Functions Activity: %s", d.Id())

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteActivity(&sfn.DeleteActivityInput{
			ActivityArn: aws.String(d.Id()),
		})

		if err == nil {
			return nil
		}

		return resource.NonRetryableError(err)
	})
}
