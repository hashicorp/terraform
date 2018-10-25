package aws

import (
	"fmt"
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/neptune"
	"github.com/hashicorp/terraform/helper/schema"
)

// setTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func setTagsNeptune(conn *neptune.Neptune, d *schema.ResourceData, arn string) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffTagsNeptune(tagsFromMapNeptune(o), tagsFromMapNeptune(n))

		// Set tags
		if len(remove) > 0 {
			log.Printf("[DEBUG] Removing tags: %s", remove)
			k := make([]*string, len(remove), len(remove))
			for i, t := range remove {
				k[i] = t.Key
			}

			_, err := conn.RemoveTagsFromResource(&neptune.RemoveTagsFromResourceInput{
				ResourceName: aws.String(arn),
				TagKeys:      k,
			})
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			log.Printf("[DEBUG] Creating tags: %s", create)
			_, err := conn.AddTagsToResource(&neptune.AddTagsToResourceInput{
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
func diffTagsNeptune(oldTags, newTags []*neptune.Tag) ([]*neptune.Tag, []*neptune.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
	}

	// Build the list of what to remove
	var remove []*neptune.Tag
	for _, t := range oldTags {
		old, ok := create[aws.StringValue(t.Key)]
		if !ok || old != aws.StringValue(t.Value) {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return tagsFromMapNeptune(create), remove
}

// tagsFromMap returns the tags for the given map of data.
func tagsFromMapNeptune(m map[string]interface{}) []*neptune.Tag {
	result := make([]*neptune.Tag, 0, len(m))
	for k, v := range m {
		t := &neptune.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		}
		if !tagIgnoredNeptune(t) {
			result = append(result, t)
		}
	}

	return result
}

// tagsToMap turns the list of tags into a map.
func tagsToMapNeptune(ts []*neptune.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		if !tagIgnoredNeptune(t) {
			result[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
		}
	}

	return result
}

// compare a tag against a list of strings and checks if it should
// be ignored or not
func tagIgnoredNeptune(t *neptune.Tag) bool {
	filter := []string{"^aws:"}
	for _, v := range filter {
		log.Printf("[DEBUG] Matching %v with %v\n", v, aws.StringValue(t.Key))
		if r, _ := regexp.MatchString(v, aws.StringValue(t.Key)); r == true {
			log.Printf("[DEBUG] Found AWS specific tag %s (val: %s), ignoring.\n", aws.StringValue(t.Key), aws.StringValue(t.Value))
			return true
		}
	}
	return false
}

func saveTagsNeptune(conn *neptune.Neptune, d *schema.ResourceData, arn string) error {
	resp, err := conn.ListTagsForResource(&neptune.ListTagsForResourceInput{
		ResourceName: aws.String(arn),
	})

	if err != nil {
		return fmt.Errorf("Error retreiving tags for ARN: %s", arn)
	}

	var dt []*neptune.Tag
	if len(resp.TagList) > 0 {
		dt = resp.TagList
	}

	return d.Set("tags", tagsToMapNeptune(dt))
}
