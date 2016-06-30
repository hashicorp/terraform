package terraform

import (
	"errors"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/config"
)

func TestEvalValidateResource_managedResource(t *testing.T) {
	mp := testProvider("aws")
	mp.ValidateResourceFn = func(rt string, c *ResourceConfig) (ws []string, es []error) {
		expected := "aws_instance"
		if rt != expected {
			t.Fatalf("expected: %s, got: %s", expected, rt)
		}
		expected = "bar"
		val, _ := c.Get("foo")
		if val != expected {
			t.Fatalf("expected: %s, got: %s", expected, val)
		}
		return
	}

	p := ResourceProvider(mp)
	rc := &ResourceConfig{
		Raw: map[string]interface{}{"foo": "bar"},
	}
	node := &EvalValidateResource{
		Provider:     &p,
		Config:       &rc,
		ResourceName: "foo",
		ResourceType: "aws_instance",
		ResourceMode: config.ManagedResourceMode,
	}

	_, err := node.Eval(&MockEvalContext{})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !mp.ValidateResourceCalled {
		t.Fatal("Expected ValidateResource to be called, but it was not!")
	}
}

func TestEvalValidateResource_dataSource(t *testing.T) {
	mp := testProvider("aws")
	mp.ValidateDataSourceFn = func(rt string, c *ResourceConfig) (ws []string, es []error) {
		expected := "aws_ami"
		if rt != expected {
			t.Fatalf("expected: %s, got: %s", expected, rt)
		}
		expected = "bar"
		val, _ := c.Get("foo")
		if val != expected {
			t.Fatalf("expected: %s, got: %s", expected, val)
		}
		return
	}

	p := ResourceProvider(mp)
	rc := &ResourceConfig{
		Raw: map[string]interface{}{"foo": "bar"},
	}
	node := &EvalValidateResource{
		Provider:     &p,
		Config:       &rc,
		ResourceName: "foo",
		ResourceType: "aws_ami",
		ResourceMode: config.DataResourceMode,
	}

	_, err := node.Eval(&MockEvalContext{})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !mp.ValidateDataSourceCalled {
		t.Fatal("Expected ValidateDataSource to be called, but it was not!")
	}
}

func TestEvalValidateResource_validReturnsNilError(t *testing.T) {
	mp := testProvider("aws")
	mp.ValidateResourceFn = func(rt string, c *ResourceConfig) (ws []string, es []error) {
		return
	}

	p := ResourceProvider(mp)
	rc := &ResourceConfig{}
	node := &EvalValidateResource{
		Provider:     &p,
		Config:       &rc,
		ResourceName: "foo",
		ResourceType: "aws_instance",
		ResourceMode: config.ManagedResourceMode,
	}

	_, err := node.Eval(&MockEvalContext{})
	if err != nil {
		t.Fatalf("Expected nil error, got: %s", err)
	}
}

func TestEvalValidateResource_warningsAndErrorsPassedThrough(t *testing.T) {
	mp := testProvider("aws")
	mp.ValidateResourceFn = func(rt string, c *ResourceConfig) (ws []string, es []error) {
		ws = append(ws, "warn")
		es = append(es, errors.New("err"))
		return
	}

	p := ResourceProvider(mp)
	rc := &ResourceConfig{}
	node := &EvalValidateResource{
		Provider:     &p,
		Config:       &rc,
		ResourceName: "foo",
		ResourceType: "aws_instance",
		ResourceMode: config.ManagedResourceMode,
	}

	_, err := node.Eval(&MockEvalContext{})
	if err == nil {
		t.Fatal("Expected an error, got none!")
	}

	verr := err.(*EvalValidateError)
	if len(verr.Warnings) != 1 || verr.Warnings[0] != "warn" {
		t.Fatalf("Expected 1 warning 'warn', got: %#v", verr.Warnings)
	}
	if len(verr.Errors) != 1 || verr.Errors[0].Error() != "err" {
		t.Fatalf("Expected 1 error 'err', got: %#v", verr.Errors)
	}
}

func TestEvalValidateResource_checksResourceName(t *testing.T) {
	mp := testProvider("aws")
	p := ResourceProvider(mp)
	rc := &ResourceConfig{}
	node := &EvalValidateResource{
		Provider:     &p,
		Config:       &rc,
		ResourceName: "bad*name",
		ResourceType: "aws_instance",
		ResourceMode: config.ManagedResourceMode,
	}

	_, err := node.Eval(&MockEvalContext{})
	if err == nil {
		t.Fatal("Expected an error, got none!")
	}
	expectErr := "resource name can only contain"
	if !strings.Contains(err.Error(), expectErr) {
		t.Fatalf("Expected err: %s to contain %s", err, expectErr)
	}
}

func TestEvalValidateResource_ignoreWarnings(t *testing.T) {
	mp := testProvider("aws")
	mp.ValidateResourceFn = func(rt string, c *ResourceConfig) (ws []string, es []error) {
		ws = append(ws, "warn")
		return
	}

	p := ResourceProvider(mp)
	rc := &ResourceConfig{}
	node := &EvalValidateResource{
		Provider:     &p,
		Config:       &rc,
		ResourceName: "foo",
		ResourceType: "aws_instance",
		ResourceMode: config.ManagedResourceMode,

		IgnoreWarnings: true,
	}

	_, err := node.Eval(&MockEvalContext{})
	if err != nil {
		t.Fatalf("Expected no error, got: %s", err)
	}
}
