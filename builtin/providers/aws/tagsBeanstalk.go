package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
)

// diffTags takes our tags locally and the ones remotely and returns
// the set of tags that must be created, and the set of tags that must
// be destroyed.
func diffTagsBeanstalk(oldTags, newTags []*elasticbeanstalk.Tag) ([]*elasticbeanstalk.Tag, []*elasticbeanstalk.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []*elasticbeanstalk.Tag
	for _, t := range oldTags {
		old, ok := create[*t.Key]
		if !ok || old != *t.Value {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return tagsFromMapBeanstalk(create), remove
}

// tagsFromMap returns the tags for the given map of data.
func tagsFromMapBeanstalk(m map[string]interface{}) []*elasticbeanstalk.Tag {
	var result []*elasticbeanstalk.Tag
	for k, v := range m {
		result = append(result, &elasticbeanstalk.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		})
	}

	return result
}

// tagsToMap turns the list of tags into a map.
func tagsToMapBeanstalk(ts []*elasticbeanstalk.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		result[*t.Key] = *t.Value
	}

	return result
}
