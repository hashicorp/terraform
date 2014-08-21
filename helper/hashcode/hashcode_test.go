package hashcode

import (
	"testing"
)

func TestString(t *testing.T) {
	v := "hello, world"
	expected := String(v)
	for i := 0; i < 100; i++ {
		actual := String(v)
		if actual != expected {
			t.Fatalf("bad: %#v", actual)
		}
	}
}
