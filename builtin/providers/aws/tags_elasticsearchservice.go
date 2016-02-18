package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	elasticsearch "github.com/aws/aws-sdk-go/service/elasticsearchservice"
	"github.com/hashicorp/terraform/helper/schema"
)

// setTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func setTagsElasticsearchService(conn *elasticsearch.ElasticsearchService, d *schema.ResourceData, arn string) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffTagsElasticsearchService(tagsFromMapElasticsearchService(o), tagsFromMapElasticsearchService(n))

		// Set tags
		if len(remove) > 0 {
			log.Printf("[DEBUG] Removing tags: %#v", remove)
			k := make([]*string, 0, len(remove))
			for _, t := range remove {
				k = append(k, t.Key)
			}
			_, err := conn.RemoveTags(&elasticsearch.RemoveTagsInput{
				ARN:     aws.String(arn),
				TagKeys: k,
			})
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			log.Printf("[DEBUG] Creating tags: %#v", create)
			_, err := conn.AddTags(&elasticsearch.AddTagsInput{
				ARN:     aws.String(arn),
				TagList: create,
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
func diffTagsElasticsearchService(oldTags, newTags []*elasticsearch.Tag) ([]*elasticsearch.Tag, []*elasticsearch.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []*elasticsearch.Tag
	for _, t := range oldTags {
		old, ok := create[*t.Key]
		if !ok || old != *t.Value {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return tagsFromMapElasticsearchService(create), remove
}

// tagsFromMap returns the tags for the given map of data.
func tagsFromMapElasticsearchService(m map[string]interface{}) []*elasticsearch.Tag {
	var result []*elasticsearch.Tag
	for k, v := range m {
		result = append(result, &elasticsearch.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		})
	}

	return result
}

// tagsToMap turns the list of tags into a map.
func tagsToMapElasticsearchService(ts []*elasticsearch.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		result[*t.Key] = *t.Value
	}

	return result
}
