package aws

// TODO: Clint: consolidate tags and tags_sdk
// tags_sdk and tags_sdk_test are used only for transition to aws-sdk-go
// and will replace tags and tags_test when the transition to aws-sdk-go/ec2 is
// complete

import (
	"log"

	"github.com/hashicorp/aws-sdk-go/aws"
	"github.com/hashicorp/aws-sdk-go/gen/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

// tagsSchema returns the schema to use for tags.
//
// TODO: uncomment this when we replace the original tags.go
//
// func tagsSchema() *schema.Schema {
// 	return &schema.Schema{
// 		Type:     schema.TypeMap,
// 		Optional: true,
// 	}
// }

// setTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func setTagsSDK(conn *ec2.EC2, d *schema.ResourceData) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffTagsSDK(tagsFromMapSDK(o), tagsFromMapSDK(n))

		// Set tags
		if len(remove) > 0 {
			log.Printf("[DEBUG] Removing tags: %#v", remove)
			err := conn.DeleteTags(&ec2.DeleteTagsRequest{
				Resources: []string{d.Id()},
				Tags:      remove,
			})
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			log.Printf("[DEBUG] Creating tags: %#v", create)
			err := conn.CreateTags(&ec2.CreateTagsRequest{
				Resources: []string{d.Id()},
				Tags:      create,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// diffTags takes our tags locally and the ones remotely and returns
// the set of tags that must be created, and the set of tags that must
// be destroyed.
func diffTagsSDK(oldTags, newTags []ec2.Tag) ([]ec2.Tag, []ec2.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []ec2.Tag
	for _, t := range oldTags {
		old, ok := create[*t.Key]
		if !ok || old != *t.Value {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return tagsFromMapSDK(create), remove
}

// tagsFromMap returns the tags for the given map of data.
func tagsFromMapSDK(m map[string]interface{}) []ec2.Tag {
	result := make([]ec2.Tag, 0, len(m))
	for k, v := range m {
		result = append(result, ec2.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		})
	}

	return result
}

// tagsToMap turns the list of tags into a map.
func tagsToMapSDK(ts []ec2.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		result[*t.Key] = *t.Value
	}

	return result
}
