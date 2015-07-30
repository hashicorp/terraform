package terraform

import (
	"reflect"
	"sync"
	"testing"
)

func TestBuiltinEvalContextProviderInput(t *testing.T) {
	var lock sync.Mutex
	cache := make(map[string]map[string]interface{})

	ctx1 := testBuiltinEvalContext(t)
	ctx1.PathValue = []string{"root"}
	ctx1.ProviderInputConfig = cache
	ctx1.ProviderLock = &lock

	ctx2 := testBuiltinEvalContext(t)
	ctx2.PathValue = []string{"root", "child"}
	ctx2.ProviderInputConfig = cache
	ctx2.ProviderLock = &lock

	expected1 := map[string]interface{}{"value": "foo"}
	ctx1.SetProviderInput("foo", expected1)

	expected2 := map[string]interface{}{"value": "bar"}
	ctx2.SetProviderInput("foo", expected2)

	actual1 := ctx1.ProviderInput("foo")
	actual2 := ctx2.ProviderInput("foo")

	if !reflect.DeepEqual(actual1, expected1) {
		t.Fatalf("bad: %#v %#v", actual1, expected1)
	}
	if !reflect.DeepEqual(actual2, expected2) {
		t.Fatalf("bad: %#v %#v", actual2, expected2)
	}
}

func testBuiltinEvalContext(t *testing.T) *BuiltinEvalContext {
	return &BuiltinEvalContext{}
}
