package aws

import (
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
)

// diffTags takes our tags locally and the ones remotely and returns
// the set of tags that must be created, and the set of tags that must
// be destroyed.
func diffTagsBeanstalk(oldTags, newTags []*elasticbeanstalk.Tag) ([]*elasticbeanstalk.Tag, []*string) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []*string
	for _, t := range oldTags {
		if _, ok := create[*t.Key]; !ok {
			// Delete it!
			remove = append(remove, t.Key)
		}
	}

	return tagsFromMapBeanstalk(create), remove
}

// tagsFromMap returns the tags for the given map of data.
func tagsFromMapBeanstalk(m map[string]interface{}) []*elasticbeanstalk.Tag {
	var result []*elasticbeanstalk.Tag
	for k, v := range m {
		t := &elasticbeanstalk.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		}
		if !tagIgnoredBeanstalk(t) {
			result = append(result, t)
		}
	}

	return result
}

// tagsToMap turns the list of tags into a map.
func tagsToMapBeanstalk(ts []*elasticbeanstalk.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		if !tagIgnoredBeanstalk(t) {
			result[*t.Key] = *t.Value
		}
	}

	return result
}

// compare a tag against a list of strings and checks if it should
// be ignored or not
func tagIgnoredBeanstalk(t *elasticbeanstalk.Tag) bool {
	filter := []string{"^aws:", "^elasticbeanstalk:", "Name"}
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
