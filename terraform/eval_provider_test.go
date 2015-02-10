package terraform

import (
	"reflect"
	"testing"
)

func TestEvalConfigProvider_impl(t *testing.T) {
	var _ EvalNode = new(EvalConfigProvider)
}

func TestEvalConfigProvider(t *testing.T) {
	config := testResourceConfig(t, map[string]interface{}{})
	provider := &MockResourceProvider{}
	n := &EvalConfigProvider{}

	ctx := &MockEvalContext{ProviderProvider: provider}
	args := []interface{}{config}
	if actual, err := n.Eval(ctx, args); err != nil {
		t.Fatalf("err: %s", err)
	} else if actual != nil {
		t.Fatalf("bad: %#v", actual)
	}

	if !ctx.ConfigureProviderCalled {
		t.Fatal("should be called")
	}
	if !reflect.DeepEqual(ctx.ConfigureProviderConfig, config) {
		t.Fatalf("bad: %#v", ctx.ConfigureProviderConfig)
	}
}

func TestEvalConfigProvider_args(t *testing.T) {
	config := testResourceConfig(t, map[string]interface{}{})
	configNode := &EvalLiteral{Value: config}
	n := &EvalConfigProvider{Provider: "foo", Config: configNode}

	args, types := n.Args()
	expectedArgs := []EvalNode{configNode}
	expectedTypes := []EvalType{EvalTypeConfig}
	if !reflect.DeepEqual(args, expectedArgs) {
		t.Fatalf("bad: %#v", args)
	}
	if !reflect.DeepEqual(types, expectedTypes) {
		t.Fatalf("bad: %#v", args)
	}
}

func TestEvalInitProvider_impl(t *testing.T) {
	var _ EvalNode = new(EvalInitProvider)
}

func TestEvalInitProvider(t *testing.T) {
	n := &EvalInitProvider{Name: "foo"}
	provider := &MockResourceProvider{}
	ctx := &MockEvalContext{InitProviderProvider: provider}
	if actual, err := n.Eval(ctx, nil); err != nil {
		t.Fatalf("err: %s", err)
	} else if actual != provider {
		t.Fatalf("bad: %#v", actual)
	}

	if !ctx.InitProviderCalled {
		t.Fatal("should be called")
	}
	if ctx.InitProviderName != "foo" {
		t.Fatalf("bad: %#v", ctx.InitProviderName)
	}
}

func TestEvalGetProvider_impl(t *testing.T) {
	var _ EvalNode = new(EvalGetProvider)
}

func TestEvalGetProvider(t *testing.T) {
	n := &EvalGetProvider{Name: "foo"}
	provider := &MockResourceProvider{}
	ctx := &MockEvalContext{ProviderProvider: provider}
	if actual, err := n.Eval(ctx, nil); err != nil {
		t.Fatalf("err: %s", err)
	} else if actual != provider {
		t.Fatalf("bad: %#v", actual)
	}

	if !ctx.ProviderCalled {
		t.Fatal("should be called")
	}
	if ctx.ProviderName != "foo" {
		t.Fatalf("bad: %#v", ctx.ProviderName)
	}
}
