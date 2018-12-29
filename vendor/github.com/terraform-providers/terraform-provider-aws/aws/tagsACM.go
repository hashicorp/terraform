package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/hashicorp/terraform/helper/schema"
)

func setTagsACM(conn *acm.ACM, d *schema.ResourceData) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffTagsACM(tagsFromMapACM(o), tagsFromMapACM(n))

		// Set tags
		if len(remove) > 0 {
			input := acm.RemoveTagsFromCertificateInput{
				CertificateArn: aws.String(d.Get("arn").(string)),
				Tags:           remove,
			}
			log.Printf("[DEBUG] Removing ACM tags: %s", input)
			_, err := conn.RemoveTagsFromCertificate(&input)
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			input := acm.AddTagsToCertificateInput{
				CertificateArn: aws.String(d.Get("arn").(string)),
				Tags:           create,
			}
			log.Printf("[DEBUG] Adding ACM tags: %s", input)
			_, err := conn.AddTagsToCertificate(&input)
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
func diffTagsACM(oldTags, newTags []*acm.Tag) ([]*acm.Tag, []*acm.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []*acm.Tag
	for _, t := range oldTags {
		old, ok := create[*t.Key]
		if !ok || old != *t.Value {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return tagsFromMapACM(create), remove
}

func tagsFromMapACM(m map[string]interface{}) []*acm.Tag {
	result := []*acm.Tag{}
	for k, v := range m {
		result = append(result, &acm.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		})
	}

	return result
}

func tagsToMapACM(ts []*acm.Tag) map[string]string {
	result := map[string]string{}
	for _, t := range ts {
		result[*t.Key] = *t.Value
	}

	return result
}
