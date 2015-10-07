package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/schema"
)

// setTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func setTagsRDS(conn *rds.RDS, d *schema.ResourceData, arn string) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffTagsRDS(tagsFromMapRDS(o), tagsFromMapRDS(n))

		// Set tags
		if len(remove) > 0 {
			log.Printf("[DEBUG] Removing tags: %s", remove)
			k := make([]*string, len(remove), len(remove))
			for i, t := range remove {
				k[i] = t.Key
			}

			_, err := conn.RemoveTagsFromResource(&rds.RemoveTagsFromResourceInput{
				ResourceName: aws.String(arn),
				TagKeys:      k,
			})
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			log.Printf("[DEBUG] Creating tags: %s", create)
			_, err := conn.AddTagsToResource(&rds.AddTagsToResourceInput{
				ResourceName: aws.String(arn),
				Tags:         create,
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
func diffTagsRDS(oldTags, newTags []*rds.Tag) ([]*rds.Tag, []*rds.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []*rds.Tag
	for _, t := range oldTags {
		old, ok := create[*t.Key]
		if !ok || old != *t.Value {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return tagsFromMapRDS(create), remove
}

// tagsFromMap returns the tags for the given map of data.
func tagsFromMapRDS(m map[string]interface{}) []*rds.Tag {
	result := make([]*rds.Tag, 0, len(m))
	for k, v := range m {
		result = append(result, &rds.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		})
	}

	return result
}

// tagsToMap turns the list of tags into a map.
func tagsToMapRDS(ts []*rds.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		result[*t.Key] = *t.Value
	}

	return result
}

func saveTagsRDS(conn *rds.RDS, d *schema.ResourceData, arn string) error {
	resp, err := conn.ListTagsForResource(&rds.ListTagsForResourceInput{
		ResourceName: aws.String(arn),
	})

	if err != nil {
		return fmt.Errorf("[DEBUG] Error retreiving tags for ARN: %s", arn)
	}

	var dt []*rds.Tag
	if len(resp.TagList) > 0 {
		dt = resp.TagList
	}

	return d.Set("tags", tagsToMapRDS(dt))
}
