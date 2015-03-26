package aws

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/xml"
	"log"

	"github.com/hashicorp/aws-sdk-go/aws"
	"github.com/hashicorp/aws-sdk-go/gen/s3"
	"github.com/hashicorp/terraform/helper/schema"
)

// setTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func setTagsS3(conn *s3.S3, d *schema.ResourceData) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffTagsS3(tagsFromMapS3(o), tagsFromMapS3(n))

		// Set tags
		if len(remove) > 0 {
			log.Printf("[DEBUG] Removing tags: %#v", remove)
			err := conn.DeleteBucketTagging(&s3.DeleteBucketTaggingRequest{
				Bucket: aws.String(d.Get("bucket").(string)),
			})
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			log.Printf("[DEBUG] Creating tags: %#v", create)
			tagging := s3.Tagging{
				TagSet: create,
				XMLName: xml.Name{
					Space: "http://s3.amazonaws.com/doc/2006-03-01/",
					Local: "Tagging",
				},
			}
			// AWS S3 API requires us to send a base64 encoded md5 hash of the
			// content, which we need to build ourselves since aws-sdk-go does not.
			b, err := xml.Marshal(tagging)
			if err != nil {
				return err
			}
			h := md5.New()
			h.Write(b)
			base := base64.StdEncoding.EncodeToString(h.Sum(nil))

			req := &s3.PutBucketTaggingRequest{
				Bucket:     aws.String(d.Get("bucket").(string)),
				ContentMD5: aws.String(base),
				Tagging:    &tagging,
			}

			err = conn.PutBucketTagging(req)
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
func diffTagsS3(oldTags, newTags []s3.Tag) ([]s3.Tag, []s3.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []s3.Tag
	for _, t := range oldTags {
		old, ok := create[*t.Key]
		if !ok || old != *t.Value {
			// Delete it!
			remove = append(remove, t)
		}
	}

	return tagsFromMapS3(create), remove
}

// tagsFromMap returns the tags for the given map of data.
func tagsFromMapS3(m map[string]interface{}) []s3.Tag {
	result := make([]s3.Tag, 0, len(m))
	for k, v := range m {
		result = append(result, s3.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		})
	}

	return result
}

// tagsToMap turns the list of tags into a map.
func tagsToMapS3(ts []s3.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		result[*t.Key] = *t.Value
	}

	return result
}
