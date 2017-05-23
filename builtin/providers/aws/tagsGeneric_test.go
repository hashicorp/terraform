package aws

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
)

// go test -v -run="TestDiffGenericTags"
func TestDiffGenericTags(t *testing.T) {
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
		c, r := diffTagsGeneric(tc.Old, tc.New)
		cm := tagsToMapGeneric(c)
		rm := tagsToMapGeneric(r)
		if !reflect.DeepEqual(cm, tc.Create) {
			t.Fatalf("%d: bad create: %#v", i, cm)
		}
		if !reflect.DeepEqual(rm, tc.Remove) {
			t.Fatalf("%d: bad remove: %#v", i, rm)
		}
	}
}

// go test -v -run="TestIgnoringTagsGeneric"
func TestIgnoringTagsGeneric(t *testing.T) {
	ignoredTags := map[string]*string{
		"aws:cloudformation:logical-id": aws.String("foo"),
		"aws:foo:bar":                   aws.String("baz"),
	}
	for k, v := range ignoredTags {
		if !tagIgnoredGeneric(k) {
			t.Fatalf("Tag %v with value %v not ignored, but should be!", k, *v)
		}
	}
}
