package aws

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

// go test -v -run="TestDiffKMSTags"
func TestDiffKMSTags(t *testing.T) {
	cases := []struct {
		Old, New       map[string]interface{}
		Create, Remove map[string]string
	}{
		// Basic add/remove
		{
			Old: map[string]interface{}{
				"foo": "bar",
			},
			New: map[string]interface{}{
				"bar": "baz",
			},
			Create: map[string]string{
				"bar": "baz",
			},
			Remove: map[string]string{
				"foo": "bar",
			},
		},

		// Modify
		{
			Old: map[string]interface{}{
				"foo": "bar",
			},
			New: map[string]interface{}{
				"foo": "baz",
			},
			Create: map[string]string{
				"foo": "baz",
			},
			Remove: map[string]string{
				"foo": "bar",
			},
		},
	}

	for i, tc := range cases {
		c, r := diffTagsKMS(tagsFromMapKMS(tc.Old), tagsFromMapKMS(tc.New))
		cm := tagsToMapKMS(c)
		rm := tagsToMapKMS(r)
		if !reflect.DeepEqual(cm, tc.Create) {
			t.Fatalf("%d: bad create: %#v", i, cm)
		}
		if !reflect.DeepEqual(rm, tc.Remove) {
			t.Fatalf("%d: bad remove: %#v", i, rm)
		}
	}
}

// go test -v -run="TestIgnoringTagsKMS"
func TestIgnoringTagsKMS(t *testing.T) {
	var ignoredTags []*kms.Tag
	ignoredTags = append(ignoredTags, &kms.Tag{
		TagKey:   aws.String("aws:cloudformation:logical-id"),
		TagValue: aws.String("foo"),
	})
	ignoredTags = append(ignoredTags, &kms.Tag{
		TagKey:   aws.String("aws:foo:bar"),
		TagValue: aws.String("baz"),
	})
	for _, tag := range ignoredTags {
		if !tagIgnoredKMS(tag) {
			t.Fatalf("Tag %v with value %v not ignored, but should be!", *tag.TagKey, *tag.TagValue)
		}
	}
}

// testAccCheckTags can be used to check the tags on a resource.
func testAccCheckKMSTags(
	ts []*kms.Tag, key string, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		m := tagsToMapKMS(ts)
		v, ok := m[key]
		if value != "" && !ok {
			return fmt.Errorf("Missing tag: %s", key)
		} else if value == "" && ok {
			return fmt.Errorf("Extra tag: %s", key)
		}
		if value == "" {
			return nil
		}

		if v != value {
			return fmt.Errorf("%s: bad value: %s", key, v)
		}

		return nil
	}
}
