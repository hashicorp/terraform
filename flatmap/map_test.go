package flatmap

import (
	"reflect"
	"testing"
)

func TestMapDelete(t *testing.T) {
	m := Flatten(map[string]interface{}{
		"foo": "bar",
		"routes": []map[string]string{
			map[string]string{
				"foo": "bar",
			},
		},
	})

	m.Delete("routes")

	expected := Map(map[string]string{"foo": "bar"})
	if !reflect.DeepEqual(m, expected) {
		t.Fatalf("bad: %#v", m)
	}
}
