package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/codebuild"
)

func tagsFromMapCodeBuild(m map[string]interface{}) []*codebuild.Tag {
	result := []*codebuild.Tag{}
	for k, v := range m {
		result = append(result, &codebuild.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		})
	}

	return result
}

func tagsToMapCodeBuild(ts []*codebuild.Tag) map[string]string {
	result := map[string]string{}
	for _, t := range ts {
		result[*t.Key] = *t.Value
	}

	return result
}
