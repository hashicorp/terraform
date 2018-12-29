package aws

import (
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/directconnect"
	"github.com/hashicorp/terraform/helper/schema"
)

// getTags is a helper to get the tags for a resource. It expects the
// tags field to be named "tags"
func getTagsDX(conn *directconnect.DirectConnect, d *schema.ResourceData, arn string) error {
	resp, err := conn.DescribeTags(&directconnect.DescribeTagsInput{
		ResourceArns: aws.StringSlice([]string{arn}),
	})
	if err != nil {
		return err
	}

	var tags []*directconnect.Tag
	if len(resp.ResourceTags) == 1 && aws.StringValue(resp.ResourceTags[0].ResourceArn) == arn {
		tags = resp.ResourceTags[0].Tags
	}

	if err := d.Set("tags", tagsToMapDX(tags)); err != nil {
		return err
	}

	return nil
}

// setTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func setTagsDX(conn *directconnect.DirectConnect, d *schema.ResourceData, arn string) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffTagsDX(tagsFromMapDX(o), tagsFromMapDX(n))

		// Set tags
		if len(remove) > 0 {
			log.Printf("[DEBUG] Removing tags: %#v", remove)
			k := make([]*string, len(remove), len(remove))
			for i, t := range remove {
				k[i] = t.Key
			}

			_, err := conn.UntagResource(&directconnect.UntagResourceInput{
				ResourceArn: aws.String(arn),
				TagKeys:     k,
			})
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			log.Printf("[DEBUG] Creating tags: %#v", create)
			_, err := conn.TagResource(&directconnect.TagResourceInput{
				ResourceArn: aws.String(arn),
				Tags:        create,
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
func diffTagsDX(oldTags, newTags []*directconnect.Tag) ([]*directconnect.Tag, []*directconnect.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
	}

	// Build the list of what to remove
	var remove []*directconnect.Tag
	for _, t := range oldTags {
		old, ok := create[aws.StringValue(t.Key)]
		if !ok || old != aws.StringValue(t.Value) {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return tagsFromMapDX(create), remove
}

// tagsFromMap returns the tags for the given map of data.
func tagsFromMapDX(m map[string]interface{}) []*directconnect.Tag {
	result := make([]*directconnect.Tag, 0, len(m))
	for k, v := range m {
		t := &directconnect.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		}
		if !tagIgnoredDX(t) {
			result = append(result, t)
		}
	}

	return result
}

// tagsToMap turns the list of tags into a map.
func tagsToMapDX(ts []*directconnect.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		if !tagIgnoredDX(t) {
			result[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
		}
	}

	return result
}

// compare a tag against a list of strings and checks if it should
// be ignored or not
func tagIgnoredDX(t *directconnect.Tag) bool {
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
