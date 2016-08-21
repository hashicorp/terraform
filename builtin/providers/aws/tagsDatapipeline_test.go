package aws

import ()
import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/datapipeline"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"reflect"
	"testing"
)

func TestDiffDatapipelineTags(t *testing.T) {
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
		c, r := diffTagsDatapipeline(tagsFromMapDatapipeline(tc.Old), tagsFromMapDatapipeline(tc.New))
		cm := tagsToMapDatapipeline(c)
		rm := tagsToMapDatapipeline(r)
		if !reflect.DeepEqual(cm, tc.Create) {
			t.Fatalf("%d: bad create: %#v", i, cm)
		}
		if !reflect.DeepEqual(rm, tc.Remove) {
			t.Fatalf("%d: bad remove: %#v", i, rm)
		}
	}
}

// testAccCheckDatapipelineCheckTags can be used to check the tags on a trail
func testAccCheckDatapipelineCheckTags(tags *[]*datapipeline.Tag, expectedTags map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if !reflect.DeepEqual(expectedTags, tagsFromMapDatapipeline(*tags)) {
			return fmt.Errorf("Tags mismatch.\nExpected: %#v\nGiven: %#v",
				expectedTags, tagsToMapDatapipeline(*tags))
		}
		return nil
	}
}
