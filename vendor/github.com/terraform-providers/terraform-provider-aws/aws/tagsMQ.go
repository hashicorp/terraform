package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/mq"
	"github.com/hashicorp/terraform/helper/schema"
)

// getTags is a helper to get the tags for a resource. It expects the
// tags field to be named "tags"
func getTagsMQ(conn *mq.MQ, d *schema.ResourceData, arn string) error {
	resp, err := conn.ListTags(&mq.ListTagsInput{
		ResourceArn: aws.String(arn),
	})
	if err != nil {
		return err
	}

	if err := d.Set("tags", tagsToMapGeneric(resp.Tags)); err != nil {
		return err
	}

	return nil
}

// setTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func setTagsMQ(conn *mq.MQ, d *schema.ResourceData, arn string) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffTagsGeneric(o, n)

		// Set tags
		if len(remove) > 0 {
			log.Printf("[DEBUG] Removing tags: %#v", remove)
			keys := make([]*string, 0, len(remove))
			for k := range remove {
				keys = append(keys, aws.String(k))
			}

			_, err := conn.DeleteTags(&mq.DeleteTagsInput{
				ResourceArn: aws.String(arn),
				TagKeys:     keys,
			})
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			log.Printf("[DEBUG] Creating tags: %#v", create)

			_, err := conn.CreateTags(&mq.CreateTagsInput{
				ResourceArn: aws.String(arn),
				Tags:        create,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
