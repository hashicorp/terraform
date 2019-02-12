package aws

import (
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ram"
)

// diffTags takes our tags locally and the ones remotely and returns
// the set of tags that must be created, and the set of tags that must
// be destroyed.
func diffTagsRAM(oldTags, newTags []*ram.Tag) ([]*ram.Tag, []*ram.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
	}

	// Build the list of what to remove
	var remove []*ram.Tag
	for _, t := range oldTags {
		old, ok := create[aws.StringValue(t.Key)]
		if !ok || old != aws.StringValue(t.Value) {
			// Delete it!
			remove = append(remove, t)
		} else if ok {
			delete(create, aws.StringValue(t.Key))
		}
	}

	return tagsFromMapRAM(create), remove
}

// tagsFromMapRAM returns the tags for the given map of data for RAM.
func tagsFromMapRAM(m map[string]interface{}) []*ram.Tag {
	result := make([]*ram.Tag, 0, len(m))
	for k, v := range m {
		t := &ram.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		}
		if !tagIgnoredRAM(t) {
			result = append(result, t)
		}
	}

	return result
}

// tagsToMapRAM turns the list of RAM tags into a map.
func tagsToMapRAM(ts []*ram.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		if !tagIgnoredRAM(t) {
			result[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
		}
	}

	return result
}

// compare a tag against a list of strings and checks if it should
// be ignored or not
func tagIgnoredRAM(t *ram.Tag) bool {
	filter := []string{"^aws:"}
	for _, v := range filter {
		log.Printf("[DEBUG] Matching %v with %v\n", v, *t.Key)
		if r, _ := regexp.MatchString(v, *t.Key); r {
			log.Printf("[DEBUG] Found AWS specific tag %s (val: %s), ignoring.\n", *t.Key, *t.Value)
			return true
		}
	}
	return false
}

// tagKeysRam returns the keys for the list of RAM tags
func tagKeysRam(ts []*ram.Tag) []*string {
	result := make([]*string, 0, len(ts))
	for _, t := range ts {
		result = append(result, t.Key)
	}
	return result
}
