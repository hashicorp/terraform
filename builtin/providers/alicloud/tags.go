package alicloud

import (
	"fmt"
	"log"

	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
)

func String(v string) *string {
	return &v
}

// tagsSchema returns the schema to use for tags.
//
func tagsSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeMap,
		//Elem:     &schema.Schema{Type: schema.TypeString},
		Optional: true,
	}
}

// setTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func setTags(client *AliyunClient, resourceType ecs.TagResourceType, d *schema.ResourceData) error {

	conn := client.ecsconn

	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffTags(tagsFromMap(o), tagsFromMap(n))

		// Set tags
		if len(remove) > 0 {
			log.Printf("[DEBUG] Removing tags: %#v from %s", remove, d.Id())
			err := RemoveTags(conn, &RemoveTagsArgs{
				RegionId:     client.Region,
				ResourceId:   d.Id(),
				ResourceType: resourceType,
				Tag:          remove,
			})
			if err != nil {
				return fmt.Errorf("Remove tags got error: %s", err)
			}
		}

		if len(create) > 0 {
			log.Printf("[DEBUG] Creating tags: %s for %s", create, d.Id())
			err := AddTags(conn, &AddTagsArgs{
				RegionId:     client.Region,
				ResourceId:   d.Id(),
				ResourceType: resourceType,
				Tag:          create,
			})
			if err != nil {
				return fmt.Errorf("Creating tags got error: %s", err)
			}
		}
	}

	return nil
}

// diffTags takes our tags locally and the ones remotely and returns
// the set of tags that must be created, and the set of tags that must
// be destroyed.
func diffTags(oldTags, newTags []Tag) ([]Tag, []Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[t.Key] = t.Value
	}

	// Build the list of what to remove
	var remove []Tag
	for _, t := range oldTags {
		old, ok := create[t.Key]
		if !ok || old != t.Value {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return tagsFromMap(create), remove
}

// tagsFromMap returns the tags for the given map of data.
func tagsFromMap(m map[string]interface{}) []Tag {
	result := make([]Tag, 0, len(m))
	for k, v := range m {
		result = append(result, Tag{
			Key:   k,
			Value: v.(string),
		})
	}

	return result
}

func tagsToMap(tags []ecs.TagItemType) map[string]string {
	result := make(map[string]string)
	for _, t := range tags {
		result[t.TagKey] = t.TagValue
	}

	return result
}

func tagsToString(tags []ecs.TagItemType) string {
	result := make([]string, 0, len(tags))

	for _, tag := range tags {
		ecsTags := ecs.TagItemType{
			TagKey:   tag.TagKey,
			TagValue: tag.TagValue,
		}
		result = append(result, ecsTags.TagKey+":"+ecsTags.TagValue)
	}

	return strings.Join(result, ",")
}
