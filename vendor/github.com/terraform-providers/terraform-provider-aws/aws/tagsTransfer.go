package aws

import (
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/transfer"
	"github.com/hashicorp/terraform/helper/schema"
)

// setTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func setTagsTransfer(conn *transfer.Transfer, d *schema.ResourceData) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffTagsTransfer(tagsFromMapTransfer(o), tagsFromMapTransfer(n))

		// Set tags
		if len(remove) > 0 {
			log.Printf("[DEBUG] Removing tags: %#v", remove)
			k := make([]*string, len(remove))
			for i, t := range remove {
				k[i] = t.Key
			}

			_, err := conn.UntagResource(&transfer.UntagResourceInput{
				Arn:     aws.String(d.Get("arn").(string)),
				TagKeys: k,
			})
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			log.Printf("[DEBUG] Creating tags: %#v", create)
			_, err := conn.TagResource(&transfer.TagResourceInput{
				Arn:  aws.String(d.Get("arn").(string)),
				Tags: create,
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
func diffTagsTransfer(oldTags, newTags []*transfer.Tag) ([]*transfer.Tag, []*transfer.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
	}

	// Build the list of what to remove
	var remove []*transfer.Tag
	for _, t := range oldTags {
		old, ok := create[aws.StringValue(t.Key)]
		if !ok || old != aws.StringValue(t.Value) {
			// Delete it!
			remove = append(remove, t)
		} else if ok {
			// already present so remove from new
			delete(create, aws.StringValue(t.Key))
		}
	}

	return tagsFromMapTransfer(create), remove
}

// tagsFromMap returns the tags for the given map of data.
func tagsFromMapTransfer(m map[string]interface{}) []*transfer.Tag {
	result := make([]*transfer.Tag, 0, len(m))
	for k, v := range m {
		t := &transfer.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		}
		if !tagIgnoredTransfer(t) {
			result = append(result, t)
		}
	}

	return result
}

// tagsToMap turns the list of tags into a map.
func tagsToMapTransfer(ts []*transfer.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		if !tagIgnoredTransfer(t) {
			result[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
		}
	}

	return result
}

// compare a tag against a list of strings and checks if it should
// be ignored or not
func tagIgnoredTransfer(t *transfer.Tag) bool {
	filter := []string{"^aws:"}
	for _, v := range filter {
		log.Printf("[DEBUG] Matching %v with %v\n", v, aws.StringValue(t.Key))
		if r, _ := regexp.MatchString(v, aws.StringValue(t.Key)); r {
			log.Printf("[DEBUG] Found AWS specific tag %s (val: %s), ignoring.\n", aws.StringValue(t.Key), aws.StringValue(t.Value))
			return true
		}
	}
	return false
}
