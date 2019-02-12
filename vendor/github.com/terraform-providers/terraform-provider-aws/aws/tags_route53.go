package aws

import (
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/hashicorp/terraform/helper/schema"
)

// setTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func setTagsR53(conn *route53.Route53, d *schema.ResourceData, resourceType string) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffTagsR53(tagsFromMapR53(o), tagsFromMapR53(n))

		// Set tags
		r := make([]*string, len(remove))
		for i, t := range remove {
			r[i] = t.Key
		}
		log.Printf("[DEBUG] Changing tags: \n\tadding: %#v\n\tremoving:%#v", create, remove)
		req := &route53.ChangeTagsForResourceInput{
			ResourceId:   aws.String(d.Id()),
			ResourceType: aws.String(resourceType),
		}

		if len(create) > 0 {
			req.AddTags = create
		}
		if len(r) > 0 {
			req.RemoveTagKeys = r
		}

		_, err := conn.ChangeTagsForResource(req)
		if err != nil {
			return err
		}
	}

	return nil
}

// diffTags takes our tags locally and the ones remotely and returns
// the set of tags that must be created, and the set of tags that must
// be destroyed.
func diffTagsR53(oldTags, newTags []*route53.Tag) ([]*route53.Tag, []*route53.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []*route53.Tag
	for _, t := range oldTags {
		old, ok := create[*t.Key]
		if !ok || old != *t.Value {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return tagsFromMapR53(create), remove
}

// tagsFromMap returns the tags for the given map of data.
func tagsFromMapR53(m map[string]interface{}) []*route53.Tag {
	result := make([]*route53.Tag, 0, len(m))
	for k, v := range m {
		t := &route53.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		}
		if !tagIgnoredRoute53(t) {
			result = append(result, t)
		}
	}

	return result
}

// tagsToMap turns the list of tags into a map.
func tagsToMapR53(ts []*route53.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		if !tagIgnoredRoute53(t) {
			result[*t.Key] = *t.Value
		}
	}

	return result
}

// compare a tag against a list of strings and checks if it should
// be ignored or not
func tagIgnoredRoute53(t *route53.Tag) bool {
	filter := []string{"^aws:"}
	for _, v := range filter {
		log.Printf("[DEBUG] Matching %v with %v\n", v, *t.Key)
		r, _ := regexp.MatchString(v, *t.Key)
		if r {
			log.Printf("[DEBUG] Found AWS specific tag %s (val: %s), ignoring.\n", *t.Key, *t.Value)
			return true
		}
	}
	return false
}
