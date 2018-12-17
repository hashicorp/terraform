package aws

import (
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/inspector"
)

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
