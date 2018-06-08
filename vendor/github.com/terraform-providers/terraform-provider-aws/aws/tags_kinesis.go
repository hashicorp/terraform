package aws

import (
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/hashicorp/terraform/helper/schema"
)

// setTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func setTagsKinesis(conn *kinesis.Kinesis, d *schema.ResourceData) error {

	sn := d.Get("name").(string)

	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffTagsKinesis(tagsFromMapKinesis(o), tagsFromMapKinesis(n))

		// Set tags
		if len(remove) > 0 {
			log.Printf("[DEBUG] Removing tags: %#v", remove)
			k := make([]*string, len(remove), len(remove))
			for i, t := range remove {
				k[i] = t.Key
			}

			_, err := conn.RemoveTagsFromStream(&kinesis.RemoveTagsFromStreamInput{
				StreamName: aws.String(sn),
				TagKeys:    k,
			})
			if err != nil {
				return err
			}
		}

		if len(create) > 0 {

			log.Printf("[DEBUG] Creating tags: %#v", create)
			t := make(map[string]*string)
			for _, tag := range create {
				t[*tag.Key] = tag.Value
			}

			_, err := conn.AddTagsToStream(&kinesis.AddTagsToStreamInput{
				StreamName: aws.String(sn),
				Tags:       t,
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
func diffTagsKinesis(oldTags, newTags []*kinesis.Tag) ([]*kinesis.Tag, []*kinesis.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []*kinesis.Tag
	for _, t := range oldTags {
		old, ok := create[*t.Key]
		if !ok || old != *t.Value {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return tagsFromMapKinesis(create), remove
}

// tagsFromMap returns the tags for the given map of data.
func tagsFromMapKinesis(m map[string]interface{}) []*kinesis.Tag {
	var result []*kinesis.Tag
	for k, v := range m {
		t := &kinesis.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		}
		if !tagIgnoredKinesis(t) {
			result = append(result, t)
		}
	}

	return result
}

// tagsToMap turns the list of tags into a map.
func tagsToMapKinesis(ts []*kinesis.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		if !tagIgnoredKinesis(t) {
			result[*t.Key] = *t.Value
		}
	}

	return result
}

// compare a tag against a list of strings and checks if it should
// be ignored or not
func tagIgnoredKinesis(t *kinesis.Tag) bool {
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
