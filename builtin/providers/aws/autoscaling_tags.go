package aws

import (
	"bytes"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

// tagsSchema returns the schema to use for tags.
func autoscalingTagsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"key": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},

				"value": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},

				"propagate_at_launch": &schema.Schema{
					Type:     schema.TypeBool,
					Required: true,
				},
			},
		},
		Set: autoscalingTagsToHash,
	}
}

func autoscalingTagsToHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["key"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["value"].(string)))
	buf.WriteString(fmt.Sprintf("%t-", m["propagate_at_launch"].(bool)))

	return hashcode.String(buf.String())
}

// setTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tag"
func setAutoscalingTags(conn *autoscaling.AutoScaling, d *schema.ResourceData) error {
	if d.HasChange("tag") {
		oraw, nraw := d.GetChange("tag")
		o := setToMapByKey(oraw.(*schema.Set), "key")
		n := setToMapByKey(nraw.(*schema.Set), "key")

		resourceID := d.Get("name").(string)
		c, r := diffAutoscalingTags(
			autoscalingTagsFromMap(o, resourceID),
			autoscalingTagsFromMap(n, resourceID),
			resourceID)
		create := autoscaling.CreateOrUpdateTagsInput{
			Tags: c,
		}
		remove := autoscaling.DeleteTagsInput{
			Tags: r,
		}

		// Set tags
		if len(r) > 0 {
			log.Printf("[DEBUG] Removing autoscaling tags: %#v", r)
			if _, err := conn.DeleteTags(&remove); err != nil {
				return err
			}
		}
		if len(c) > 0 {
			log.Printf("[DEBUG] Creating autoscaling tags: %#v", c)
			if _, err := conn.CreateOrUpdateTags(&create); err != nil {
				return err
			}
		}
	}

	return nil
}

// diffTags takes our tags locally and the ones remotely and returns
// the set of tags that must be created, and the set of tags that must
// be destroyed.
func diffAutoscalingTags(oldTags, newTags []*autoscaling.Tag, resourceID string) ([]*autoscaling.Tag, []*autoscaling.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		tag := map[string]interface{}{
			"value":               *t.Value,
			"propagate_at_launch": *t.PropagateAtLaunch,
		}
		create[*t.Key] = tag
	}

	// Build the list of what to remove
	var remove []*autoscaling.Tag
	for _, t := range oldTags {
		old, ok := create[*t.Key].(map[string]interface{})

		if !ok || old["value"] != *t.Value || old["propagate_at_launch"] != *t.PropagateAtLaunch {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return autoscalingTagsFromMap(create, resourceID), remove
}

// tagsFromMap returns the tags for the given map of data.
func autoscalingTagsFromMap(m map[string]interface{}, resourceID string) []*autoscaling.Tag {
	result := make([]*autoscaling.Tag, 0, len(m))
	for k, v := range m {
		attr := v.(map[string]interface{})
		result = append(result, &autoscaling.Tag{
			Key:               aws.String(k),
			Value:             aws.String(attr["value"].(string)),
			PropagateAtLaunch: aws.Bool(attr["propagate_at_launch"].(bool)),
			ResourceId:        aws.String(resourceID),
			ResourceType:      aws.String("auto-scaling-group"),
		})
	}

	return result
}

// autoscalingTagsToMap turns the list of tags into a map.
func autoscalingTagsToMap(ts []*autoscaling.Tag) map[string]interface{} {
	tags := make(map[string]interface{})
	for _, t := range ts {
		tag := map[string]interface{}{
			"value":               *t.Value,
			"propagate_at_launch": *t.PropagateAtLaunch,
		}
		tags[*t.Key] = tag
	}

	return tags
}

// autoscalingTagDescriptionsToMap turns the list of tags into a map.
func autoscalingTagDescriptionsToMap(ts *[]*autoscaling.TagDescription) map[string]map[string]interface{} {
	tags := make(map[string]map[string]interface{})
	for _, t := range *ts {
		tag := map[string]interface{}{
			"value":               *t.Value,
			"propagate_at_launch": *t.PropagateAtLaunch,
		}
		tags[*t.Key] = tag
	}

	return tags
}

// autoscalingTagDescriptionsToSlice turns the list of tags into a slice.
func autoscalingTagDescriptionsToSlice(ts []*autoscaling.TagDescription) []map[string]interface{} {
	tags := make([]map[string]interface{}, 0, len(ts))
	for _, t := range ts {
		tags = append(tags, map[string]interface{}{
			"key":                 *t.Key,
			"value":               *t.Value,
			"propagate_at_launch": *t.PropagateAtLaunch,
		})
	}

	return tags
}

func setToMapByKey(s *schema.Set, key string) map[string]interface{} {
	result := make(map[string]interface{})
	for _, rawData := range s.List() {
		data := rawData.(map[string]interface{})
		result[data[key].(string)] = data
	}

	return result
}
