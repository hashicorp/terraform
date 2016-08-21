package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/datapipeline"
	"github.com/hashicorp/terraform/helper/schema"
)

// setTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func setTagsDatapipeline(conn *datapipeline.DataPipeline, d *schema.ResourceData) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffTagsDatapipeline(tagsFromMapDatapipeline(o), tagsFromMapDatapipeline(n))
		removeStrings := tagsKeysFromMapDatapipeline(remove)

		// Set tags
		if len(remove) > 0 {
			input := datapipeline.RemoveTagsInput{
				PipelineId: aws.String(d.Id()),
				TagKeys:    removeStrings,
			}
			log.Printf("[DEBUG] Removing Datapipeline tags: %s", input)
			_, err := conn.RemoveTags(&input)
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			input := datapipeline.AddTagsInput{
				PipelineId: aws.String(d.Id()),
				Tags:       create,
			}
			log.Printf("[DEBUG] Adding Datapipeline tags: %s", input)
			_, err := conn.AddTags(&input)
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
func diffTagsDatapipeline(oldTags, newTags []*datapipeline.Tag) ([]*datapipeline.Tag, []*datapipeline.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []*datapipeline.Tag
	for _, t := range oldTags {
		old, ok := create[*t.Key]
		if !ok || old != *t.Value {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return tagsFromMapDatapipeline(create), remove
}

func tagsFromMapDatapipeline(m map[string]interface{}) []*datapipeline.Tag {
	var result []*datapipeline.Tag
	for k, v := range m {
		result = append(result, &datapipeline.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		})
	}

	return result
}

func tagsKeysFromMapDatapipeline(ts []*datapipeline.Tag) []*string {
	var result []*string
	for _, t := range ts {
		result = append(result, t.Key)
	}

	return result
}

func tagsToMapDatapipeline(ts []datapipeline.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		result[*t.Key] = *t.Value
	}

	return result
}
