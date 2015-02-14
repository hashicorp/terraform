package terraform

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config"
)

func TestEvalInterpolate_impl(t *testing.T) {
	var _ EvalNode = new(EvalInterpolate)
}

func TestEvalInterpolate(t *testing.T) {
	config, err := config.NewRawConfig(map[string]interface{}{})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	var actual *ResourceConfig
	n := &EvalInterpolate{Config: config, Output: &actual}
	result := testResourceConfig(t, map[string]interface{}{})
	ctx := &MockEvalContext{InterpolateConfigResult: result}
	if _, err := n.Eval(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != result {
		t.Fatalf("bad: %#v", actual)
	}

	if !ctx.InterpolateCalled {
		t.Fatal("should be called")
	}
	if !reflect.DeepEqual(ctx.InterpolateConfig, config) {
		t.Fatalf("bad: %#v", ctx.InterpolateConfig)
	}
}
