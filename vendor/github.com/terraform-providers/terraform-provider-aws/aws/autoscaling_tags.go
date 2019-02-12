package aws

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

// autoscalingTagSchema returns the schema to use for the tag element.
func autoscalingTagSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"key": {
					Type:     schema.TypeString,
					Required: true,
				},

				"value": {
					Type:     schema.TypeString,
					Required: true,
				},

				"propagate_at_launch": {
					Type:     schema.TypeBool,
					Required: true,
				},
			},
		},
		Set: autoscalingTagToHash,
	}
}

func autoscalingTagToHash(v interface{}) int {
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
	resourceID := d.Get("name").(string)
	var createTags, removeTags []*autoscaling.Tag

	if d.HasChange("tag") || d.HasChange("tags") {
		oraw, nraw := d.GetChange("tag")
		o := setToMapByKey(oraw.(*schema.Set))
		n := setToMapByKey(nraw.(*schema.Set))

		old, err := autoscalingTagsFromMap(o, resourceID)
		if err != nil {
			return err
		}

		new, err := autoscalingTagsFromMap(n, resourceID)
		if err != nil {
			return err
		}

		c, r, err := diffAutoscalingTags(old, new, resourceID)
		if err != nil {
			return err
		}

		createTags = append(createTags, c...)
		removeTags = append(removeTags, r...)

		oraw, nraw = d.GetChange("tags")
		old, err = autoscalingTagsFromList(oraw.([]interface{}), resourceID)
		if err != nil {
			return err
		}

		new, err = autoscalingTagsFromList(nraw.([]interface{}), resourceID)
		if err != nil {
			return err
		}

		c, r, err = diffAutoscalingTags(old, new, resourceID)
		if err != nil {
			return err
		}

		createTags = append(createTags, c...)
		removeTags = append(removeTags, r...)
	}

	// Set tags
	if len(removeTags) > 0 {
		log.Printf("[DEBUG] Removing autoscaling tags: %#v", removeTags)

		remove := autoscaling.DeleteTagsInput{
			Tags: removeTags,
		}

		if _, err := conn.DeleteTags(&remove); err != nil {
			return err
		}
	}

	if len(createTags) > 0 {
		log.Printf("[DEBUG] Creating autoscaling tags: %#v", createTags)

		create := autoscaling.CreateOrUpdateTagsInput{
			Tags: createTags,
		}

		if _, err := conn.CreateOrUpdateTags(&create); err != nil {
			return err
		}
	}

	return nil
}

// diffTags takes our tags locally and the ones remotely and returns
// the set of tags that must be created, and the set of tags that must
// be destroyed.
func diffAutoscalingTags(oldTags, newTags []*autoscaling.Tag, resourceID string) ([]*autoscaling.Tag, []*autoscaling.Tag, error) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		tag := map[string]interface{}{
			"key":                 *t.Key,
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

	createTags, err := autoscalingTagsFromMap(create, resourceID)
	if err != nil {
		return nil, nil, err
	}

	return createTags, remove, nil
}

func autoscalingTagsFromList(vs []interface{}, resourceID string) ([]*autoscaling.Tag, error) {
	result := make([]*autoscaling.Tag, 0, len(vs))
	for _, tag := range vs {
		attr, ok := tag.(map[string]interface{})
		if !ok {
			continue
		}

		t, err := autoscalingTagFromMap(attr, resourceID)
		if err != nil {
			return nil, err
		}

		if t != nil {
			result = append(result, t)
		}
	}
	return result, nil
}

// tagsFromMap returns the tags for the given map of data.
func autoscalingTagsFromMap(m map[string]interface{}, resourceID string) ([]*autoscaling.Tag, error) {
	result := make([]*autoscaling.Tag, 0, len(m))
	for _, v := range m {
		attr, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		t, err := autoscalingTagFromMap(attr, resourceID)
		if err != nil {
			return nil, err
		}

		if t != nil {
			result = append(result, t)
		}
	}

	return result, nil
}

func autoscalingTagFromMap(attr map[string]interface{}, resourceID string) (*autoscaling.Tag, error) {
	if _, ok := attr["key"]; !ok {
		return nil, fmt.Errorf("%s: invalid tag attributes: key missing", resourceID)
	}

	if _, ok := attr["value"]; !ok {
		return nil, fmt.Errorf("%s: invalid tag attributes: value missing", resourceID)
	}

	if _, ok := attr["propagate_at_launch"]; !ok {
		return nil, fmt.Errorf("%s: invalid tag attributes: propagate_at_launch missing", resourceID)
	}

	var propagateAtLaunch bool
	var err error

	if v, ok := attr["propagate_at_launch"].(bool); ok {
		propagateAtLaunch = v
	}

	if v, ok := attr["propagate_at_launch"].(string); ok {
		if propagateAtLaunch, err = strconv.ParseBool(v); err != nil {
			return nil, fmt.Errorf(
				"%s: invalid tag attribute: invalid value for propagate_at_launch: %s",
				resourceID,
				v,
			)
		}
	}

	t := &autoscaling.Tag{
		Key:               aws.String(attr["key"].(string)),
		Value:             aws.String(attr["value"].(string)),
		PropagateAtLaunch: aws.Bool(propagateAtLaunch),
		ResourceId:        aws.String(resourceID),
		ResourceType:      aws.String("auto-scaling-group"),
	}

	if tagIgnoredAutoscaling(t) {
		return nil, nil
	}

	return t, nil
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

func setToMapByKey(s *schema.Set) map[string]interface{} {
	result := make(map[string]interface{})
	for _, rawData := range s.List() {
		data := rawData.(map[string]interface{})
		result[data["key"].(string)] = data
	}

	return result
}

// compare a tag against a list of strings and checks if it should
// be ignored or not
func tagIgnoredAutoscaling(t *autoscaling.Tag) bool {
	filter := []string{"^aws:"}
	for _, v := range filter {
		log.Printf("[DEBUG] Matching %v with %v\n", v, *t.Key)
		r, _ := regexp.MatchString(v, *t.Key)
		if r {
			log.Printf("[DEBUG] Found AWS specific tag %s (val: %s), ignoring.\n", *t.Key, *t.Value)
			return true
		}
	}
	return false
}
