package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/codebuild"
)

// diffTags takes our tags locally and the ones remotely and returns
// the set of tags that must be created, and the set of tags that must
// be destroyed.
func diffTagsCodeBuild(oldTags, newTags []*codebuild.Tag) ([]*codebuild.Tag, []*codebuild.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []*codebuild.Tag
	for _, t := range oldTags {
		old, ok := create[*t.Key]
		if !ok || old != *t.Value {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return tagsFromMapCodeBuild(create), remove
}

func tagsFromMapCodeBuild(m map[string]interface{}) []*codebuild.Tag {
	result := make([]*codebuild.Tag, 0, len(m))
	for k, v := range m {
		result = append(result, &codebuild.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		})
	}

	return result
}

func tagsToMapCodeBuild(ts []*codebuild.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		result[*t.Key] = *t.Value
	}

	return result
}
