package aws

import (
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sfn"
)

// diffTags takes our tags locally and the ones remotely and returns
// the set of tags that must be created, and the set of tags that must
// be destroyed.
func diffTagsSfn(oldTags, newTags []*sfn.Tag) ([]*sfn.Tag, []*sfn.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
	}

	// Build the list of what to remove
	var remove []*sfn.Tag
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

	return tagsFromMapSfn(create), remove
}

// tagsFromMap returns the tags for the given map of data.
func tagsFromMapSfn(tagMap map[string]interface{}) []*sfn.Tag {
	tags := make([]*sfn.Tag, 0, len(tagMap))
	for tagKey, tagValueRaw := range tagMap {
		tag := &sfn.Tag{
			Key:   aws.String(tagKey),
			Value: aws.String(tagValueRaw.(string)),
		}
		if !tagIgnoredSfn(tag) {
			tags = append(tags, tag)
		}
	}

	return tags
}

// tagsToMap turns the list of tags into a map.
func tagsToMapSfn(tags []*sfn.Tag) map[string]string {
	tagMap := make(map[string]string)
	for _, tag := range tags {
		if !tagIgnoredSfn(tag) {
			tagMap[aws.StringValue(tag.Key)] = aws.StringValue(tag.Value)
		}
	}

	return tagMap
}

// compare a tag against a list of strings and checks if it should
// be ignored or not
func tagIgnoredSfn(t *sfn.Tag) bool {
	filter := []string{"^aws:"}
	for _, v := range filter {
		log.Printf("[DEBUG] Matching %v with %v\n", v, aws.StringValue(t.Key))
		r, _ := regexp.MatchString(v, aws.StringValue(t.Key))
		if r {
			log.Printf("[DEBUG] Found AWS specific tag %s (val: %s), ignoring.\n", aws.StringValue(t.Key), aws.StringValue(t.Value))
			return true
		}
	}
	return false
}
