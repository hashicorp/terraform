package diff

import (
	"testing"
)

func TestLazyResourceMap(t *testing.T) {
	rb1 := new(ResourceBuilder)
	rb2 := new(ResourceBuilder)

	rm := &LazyResourceMap{
		Resources: map[string]ResourceBuilderFactory{
			"foo": testRBFactory(rb1),
			"bar": testRBFactory(rb2),
			"diff": func() *ResourceBuilder {
				return new(ResourceBuilder)
			},
		},
	}

	actual := rm.Get("foo")
	if actual == nil {
		t.Fatal("should not be nil")
	}
	if actual != rb1 {
		t.Fatalf("bad: %p %p", rb1, actual)
	}
	if actual == rm.Get("bar") {
		t.Fatalf("bad: %p %p", actual, rm.Get("bar"))
	}

	actual = rm.Get("diff")
	if actual == nil {
		t.Fatal("should not be nil")
	}
	if actual != rm.Get("diff") {
		t.Fatal("should memoize")
	}
}

func testRBFactory(rb *ResourceBuilder) ResourceBuilderFactory {
	return func() *ResourceBuilder {
		return rb
	}
}
