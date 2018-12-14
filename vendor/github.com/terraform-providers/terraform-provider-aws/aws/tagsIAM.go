package aws

import (
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
)

// diffTags takes our tags locally and the ones remotely and returns
// the set of tags that must be created, and the set of tags that must
// be destroyed.
func diffTagsIAM(oldTags, newTags []*iam.Tag) ([]*iam.Tag, []*iam.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
	}

	// Build the list of what to remove
	var remove []*iam.Tag
	for _, t := range oldTags {
		old, ok := create[aws.StringValue(t.Key)]
		if !ok || old != aws.StringValue(t.Value) {
			// Delete it!
			remove = append(remove, t)
		} else if ok {
			delete(create, aws.StringValue(t.Key))
		}
	}

	return tagsFromMapIAM(create), remove
}

// tagsFromMapIAM returns the tags for the given map of data for IAM.
func tagsFromMapIAM(m map[string]interface{}) []*iam.Tag {
	result := make([]*iam.Tag, 0, len(m))
	for k, v := range m {
		t := &iam.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		}
		if !tagIgnoredIAM(t) {
			result = append(result, t)
		}
	}

	return result
}

// tagsToMapIAM turns the list of IAM tags into a map.
func tagsToMapIAM(ts []*iam.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		if !tagIgnoredIAM(t) {
			result[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
		}
	}

	return result
}

// compare a tag against a list of strings and checks if it should
// be ignored or not
func tagIgnoredIAM(t *iam.Tag) bool {
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

// tagKeysIam returns the keys for the list of IAM tags
func tagKeysIam(ts []*iam.Tag) []*string {
	result := make([]*string, 0, len(ts))
	for _, t := range ts {
		result = append(result, t.Key)
	}
	return result
}
