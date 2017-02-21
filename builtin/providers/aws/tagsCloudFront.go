package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/hashicorp/terraform/helper/schema"
)

func setTagsCloudFront(conn *cloudfront.CloudFront, d *schema.ResourceData, arn string) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffTagsCloudFront(tagsFromMapCloudFront(o), tagsFromMapCloudFront(n))

		if len(remove) > 0 {
			log.Printf("[DEBUG] Removing tags: %s", remove)
			k := make([]*string, 0, len(remove))
			for _, t := range remove {
				k = append(k, t.Key)
			}

			_, err := conn.UntagResource(&cloudfront.UntagResourceInput{
				Resource: aws.String(arn),
				TagKeys: &cloudfront.TagKeys{
					Items: k,
				},
			})
			if err != nil {
				return err
			}
		}

		if len(create) > 0 {
			log.Printf("[DEBUG] Creating tags: %s", create)
			_, err := conn.TagResource(&cloudfront.TagResourceInput{
				Resource: aws.String(arn),
				Tags: &cloudfront.Tags{
					Items: create,
				},
			})
			if err != nil {
				return err
			}
		}

	}

	return nil
}
func diffTagsCloudFront(oldTags, newTags *cloudfront.Tags) ([]*cloudfront.Tag, []*cloudfront.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags.Items {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []*cloudfront.Tag
	for _, t := range oldTags.Items {
		old, ok := create[*t.Key]
		if !ok || old != *t.Value {
			// Delete it!
			remove = append(remove, t)
		}
	}

	createTags := tagsFromMapCloudFront(create)
	return createTags.Items, remove
}

func tagsFromMapCloudFront(m map[string]interface{}) *cloudfront.Tags {
	result := make([]*cloudfront.Tag, 0, len(m))
	for k, v := range m {
		result = append(result, &cloudfront.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		})
	}

	tags := &cloudfront.Tags{
		Items: result,
	}

	return tags
}

func tagsToMapCloudFront(ts *cloudfront.Tags) map[string]string {
	result := make(map[string]string)

	for _, t := range ts.Items {
		result[*t.Key] = *t.Value
	}

	return result
}
