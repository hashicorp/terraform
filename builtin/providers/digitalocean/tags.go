package digitalocean

import (
	"context"
	"log"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/schema"
)

// setTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func setTags(conn *godo.Client, d *schema.ResourceData) error {
	oraw, nraw := d.GetChange("tags")
	remove, create := diffTags(tagsFromSchema(oraw), tagsFromSchema(nraw))

	log.Printf("[DEBUG] Removing tags: %#v from %s", remove, d.Id())
	for _, tag := range remove {
		_, err := conn.Tags.UntagResources(context.Background(), tag, &godo.UntagResourcesRequest{
			Resources: []godo.Resource{
				{
					ID:   d.Id(),
					Type: godo.DropletResourceType,
				},
			},
		})
		if err != nil {
			return err
		}
	}

	log.Printf("[DEBUG] Creating tags: %s for %s", create, d.Id())
	for _, tag := range create {
		_, err := conn.Tags.TagResources(context.Background(), tag, &godo.TagResourcesRequest{
			Resources: []godo.Resource{
				{
					ID:   d.Id(),
					Type: godo.DropletResourceType,
				},
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// tagsFromSchema takes the raw schema tags and returns them as a
// properly asserted map[string]string
func tagsFromSchema(raw interface{}) map[string]string {
	result := make(map[string]string)
	for _, t := range raw.([]interface{}) {
		result[t.(string)] = t.(string)
	}

	return result
}

// diffTags takes the old and the new tag sets and returns the difference of
// both. The remaining tags are those that need to be removed and created
func diffTags(oldTags, newTags map[string]string) (map[string]string, map[string]string) {
	for k := range oldTags {
		_, ok := newTags[k]
		if ok {
			delete(newTags, k)
			delete(oldTags, k)
		}
	}

	return oldTags, newTags
}
