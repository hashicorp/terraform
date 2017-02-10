package circonus

import (
	"reflect"
	"testing"
)

func TestTagsRoundtrip(t *testing.T) {
	tagsIn := []string{"category:val1", "category:val2", "cat2:val", "other", "other:empty"}
	tags := apiToTags(tagsIn)
	if len(tags) != len(tagsIn) {
		t.Fatalf("length must be the same")
	}
	for i := range tagsIn {
		if tagsIn[i] != string(tags[i]) {
			t.Errorf("[%d] Tags should be identical (beyond types) after conversion: %#v/%#v", i, tagsIn[i], tags[i])
		}
	}

	tagsOut := tagsToAPI(tags)
	if len(tagsOut) != len(tags) {
		t.Fatalf("length must be the same")
	}
	for i := range tags {
		if string(tags[i]) != tagsOut[i] {
			t.Errorf("[%d] Tags should be identical (beyond types) after conversion: %#v/%#v", i, tags[i], tagsOut[i])
		}
	}

	if !reflect.DeepEqual(tagsIn, tagsOut) {
		t.Errorf("Tags should be identical after round trip: %#v/%#v", tagsIn, tagsOut)
	}
}
