package aws

import (
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
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
			_, err := RetryOnAwsCodes([]string{"NoSuchBucket", "OperationAborted"}, func() (interface{}, error) {
				return conn.DeleteBucketTagging(&s3.DeleteBucketTaggingInput{
					Bucket: aws.String(d.Get("bucket").(string)),
				})
			})
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			log.Printf("[DEBUG] Creating tags: %#v", create)
			req := &s3.PutBucketTaggingInput{
				Bucket: aws.String(d.Get("bucket").(string)),
				Tagging: &s3.Tagging{
					TagSet: create,
				},
			}

			_, err := RetryOnAwsCodes([]string{"NoSuchBucket", "OperationAborted"}, func() (interface{}, error) {
				return conn.PutBucketTagging(req)
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
func diffTagsS3(oldTags, newTags []*s3.Tag) ([]*s3.Tag, []*s3.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []*s3.Tag
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
func tagsFromMapS3(m map[string]interface{}) []*s3.Tag {
	result := make([]*s3.Tag, 0, len(m))
	for k, v := range m {
		t := &s3.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		}
		if !tagIgnoredS3(t) {
			result = append(result, t)
		}
	}

	return result
}

// tagsToMap turns the list of tags into a map.
func tagsToMapS3(ts []*s3.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		if !tagIgnoredS3(t) {
			result[*t.Key] = *t.Value
		}
	}

	return result
}

// return a slice of s3 tags associated with the given s3 bucket. Essentially
// s3.GetBucketTagging, except returns an empty slice instead of an error when
// there are no tags.
func getTagSetS3(s3conn *s3.S3, bucket string) ([]*s3.Tag, error) {
	request := &s3.GetBucketTaggingInput{
		Bucket: aws.String(bucket),
	}

	response, err := s3conn.GetBucketTagging(request)
	if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "NoSuchTagSet" {
		// There is no tag set associated with the bucket.
		return []*s3.Tag{}, nil
	} else if err != nil {
		return nil, err
	}

	return response.TagSet, nil
}

// compare a tag against a list of strings and checks if it should
// be ignored or not
func tagIgnoredS3(t *s3.Tag) bool {
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
