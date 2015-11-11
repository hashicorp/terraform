package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/redshift"
)

func tagsFromMapRedshift(m map[string]interface{}) []*redshift.Tag {
	result := make([]*redshift.Tag, 0, len(m))
	for k, v := range m {
		result = append(result, &redshift.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		})
	}

	return result
}

func tagsToMapRedshift(ts []*redshift.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		result[*t.Key] = *t.Value
	}

	return result
}
