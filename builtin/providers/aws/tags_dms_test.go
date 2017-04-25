package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	dms "github.com/aws/aws-sdk-go/service/databasemigrationservice"
	"reflect"
)

func TestDmsTagsToMap(t *testing.T) {
	tags := []*dms.Tag{
		{
			Key:   aws.String("test-key-1"),
			Value: aws.String("test-value-1"),
		},
		{
			Key:   aws.String("test-key-2"),
			Value: aws.String("test-value-2"),
		},
	}

	result := dmsTagsToMap(tags)

	for _, tag := range tags {
		if v, ok := result[*tag.Key]; ok {
			if v != *tag.Value {
				t.Fatalf("Key %s had value of %s. Expected %s.", *tag.Key, v, *tag.Value)
			}
		} else {
			t.Fatalf("Key %s not in map.", *tag.Key)
		}
	}
}

func TestDmsTagsFromMap(t *testing.T) {
	tagMap := map[string]interface{}{
		"test-key-1": "test-value-1",
		"test-key-2": "test-value-2",
	}

	result := dmsTagsFromMap(tagMap)

	for k, v := range tagMap {
		found := false
		for _, tag := range result {
			if k == *tag.Key {
				if v != *tag.Value {
					t.Fatalf("Key %s had value of %s. Expected %s.", k, v, *tag.Value)
				}
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Key %s not in tags.", k)
		}
	}
}

func TestDmsDiffTags(t *testing.T) {
	cases := []struct {
		o, n map[string]interface{}
		a, r map[string]string
	}{
		// basic add / remove
		{
			o: map[string]interface{}{"test-key-1": "test-value-1"},
			n: map[string]interface{}{"test-key-2": "test-value-2"},
			a: map[string]string{"test-key-2": "test-value-2"},
			r: map[string]string{"test-key-1": "test-value-1"},
		},
		// modify
		{
			o: map[string]interface{}{"test-key-1": "test-value-1"},
			n: map[string]interface{}{"test-key-1": "test-value-1-modified"},
			a: map[string]string{"test-key-1": "test-value-1-modified"},
			r: map[string]string{"test-key-1": "test-value-1"},
		},
	}

	for _, c := range cases {
		ar, rr := dmsDiffTags(dmsTagsFromMap(c.o), dmsTagsFromMap(c.n))
		a := dmsTagsToMap(ar)
		r := dmsTagsToMap(rr)

		if !reflect.DeepEqual(a, c.a) {
			t.Fatalf("Add tags mismatch: Actual %#v; Expected %#v", a, c.a)
		}
		if !reflect.DeepEqual(r, c.r) {
			t.Fatalf("Remove tags mismatch: Actual %#v; Expected %#v", r, c.r)
		}
	}
}

func TestDmsGetTagKeys(t *testing.T) {
	tags := []*dms.Tag{
		{
			Key:   aws.String("test-key-1"),
			Value: aws.String("test-value-1"),
		},
		{
			Key:   aws.String("test-key-2"),
			Value: aws.String("test-value-2"),
		},
	}

	result := dmsGetTagKeys(tags)
	expected := []*string{aws.String("test-key-1"), aws.String("test-key-2")}

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("Actual %s; Expected %s", aws.StringValueSlice(result), aws.StringValueSlice(expected))
	}
}
