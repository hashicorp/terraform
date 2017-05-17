package terraform

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestShadowResourceProvisioner_impl(t *testing.T) {
	var _ Shadow = new(shadowResourceProvisionerShadow)
}

func TestShadowResourceProvisionerValidate(t *testing.T) {
	mock := new(MockResourceProvisioner)
	real, shadow := newShadowResourceProvisioner(mock)

	// Test values
	config := testResourceConfig(t, map[string]interface{}{
		"foo": "bar",
	})
	returnWarns := []string{"foo"}
	returnErrs := []error{fmt.Errorf("bar")}

	// Configure the mock
	mock.ValidateReturnWarns = returnWarns
	mock.ValidateReturnErrors = returnErrs

	// Verify that it blocks until the real func is called
	var warns []string
	var errs []error
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		warns, errs = shadow.Validate(config)
	}()

	select {
	case <-doneCh:
		t.Fatal("should block until finished")
	case <-time.After(10 * time.Millisecond):
	}

	// Call the real func
	realWarns, realErrs := real.Validate(config)
	if !reflect.DeepEqual(realWarns, returnWarns) {
		t.Fatalf("bad: %#v", realWarns)
	}
	if !reflect.DeepEqual(realErrs, returnErrs) {
		t.Fatalf("bad: %#v", realWarns)
	}

	// The shadow should finish now
	<-doneCh

	// Verify the shadow returned the same values
	if !reflect.DeepEqual(warns, returnWarns) {
		t.Fatalf("bad: %#v", warns)
	}
	if !reflect.DeepEqual(errs, returnErrs) {
		t.Fatalf("bad: %#v", errs)
	}

	// Verify we have no errors
	if err := shadow.CloseShadow(); err != nil {
		t.Fatalf("bad: %s", err)
	}
}

func TestShadowResourceProvisionerValidate_diff(t *testing.T) {
	mock := new(MockResourceProvisioner)
	real, shadow := newShadowResourceProvisioner(mock)

	// Test values
	config := testResourceConfig(t, map[string]interface{}{
		"foo": "bar",
	})
	returnWarns := []string{"foo"}
	returnErrs := []error{fmt.Errorf("bar")}

	// Configure the mock
	mock.ValidateReturnWarns = returnWarns
	mock.ValidateReturnErrors = returnErrs

	// Run a real validation with a config
	real.Validate(testResourceConfig(t, map[string]interface{}{"bar": "baz"}))

	// Verify that it blocks until the real func is called
	var warns []string
	var errs []error
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		warns, errs = shadow.Validate(config)
	}()

	select {
	case <-doneCh:
		t.Fatal("should block until finished")
	case <-time.After(10 * time.Millisecond):
	}

	// Call the real func
	realWarns, realErrs := real.Validate(config)
	if !reflect.DeepEqual(realWarns, returnWarns) {
		t.Fatalf("bad: %#v", realWarns)
	}
	if !reflect.DeepEqual(realErrs, returnErrs) {
		t.Fatalf("bad: %#v", realWarns)
	}

	// The shadow should finish now
	<-doneCh

	// Verify the shadow returned the same values
	if !reflect.DeepEqual(warns, returnWarns) {
		t.Fatalf("bad: %#v", warns)
	}
	if !reflect.DeepEqual(errs, returnErrs) {
		t.Fatalf("bad: %#v", errs)
	}

	// Verify we have no errors
	if err := shadow.CloseShadow(); err != nil {
		t.Fatalf("bad: %s", err)
	}
}

func TestShadowResourceProvisionerApply(t *testing.T) {
	mock := new(MockResourceProvisioner)
	real, shadow := newShadowResourceProvisioner(mock)

	// Test values
	output := new(MockUIOutput)
	state := &InstanceState{ID: "foo"}
	config := testResourceConfig(t, map[string]interface{}{"foo": "bar"})
	mockReturn := errors.New("err")

	// Configure the mock
	mock.ApplyReturnError = mockReturn

	// Verify that it blocks until the real func is called
	var err error
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		err = shadow.Apply(output, state, config)
	}()

	select {
	case <-doneCh:
		t.Fatal("should block until finished")
	case <-time.After(10 * time.Millisecond):
	}

	// Call the real func
	realErr := real.Apply(output, state, config)
	if realErr != mockReturn {
		t.Fatalf("bad: %#v", realErr)
	}

	// The shadow should finish now
	<-doneCh

	// Verify the shadow returned the same values
	if err != mockReturn {
		t.Errorf("bad: %#v", err)
	}

	// Verify we have no errors
	if err := shadow.CloseShadow(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	if err := shadow.ShadowError(); err != nil {
		t.Fatalf("bad: %s", err)
	}
}
