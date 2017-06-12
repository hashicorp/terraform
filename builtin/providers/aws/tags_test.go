package aws

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestDiffTags(t *testing.T) {
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
		c, r := diffTags(tagsFromMap(tc.Old), tagsFromMap(tc.New))
		cm := tagsToMap(c)
		rm := tagsToMap(r)
		if !reflect.DeepEqual(cm, tc.Create) {
			t.Fatalf("%d: bad create: %#v", i, cm)
		}
		if !reflect.DeepEqual(rm, tc.Remove) {
			t.Fatalf("%d: bad remove: %#v", i, rm)
		}
	}
}

// Test the ability to opt-out of internal AWS tag filtering when using a data
// source.
func TestAllowInternalTags(t *testing.T) {
	var ignoredTags []*ec2.Tag
	var ignoredTagsMap map[string]interface{}

	const (
		tagKey   string = "aws:cloudformation:logical-id"
		tagValue string = "foo"
	)

	ignoredTags = append(ignoredTags,
		&ec2.Tag{
			Key:   aws.String(tagKey),
			Value: aws.String(tagValue),
		})

	ignoredTagsMap = make(map[string]interface{})
	ignoredTagsMap[tagKey] = tagValue

	// Make two calls, one that should allow internal AWS tags and one that should not.
	failFromMap := tagsFromMap(ignoredTagsMap)
	successFromMap := tagsFromMapUnfiltered(ignoredTagsMap)
	if len(failFromMap) != 0 {
		t.Fatalf("Test[tagsFromMap]: Tag %v with value %v was not ignored and should have been.", tagKey, tagValue)
	}

	if len(successFromMap) != 0 {
		for _, tag := range successFromMap {
			if (*tag.Key != tagKey) || (*tag.Value != tagValue) {
				t.Fatalf("Test[tagsFromMap]: Tag %v with value %v does not match the expected tag %v with value %v.", *tag.Key, *tag.Value, tagKey, tagValue)
			}
		}
	} else {
		t.Fatalf("Test[tagsFromMap]: Tag %v with value %v was ignored and should not have been.", tagKey, tagValue)
	}

	// Make two calls, one that should allow internal AWS tags and one that should not.
	failToMap := tagsToMap(ignoredTags)
	successToMap := tagsToMapUnfiltered(ignoredTags)

	if len(successToMap) != 0 {
		for tag, value := range successToMap {
			if (tag != tagKey) || (value != tagValue) {
				t.Fatalf("Test[tagsToMap]: Tag %v with value %v does not match the expected tag %v with value %v.", tag, value, tagKey, tagValue)
			}
		}
	} else {
		t.Fatalf("Test[tagsToMap]: Tag %v with value %v was ignored and should not have been.", tagKey, tagValue)
	}

	if len(failToMap) != 0 {
		t.Fatalf("Test[tagsToMap]: Tag %v with value %v was not ignored and should have been.", tagKey, tagValue)
	}
}

func TestIgnoringTags(t *testing.T) {
	var ignoredTags []*ec2.Tag
	ignoredTags = append(ignoredTags, &ec2.Tag{

		Key:   aws.String("aws:cloudformation:logical-id"),
		Value: aws.String("foo"),
	})
	ignoredTags = append(ignoredTags, &ec2.Tag{
		Key:   aws.String("aws:foo:bar"),
		Value: aws.String("baz"),
	})
	for _, tag := range ignoredTags {
		if !tagIgnored(tag) {
			t.Fatalf("Tag %v with value %v not ignored, but should be!", *tag.Key, *tag.Value)
		}
	}
}

// testAccCheckTags can be used to check the tags on a resource.
func testAccCheckTags(
	ts *[]*ec2.Tag, key string, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		m := tagsToMap(*ts)
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
