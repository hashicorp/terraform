package aws

import (
	"log"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sagemaker"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func tagsFromMapSagemaker(m map[string]interface{}) []*sagemaker.Tag {
	result := make([]*sagemaker.Tag, 0, len(m))
	for k, v := range m {
		t := &sagemaker.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		}
		if !tagIgnoredSagemaker(t) {
			result = append(result, t)
		}
	}

	return result
}

func tagsToMapSagemaker(ts []*sagemaker.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		if !tagIgnoredSagemaker(t) {
			result[*t.Key] = *t.Value
		}
	}

	return result
}

func setSagemakerTags(conn *sagemaker.SageMaker, d *schema.ResourceData) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffSagemakerTags(tagsFromMapSagemaker(o), tagsFromMapSagemaker(n))

		if len(remove) > 0 {
			err := resource.Retry(5*time.Minute, func() *resource.RetryError {
				log.Printf("[DEBUG] Removing tags: %#v from %s", remove, d.Id())
				_, err := conn.DeleteTags(&sagemaker.DeleteTagsInput{
					ResourceArn: aws.String(d.Get("arn").(string)),
					TagKeys:     remove,
				})
				if err != nil {
					sagemakerErr, ok := err.(awserr.Error)
					if ok && sagemakerErr.Code() == "ResourceNotFound" {
						return resource.RetryableError(err)
					}
					return resource.NonRetryableError(err)
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			err := resource.Retry(5*time.Minute, func() *resource.RetryError {
				log.Printf("[DEBUG] Creating tags: %s for %s", create, d.Id())
				_, err := conn.AddTags(&sagemaker.AddTagsInput{
					ResourceArn: aws.String(d.Get("arn").(string)),
					Tags:        create,
				})
				if err != nil {
					sagemakerErr, ok := err.(awserr.Error)
					if ok && sagemakerErr.Code() == "ResourceNotFound" {
						return resource.RetryableError(err)
					}
					return resource.NonRetryableError(err)
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func diffSagemakerTags(oldTags, newTags []*sagemaker.Tag) ([]*sagemaker.Tag, []*string) {
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	var remove []*string
	for _, t := range oldTags {
		old, ok := create[*t.Key]
		if !ok || old != *t.Value {
			remove = append(remove, t.Key)
		}
	}

	return tagsFromMapSagemaker(create), remove
}

func tagIgnoredSagemaker(t *sagemaker.Tag) bool {
	filter := []string{"^aws:"}
	for _, v := range filter {
		log.Printf("[DEBUG] Matching %v with %v\n", v, *t.Key)
		if r, _ := regexp.MatchString(v, *t.Key); r {
			log.Printf("[DEBUG] Found AWS specific tag %s (val: %s), ignoring.\n", *t.Key, *t.Value)
			return true
		}
	}
	return false
}
