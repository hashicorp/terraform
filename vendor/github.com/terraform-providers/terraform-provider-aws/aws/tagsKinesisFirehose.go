package aws

import (
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/hashicorp/terraform/helper/schema"
)

// getTags is a helper to get the tags for a resource. It expects the
// tags field to be named "tags"
func getTagsKinesisFirehose(conn *firehose.Firehose, d *schema.ResourceData, sn string) error {
	tags := make([]*firehose.Tag, 0)
	var exclusiveStartTagKey string
	for {
		req := &firehose.ListTagsForDeliveryStreamInput{
			DeliveryStreamName: aws.String(sn),
		}
		if exclusiveStartTagKey != "" {
			req.ExclusiveStartTagKey = aws.String(exclusiveStartTagKey)
		}

		resp, err := conn.ListTagsForDeliveryStream(req)
		if err != nil {
			return err
		}

		for _, tag := range resp.Tags {
			tags = append(tags, tag)
		}

		// If HasMoreTags is true in the response, more tags are available.
		// To list the remaining tags, set ExclusiveStartTagKey to the key
		// of the last tag returned and call ListTagsForDeliveryStream again.
		if !aws.BoolValue(resp.HasMoreTags) {
			break
		}
		exclusiveStartTagKey = aws.StringValue(tags[len(tags)-1].Key)
	}

	if err := d.Set("tags", tagsToMapKinesisFirehose(tags)); err != nil {
		return err
	}

	return nil
}

// setTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func setTagsKinesisFirehose(conn *firehose.Firehose, d *schema.ResourceData, sn string) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffTagsKinesisFirehose(tagsFromMapKinesisFirehose(o), tagsFromMapKinesisFirehose(n))

		// Set tags
		if len(remove) > 0 {
			log.Printf("[DEBUG] Removing tags: %#v", remove)
			k := make([]*string, len(remove), len(remove))
			for i, t := range remove {
				k[i] = t.Key
			}

			_, err := conn.UntagDeliveryStream(&firehose.UntagDeliveryStreamInput{
				DeliveryStreamName: aws.String(sn),
				TagKeys:            k,
			})
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			log.Printf("[DEBUG] Creating tags: %#v", create)
			_, err := conn.TagDeliveryStream(&firehose.TagDeliveryStreamInput{
				DeliveryStreamName: aws.String(sn),
				Tags:               create,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// diffTags takes our tags locally and the ones remotely and returns
// the set of tags that must be created, and the set of tags that must
// be destroyed.
func diffTagsKinesisFirehose(oldTags, newTags []*firehose.Tag) ([]*firehose.Tag, []*firehose.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
	}

	// Build the list of what to remove
	var remove []*firehose.Tag
	for _, t := range oldTags {
		old, ok := create[aws.StringValue(t.Key)]
		if !ok || old != aws.StringValue(t.Value) {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return tagsFromMapKinesisFirehose(create), remove
}

// tagsFromMap returns the tags for the given map of data.
func tagsFromMapKinesisFirehose(m map[string]interface{}) []*firehose.Tag {
	result := make([]*firehose.Tag, 0, len(m))
	for k, v := range m {
		t := &firehose.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		}
		if !tagIgnoredKinesisFirehose(t) {
			result = append(result, t)
		}
	}

	return result
}

// tagsToMap turns the list of tags into a map.
func tagsToMapKinesisFirehose(ts []*firehose.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		if !tagIgnoredKinesisFirehose(t) {
			result[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
		}
	}

	return result
}

// compare a tag against a list of strings and checks if it should
// be ignored or not
func tagIgnoredKinesisFirehose(t *firehose.Tag) bool {
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
