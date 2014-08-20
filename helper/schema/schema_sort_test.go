package schema

import (
	"reflect"
	"sort"
	"testing"
)

func TestListSort_impl(t *testing.T) {
	var _ sort.Interface = new(listSort)
}

func TestListSort(t *testing.T) {
	s := &listSort{
		List: []interface{}{5, 2, 1},
		Schema: &Schema{
			Order: func(a, b interface{}) bool {
				return a.(int) < b.(int)
			},
		},
	}

	sort.Sort(s)

	expected := []interface{}{1, 2, 5}
	if !reflect.DeepEqual(s.List, expected) {
		t.Fatalf("bad: %#v", s.List)
	}
}
