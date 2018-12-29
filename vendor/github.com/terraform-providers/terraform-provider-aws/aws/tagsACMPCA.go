package aws

import (
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/acmpca"
)

// diffTags takes our tags locally and the ones remotely and returns
// the set of tags that must be created, and the set of tags that must
// be destroyed.
func diffTagsACMPCA(oldTags, newTags []*acmpca.Tag) ([]*acmpca.Tag, []*acmpca.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
	}

	// Build the list of what to remove
	var remove []*acmpca.Tag
	for _, t := range oldTags {
		old, ok := create[aws.StringValue(t.Key)]
		if !ok || old != aws.StringValue(t.Value) {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return tagsFromMapACMPCA(create), remove
}

func tagsFromMapACMPCA(m map[string]interface{}) []*acmpca.Tag {
	result := []*acmpca.Tag{}
	for k, v := range m {
		result = append(result, &acmpca.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		})
	}

	return result
}

func tagsToMapACMPCA(ts []*acmpca.Tag) map[string]string {
	result := map[string]string{}
	for _, t := range ts {
		result[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
	}

	return result
}

// compare a tag against a list of strings and checks if it should
// be ignored or not
func tagIgnoredACMPCA(t *acmpca.Tag) bool {
	filter := []string{"^aws:"}
	for _, v := range filter {
		log.Printf("[DEBUG] Matching %v with %v\n", v, aws.StringValue(t.Key))
		if r, _ := regexp.MatchString(v, aws.StringValue(t.Key)); r == true {
			log.Printf("[DEBUG] Found AWS specific tag %s (val: %s), ignoring.\n", aws.StringValue(t.Key), aws.StringValue(t.Value))
			return true
		}
	}
	return false
}
