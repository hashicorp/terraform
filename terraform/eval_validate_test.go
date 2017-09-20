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
	rc := testResourceConfig(t, map[string]interface{}{"foo": "bar"})
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
	rc := testResourceConfig(t, map[string]interface{}{"foo": "bar"})
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

func TestEvalValidateProvisioner_valid(t *testing.T) {
	mp := &MockResourceProvisioner{}
	var p ResourceProvisioner = mp
	ctx := &MockEvalContext{}

	cfg := &ResourceConfig{}
	connInfo, err := config.NewRawConfig(map[string]interface{}{})
	if err != nil {
		t.Fatalf("failed to make connInfo: %s", err)
	}
	connConfig := NewResourceConfig(connInfo)

	node := &EvalValidateProvisioner{
		Provisioner: &p,
		Config:      &cfg,
		ConnConfig:  &connConfig,
	}

	result, err := node.Eval(ctx)
	if err != nil {
		t.Fatalf("node.Eval failed: %s", err)
	}
	if result != nil {
		t.Errorf("node.Eval returned non-nil result")
	}

	if !mp.ValidateCalled {
		t.Fatalf("p.Config not called")
	}
	if mp.ValidateConfig != cfg {
		t.Errorf("p.Config called with wrong config")
	}
}

func TestEvalValidateProvisioner_warning(t *testing.T) {
	mp := &MockResourceProvisioner{}
	var p ResourceProvisioner = mp
	ctx := &MockEvalContext{}

	cfg := &ResourceConfig{}
	connInfo, err := config.NewRawConfig(map[string]interface{}{})
	if err != nil {
		t.Fatalf("failed to make connInfo: %s", err)
	}
	connConfig := NewResourceConfig(connInfo)

	node := &EvalValidateProvisioner{
		Provisioner: &p,
		Config:      &cfg,
		ConnConfig:  &connConfig,
	}

	mp.ValidateReturnWarns = []string{"foo is deprecated"}

	_, err = node.Eval(ctx)
	if err == nil {
		t.Fatalf("node.Eval succeeded; want error")
	}

	valErr, ok := err.(*EvalValidateError)
	if !ok {
		t.Fatalf("node.Eval error is %#v; want *EvalValidateError", valErr)
	}

	warns := valErr.Warnings
	if warns == nil || len(warns) != 1 {
		t.Fatalf("wrong number of warnings in %#v; want one warning", warns)
	}
	if warns[0] != mp.ValidateReturnWarns[0] {
		t.Fatalf("wrong warning %q; want %q", warns[0], mp.ValidateReturnWarns[0])
	}
}

func TestEvalValidateProvisioner_connectionInvalid(t *testing.T) {
	var p ResourceProvisioner = &MockResourceProvisioner{}
	ctx := &MockEvalContext{}

	cfg := &ResourceConfig{}
	connInfo, err := config.NewRawConfig(map[string]interface{}{
		"bananananananana": "foo",
		"bazaz":            "bar",
	})
	if err != nil {
		t.Fatalf("failed to make connInfo: %s", err)
	}
	connConfig := NewResourceConfig(connInfo)

	node := &EvalValidateProvisioner{
		Provisioner: &p,
		Config:      &cfg,
		ConnConfig:  &connConfig,
	}

	_, err = node.Eval(ctx)
	if err == nil {
		t.Fatalf("node.Eval succeeded; want error")
	}

	valErr, ok := err.(*EvalValidateError)
	if !ok {
		t.Fatalf("node.Eval error is %#v; want *EvalValidateError", valErr)
	}

	errs := valErr.Errors
	if errs == nil || len(errs) != 2 {
		t.Fatalf("wrong number of errors in %#v; want two errors", errs)
	}

	errStr := errs[0].Error()
	if !(strings.Contains(errStr, "bananananananana") || strings.Contains(errStr, "bazaz")) {
		t.Fatalf("wrong first error %q; want something about our invalid connInfo keys", errStr)
	}
}
