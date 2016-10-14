package aws

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/hashicorp/terraform/helper/schema"
)

func TestBuildEC2AttributeFilterList(t *testing.T) {
	type TestCase struct {
		Attrs    map[string]string
		Expected []*ec2.Filter
	}
	testCases := []TestCase{
		{
			map[string]string{
				"foo": "bar",
				"baz": "boo",
			},
			[]*ec2.Filter{
				{
					Name:   aws.String("baz"),
					Values: []*string{aws.String("boo")},
				},
				{
					Name:   aws.String("foo"),
					Values: []*string{aws.String("bar")},
				},
			},
		},
		{
			map[string]string{
				"foo": "bar",
				"baz": "",
			},
			[]*ec2.Filter{
				{
					Name:   aws.String("foo"),
					Values: []*string{aws.String("bar")},
				},
			},
		},
	}

	for i, testCase := range testCases {
		result := buildEC2AttributeFilterList(testCase.Attrs)

		if !reflect.DeepEqual(result, testCase.Expected) {
			t.Errorf(
				"test case %d: got %#v, but want %#v",
				i, result, testCase.Expected,
			)
		}
	}
}

func TestBuildEC2TagFilterList(t *testing.T) {
	type TestCase struct {
		Tags     []*ec2.Tag
		Expected []*ec2.Filter
	}
	testCases := []TestCase{
		{
			[]*ec2.Tag{
				{
					Key:   aws.String("foo"),
					Value: aws.String("bar"),
				},
				{
					Key:   aws.String("baz"),
					Value: aws.String("boo"),
				},
			},
			[]*ec2.Filter{
				{
					Name:   aws.String("tag:foo"),
					Values: []*string{aws.String("bar")},
				},
				{
					Name:   aws.String("tag:baz"),
					Values: []*string{aws.String("boo")},
				},
			},
		},
	}

	for i, testCase := range testCases {
		result := buildEC2TagFilterList(testCase.Tags)

		if !reflect.DeepEqual(result, testCase.Expected) {
			t.Errorf(
				"test case %d: got %#v, but want %#v",
				i, result, testCase.Expected,
			)
		}
	}
}

func TestBuildEC2CustomFilterList(t *testing.T) {

	// We need to get a set with the appropriate hash function,
	// so we'll use the schema to help us produce what would
	// be produced in the normal case.
	filtersSchema := ec2CustomFiltersSchema()

	// The zero value of this schema will be an interface{}
	// referring to a new, empty *schema.Set with the
	// appropriate hash function configured.
	filters := filtersSchema.ZeroValue().(*schema.Set)

	// We also need an appropriately-configured set for
	// the list of values.
	valuesSchema := filtersSchema.Elem.(*schema.Resource).Schema["values"]
	valuesSet := func(vals ...string) *schema.Set {
		ret := valuesSchema.ZeroValue().(*schema.Set)
		for _, val := range vals {
			ret.Add(val)
		}
		return ret
	}

	filters.Add(map[string]interface{}{
		"name":   "foo",
		"values": valuesSet("bar", "baz"),
	})
	filters.Add(map[string]interface{}{
		"name":   "pizza",
		"values": valuesSet("cheese"),
	})

	expected := []*ec2.Filter{
		// These are produced in the deterministic order guaranteed
		// by schema.Set.List(), which happens to produce them in
		// the following order for our current input. If this test
		// evolves with different input data in future then they
		// will likely be emitted in a different order, which is fine.
		{
			Name:   aws.String("pizza"),
			Values: []*string{aws.String("cheese")},
		},
		{
			Name:   aws.String("foo"),
			Values: []*string{aws.String("bar"), aws.String("baz")},
		},
	}
	result := buildEC2CustomFilterList(filters)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf(
			"got %#v, but want %#v",
			result, expected,
		)
	}
}
