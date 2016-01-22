package cloudstack

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

// tagsSchema returns the schema to use for tags.
//
func tagsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
	}
}

// setTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func setTags(cs *cloudstack.CloudStackClient, d *schema.ResourceData, resourcetype string) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffTags(tagsFromSchema(o), tagsFromSchema(n))
		log.Printf("[DEBUG] tags to remove: %v", remove)
		log.Printf("[DEBUG] tags to create: %v", create)

		// Set tags
		if len(remove) > 0 {
			log.Printf("[DEBUG] Removing tags: %v from %s", remove, d.Id())
			p := cs.Resourcetags.NewDeleteTagsParams([]string{d.Id()}, resourcetype)
			p.SetTags(remove)
			_, err := cs.Resourcetags.DeleteTags(p)
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			log.Printf("[DEBUG] Creating tags: %v for %s", create, d.Id())
			p := cs.Resourcetags.NewCreateTagsParams([]string{d.Id()}, resourcetype, create)
			_, err := cs.Resourcetags.CreateTags(p)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// diffTags takes our tags locally and the ones remotely and returns
// the set of tags that must be destroyed, and the set of tags that must
// be created.
func diffTags(oldTags, newTags map[string]string) (map[string]string, map[string]string) {
	remove := make(map[string]string)
	for k, v := range oldTags {
		old, ok := newTags[k]
		if !ok || old != v {
			// Delete it!
			remove[k] = v
		} else {
			// We need to remove the modified tags to create them again,
			// but we should avoid creating what we already have
			delete(newTags, k)
		}
	}

	return newTags, remove
}

// tagsFromSchema returns the tags for the given tags schema.
// It's needed to properly unpack all string:interface types
// to a proper string:string map
func tagsFromSchema(m map[string]interface{}) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = v.(string)
	}
	return result
}
