package aws

import (
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
)

// diffTags takes our tags locally and the ones remotely and returns
// the set of tags that must be created, and the set of tags that must
// be destroyed.
func diffTagsGeneric(oldTags, newTags map[string]interface{}) (map[string]*string, map[string]*string) {
	// First, we're creating everything we have
	create := make(map[string]*string)
	for k, v := range newTags {
		create[k] = aws.String(v.(string))
	}

	// Build the map of what to remove
	remove := make(map[string]*string)
	for k, v := range oldTags {
		old, ok := create[k]
		if !ok || old != aws.String(v.(string)) {
			// Delete it!
			remove[k] = aws.String(v.(string))
		}
	}

	return create, remove
}

// tagsFromMap returns the tags for the given map of data.
func tagsFromMapGeneric(m map[string]interface{}) map[string]*string {
	result := make(map[string]*string)
	for k, v := range m {
		if !tagIgnoredGeneric(k) {
			result[k] = aws.String(v.(string))
		}
	}

	return result
}

// tagsToMap turns the tags into a map.
func tagsToMapGeneric(ts map[string]*string) map[string]string {
	result := make(map[string]string)
	for k, v := range ts {
		if !tagIgnoredGeneric(k) {
			result[k] = aws.StringValue(v)
		}
	}

	return result
}

// compare a tag against a list of strings and checks if it should
// be ignored or not
func tagIgnoredGeneric(k string) bool {
	filter := []string{"^aws:"}
	for _, v := range filter {
		log.Printf("[DEBUG] Matching %v with %v\n", v, k)
		if r, _ := regexp.MatchString(v, k); r == true {
			log.Printf("[DEBUG] Found AWS specific tag %s, ignoring.\n", k)
			return true
		}
	}
	return false
}
