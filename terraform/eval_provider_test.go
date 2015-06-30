package terraform

import (
	"reflect"
	"testing"
)

func TestEvalBuildProviderConfig_impl(t *testing.T) {
	var _ EvalNode = new(EvalBuildProviderConfig)
}

func TestEvalBuildProviderConfig(t *testing.T) {
	config := testResourceConfig(t, map[string]interface{}{})
	provider := "foo"

	n := &EvalBuildProviderConfig{
		Provider: provider,
		Config:   &config,
		Output:   &config,
	}

	ctx := &MockEvalContext{
		ParentProviderConfigConfig: testResourceConfig(t, map[string]interface{}{
			"foo": "bar",
		}),
		ProviderInputConfig: map[string]interface{}{
			"bar": "baz",
		},
	}
	if _, err := n.Eval(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := map[string]interface{}{
		"foo": "bar",
		"bar": "baz",
	}
	if !reflect.DeepEqual(config.Raw, expected) {
		t.Fatalf("bad: %#v", config.Raw)
	}
}

func TestEvalBuildProviderConfig_parentPriority(t *testing.T) {
	config := testResourceConfig(t, map[string]interface{}{})
	provider := "foo"

	n := &EvalBuildProviderConfig{
		Provider: provider,
		Config:   &config,
		Output:   &config,
	}

	ctx := &MockEvalContext{
		ParentProviderConfigConfig: testResourceConfig(t, map[string]interface{}{
			"foo": "bar",
		}),
		ProviderInputConfig: map[string]interface{}{
			"foo": "baz",
		},
	}
	if _, err := n.Eval(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := map[string]interface{}{
		"foo": "bar",
	}
	if !reflect.DeepEqual(config.Raw, expected) {
		t.Fatalf("bad: %#v", config.Raw)
	}
}

func TestEvalConfigProvider_impl(t *testing.T) {
	var _ EvalNode = new(EvalConfigProvider)
}

func TestEvalConfigProvider(t *testing.T) {
	config := testResourceConfig(t, map[string]interface{}{})
	provider := &MockResourceProvider{}
	n := &EvalConfigProvider{Config: &config}

	ctx := &MockEvalContext{ProviderProvider: provider}
	if _, err := n.Eval(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !ctx.ConfigureProviderCalled {
		t.Fatal("should be called")
	}
	if !reflect.DeepEqual(ctx.ConfigureProviderConfig, config) {
		t.Fatalf("bad: %#v", ctx.ConfigureProviderConfig)
	}
}

func TestEvalInitProvider_impl(t *testing.T) {
	var _ EvalNode = new(EvalInitProvider)
}

func TestEvalInitProvider(t *testing.T) {
	n := &EvalInitProvider{Name: "foo"}
	provider := &MockResourceProvider{}
	ctx := &MockEvalContext{InitProviderProvider: provider}
	if _, err := n.Eval(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !ctx.InitProviderCalled {
		t.Fatal("should be called")
	}
	if ctx.InitProviderName != "foo" {
		t.Fatalf("bad: %#v", ctx.InitProviderName)
	}
}

func TestEvalCloseProvider(t *testing.T) {
	n := &EvalCloseProvider{Name: "foo"}
	provider := &MockResourceProvider{}
	ctx := &MockEvalContext{CloseProviderProvider: provider}
	if _, err := n.Eval(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !ctx.CloseProviderCalled {
		t.Fatal("should be called")
	}
	if ctx.CloseProviderName != "foo" {
		t.Fatalf("bad: %#v", ctx.CloseProviderName)
	}
}

func TestEvalGetProvider_impl(t *testing.T) {
	var _ EvalNode = new(EvalGetProvider)
}

func TestEvalGetProvider(t *testing.T) {
	var actual ResourceProvider
	n := &EvalGetProvider{Name: "foo", Output: &actual}
	provider := &MockResourceProvider{}
	ctx := &MockEvalContext{ProviderProvider: provider}
	if _, err := n.Eval(ctx); err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != provider {
		t.Fatalf("bad: %#v", actual)
	}

	if !ctx.ProviderCalled {
		t.Fatal("should be called")
	}
	if ctx.ProviderName != "foo" {
		t.Fatalf("bad: %#v", ctx.ProviderName)
	}
}
