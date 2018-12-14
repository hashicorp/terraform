package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/datasync"
)

// dataSyncTagsDiff takes our tags locally and the ones remotely and returns
// the set of tags that must be created, and the set of tags that must
// be destroyed.
func dataSyncTagsDiff(oldTags, newTags []*datasync.TagListEntry) ([]*datasync.TagListEntry, []*datasync.TagListEntry) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
	}

	// Build the list of what to remove
	var remove []*datasync.TagListEntry
	for _, t := range oldTags {
		old, ok := create[aws.StringValue(t.Key)]
		if !ok || old != aws.StringValue(t.Value) {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return expandDataSyncTagListEntry(create), remove
}

func dataSyncTagsKeys(tags []*datasync.TagListEntry) []*string {
	keys := make([]*string, 0)

	for _, tag := range tags {
		if tag == nil {
			continue
		}
		keys = append(keys, tag.Key)
	}

	return keys
}

func expandDataSyncTagListEntry(m map[string]interface{}) []*datasync.TagListEntry {
	result := []*datasync.TagListEntry{}
	for k, v := range m {
		result = append(result, &datasync.TagListEntry{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		})
	}

	return result
}

func flattenDataSyncTagListEntry(ts []*datasync.TagListEntry) map[string]string {
	result := map[string]string{}
	for _, t := range ts {
		result[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
	}

	return result
}
