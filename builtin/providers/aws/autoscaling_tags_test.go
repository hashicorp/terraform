package aws

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestDiffAutoscalingTags(t *testing.T) {
	cases := []struct {
		Old, New       map[string]interface{}
		Create, Remove map[string]interface{}
	}{
		// Basic add/remove
		{
			Old: map[string]interface{}{
				"Name": map[string]interface{}{
					"key":                 "Name",
					"value":               "bar",
					"propagate_at_launch": true,
				},
			},
			New: map[string]interface{}{
				"DifferentTag": map[string]interface{}{
					"key":                 "DifferentTag",
					"value":               "baz",
					"propagate_at_launch": true,
				},
			},
			Create: map[string]interface{}{
				"DifferentTag": map[string]interface{}{
					"key":                 "DifferentTag",
					"value":               "baz",
					"propagate_at_launch": true,
				},
			},
			Remove: map[string]interface{}{
				"Name": map[string]interface{}{
					"key":                 "Name",
					"value":               "bar",
					"propagate_at_launch": true,
				},
			},
		},

		// Modify
		{
			Old: map[string]interface{}{
				"Name": map[string]interface{}{
					"key":                 "Name",
					"value":               "bar",
					"propagate_at_launch": true,
				},
			},
			New: map[string]interface{}{
				"Name": map[string]interface{}{
					"key":                 "Name",
					"value":               "baz",
					"propagate_at_launch": false,
				},
			},
			Create: map[string]interface{}{
				"Name": map[string]interface{}{
					"key":                 "Name",
					"value":               "baz",
					"propagate_at_launch": false,
				},
			},
			Remove: map[string]interface{}{
				"Name": map[string]interface{}{
					"key":                 "Name",
					"value":               "bar",
					"propagate_at_launch": true,
				},
			},
		},
	}

	var resourceID = "sample"

	for i, tc := range cases {
		awsTagsOld, err := autoscalingTagsFromMap(tc.Old, resourceID)
		if err != nil {
			t.Fatalf("%d: unexpected error convertig old tags: %v", i, err)
		}

		awsTagsNew, err := autoscalingTagsFromMap(tc.New, resourceID)
		if err != nil {
			t.Fatalf("%d: unexpected error convertig new tags: %v", i, err)
		}

		c, r, err := diffAutoscalingTags(awsTagsOld, awsTagsNew, resourceID)
		if err != nil {
			t.Fatalf("%d: unexpected error diff'ing tags: %v", i, err)
		}

		cm := autoscalingTagsToMap(c)
		rm := autoscalingTagsToMap(r)
		if !reflect.DeepEqual(cm, tc.Create) {
			t.Fatalf("%d: bad create: \n%#v\n%#v", i, cm, tc.Create)
		}
		if !reflect.DeepEqual(rm, tc.Remove) {
			t.Fatalf("%d: bad remove: \n%#v\n%#v", i, rm, tc.Remove)
		}
	}
}

// testAccCheckTags can be used to check the tags on a resource.
func testAccCheckAutoscalingTags(
	ts *[]*autoscaling.TagDescription, key string, expected map[string]interface{}) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		m := autoscalingTagDescriptionsToMap(ts)
		v, ok := m[key]
		if !ok {
			return fmt.Errorf("Missing tag: %s", key)
		}

		if v["value"] != expected["value"].(string) ||
			v["propagate_at_launch"] != expected["propagate_at_launch"].(bool) {
			return fmt.Errorf("%s: bad value: %s", key, v)
		}

		return nil
	}
}

func testAccCheckAutoscalingTagNotExists(ts *[]*autoscaling.TagDescription, key string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		m := autoscalingTagDescriptionsToMap(ts)
		if _, ok := m[key]; ok {
			return fmt.Errorf("Tag exists when it should not: %s", key)
		}

		return nil
	}
}

func TestIgnoringTagsAutoscaling(t *testing.T) {
	var ignoredTags []*autoscaling.Tag
	ignoredTags = append(ignoredTags, &autoscaling.Tag{
		Key:   aws.String("aws:cloudformation:logical-id"),
		Value: aws.String("foo"),
	})
	ignoredTags = append(ignoredTags, &autoscaling.Tag{
		Key:   aws.String("aws:foo:bar"),
		Value: aws.String("baz"),
	})
	for _, tag := range ignoredTags {
		if !tagIgnoredAutoscaling(tag) {
			t.Fatalf("Tag %v with value %v not ignored, but should be!", *tag.Key, *tag.Value)
		}
	}
}
