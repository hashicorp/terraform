package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/hashicorp/terraform/helper/schema"
)

// setTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func setTagsELB(conn *elb.ELB, d *schema.ResourceData) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffTagsELB(tagsFromMapELB(o), tagsFromMapELB(n))

		// Set tags
		if len(remove) > 0 {
			log.Printf("[DEBUG] Removing tags: %#v", remove)
			k := make([]*elb.TagKeyOnly, 0, len(remove))
			for _, t := range remove {
				k = append(k, &elb.TagKeyOnly{Key: t.Key})
			}
			_, err := conn.RemoveTags(&elb.RemoveTagsInput{
				LoadBalancerNames: []*string{aws.String(d.Get("name").(string))},
				Tags:              k,
			})
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			log.Printf("[DEBUG] Creating tags: %#v", create)
			_, err := conn.AddTags(&elb.AddTagsInput{
				LoadBalancerNames: []*string{aws.String(d.Get("name").(string))},
				Tags:              create,
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
func diffTagsELB(oldTags, newTags []*elb.Tag) ([]*elb.Tag, []*elb.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []*elb.Tag
	for _, t := range oldTags {
		old, ok := create[*t.Key]
		if !ok || old != *t.Value {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return tagsFromMapELB(create), remove
}

// tagsFromMap returns the tags for the given map of data.
func tagsFromMapELB(m map[string]interface{}) []*elb.Tag {
	var result []*elb.Tag
	for k, v := range m {
		result = append(result, &elb.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		})
	}

	return result
}

// tagsToMap turns the list of tags into a map.
func tagsToMapELB(ts []*elb.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		result[*t.Key] = *t.Value
	}

	return result
}
