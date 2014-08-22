package aws

import (
	"log"
	"sort"

	"github.com/hashicorp/terraform/flatmap"
	"github.com/mitchellh/goamz/ec2"
)

func resource_aws_build_tags(attributes map[string]string, prefix string) []ec2.Tag {
	tags := make([]ec2.Tag, 0)

	if rawTagList := flatmap.Expand(attributes, prefix); rawTagList != nil {
		if tagList, ok := rawTagList.([]interface{}); ok {
			for _, rawTag := range tagList {
				tag, ok := rawTag.(map[string]interface{})
				if !ok {
					continue
				}

				tagKeyRaw, ok := tag["key"]
				if !ok {
					continue
				}

				tagValueRaw, ok := tag["value"]
				if !ok {
					continue
				}

				tagKey, ok := tagKeyRaw.(string)
				if !ok {
					continue
				}

				tagValue, ok := tagValueRaw.(string)
				if !ok {
					continue
				}

				tags = append(tags, ec2.Tag{
					Key:   tagKey,
					Value: tagValue,
				})
			}
		}
	}

	sort.Stable(sortableTags(tags))

	return tags
}

func resource_aws_sync_tags(ec2conn *ec2.EC2, resourceId string, oldTags, newTags []ec2.Tag) error {
	toDelete := make([]ec2.Tag, 0)
	toModify := make([]ec2.Tag, 0)

	for i := 0; i < len(oldTags); i++ {
		found := false

		for j := 0; j < len(newTags); j++ {
			if oldTags[i].Key == newTags[j].Key {
				found = true

				if oldTags[i].Value != newTags[j].Value {
					toModify = append(toModify, ec2.Tag{
						Key:   oldTags[i].Key,
						Value: newTags[j].Value,
					})
				}

				break
			}
		}

		if !found {
			toDelete = append(toDelete, ec2.Tag{
				Key: oldTags[i].Key,
			})
		}
	}

	for i := 0; i < len(newTags); i++ {
		found := false

		for j := 0; j < len(oldTags); j++ {
			if newTags[i].Key == oldTags[j].Key {
				found = true

				break
			}
		}

		if !found {
			toModify = append(toModify, ec2.Tag{
				Key:   newTags[i].Key,
				Value: newTags[i].Value,
			})
		}
	}

	log.Printf("[DEBUG] deleting tags: %#v", toDelete)
	log.Printf("[DEBUG] modifying tags: %#v", toModify)

	if len(toDelete) > 0 {
		if _, err := ec2conn.DeleteTags([]string{resourceId}, toDelete); err != nil {
			return err
		}
	}

	if len(toModify) > 0 {
		if _, err := ec2conn.CreateTags([]string{resourceId}, toModify); err != nil {
			return err
		}
	}

	return nil
}

type sortableTags []ec2.Tag

func (s sortableTags) Len() int           { return len(s) }
func (s sortableTags) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s sortableTags) Less(i, j int) bool { return s[i].Key < s[j].Key }
