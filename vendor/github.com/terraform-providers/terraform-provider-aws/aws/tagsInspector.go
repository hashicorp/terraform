package aws

import (
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/inspector"
)

// diffTags takes our tags locally and the ones remotely and returns
// the set of tags that must be created, and the set of tags that must
// be destroyed.
func diffTagsInspector(oldTags, newTags []*inspector.ResourceGroupTag) ([]*inspector.ResourceGroupTag, []*inspector.ResourceGroupTag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []*inspector.ResourceGroupTag
	for _, t := range oldTags {
		old, ok := create[*t.Key]
		if !ok || old != *t.Value {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return tagsFromMapInspector(create), remove
}

// tagsFromMap returns the tags for the given map of data.
func tagsFromMapInspector(m map[string]interface{}) []*inspector.ResourceGroupTag {
	var result []*inspector.ResourceGroupTag
	for k, v := range m {
		t := &inspector.ResourceGroupTag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		}
		if !tagIgnoredInspector(t) {
			result = append(result, t)
		}
	}

	return result
}

// tagsToMap turns the list of tags into a map.
func tagsToMapInspector(ts []*inspector.ResourceGroupTag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		if !tagIgnoredInspector(t) {
			result[*t.Key] = *t.Value
		}
	}

	return result
}

// compare a tag against a list of strings and checks if it should
// be ignored or not
func tagIgnoredInspector(t *inspector.ResourceGroupTag) bool {
	filter := []string{"^aws:"}
	for _, v := range filter {
		log.Printf("[DEBUG] Matching %v with %v\n", v, *t.Key)
		if r, _ := regexp.MatchString(v, *t.Key); r == true {
			log.Printf("[DEBUG] Found AWS specific tag %s (val: %s), ignoring.\n", *t.Key, *t.Value)
			return true
		}
	}
	return false
}
