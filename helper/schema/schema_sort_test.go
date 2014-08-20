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
		List: []interface{}{5, 2, 1, 3, 4},
		Schema: &Schema{
			Order: func(a, b interface{}) bool {
				return a.(int) < b.(int)
			},
		},
	}

	sort.Sort(s)

	expected := []interface{}{1, 2, 3, 4, 5}
	if !reflect.DeepEqual(s.List, expected) {
		t.Fatalf("bad: %#v", s.List)
	}

	expectedMap := map[int]int{
		0: 2,
		1: 1,
		2: 3,
		3: 4,
		4: 0,
	}
	if !reflect.DeepEqual(s.Map, expectedMap) {
		t.Fatalf("bad: %#v", s.Map)
	}
}
