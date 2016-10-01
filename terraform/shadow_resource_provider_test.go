package terraform

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestShadowResourceProvider_cachedValues(t *testing.T) {
	mock := new(MockResourceProvider)
	real, shadow := newShadowResourceProvider(mock)

	// Resources
	{
		actual := shadow.Resources()
		expected := real.Resources()
		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("bad:\n\n%#v\n\n%#v", actual, expected)
		}
	}

	// DataSources
	{
		actual := shadow.DataSources()
		expected := real.DataSources()
		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("bad:\n\n%#v\n\n%#v", actual, expected)
		}
	}
}

func TestShadowResourceProviderInput(t *testing.T) {
	mock := new(MockResourceProvider)
	real, shadow := newShadowResourceProvider(mock)

	// Test values
	ui := new(MockUIInput)
	config := testResourceConfig(t, map[string]interface{}{
		"foo": "bar",
	})
	returnConfig := testResourceConfig(t, map[string]interface{}{
		"bar": "baz",
	})

	// Configure the mock
	mock.InputReturnConfig = returnConfig

	// Verify that it blocks until the real input is called
	var actual *ResourceConfig
	var err error
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		actual, err = shadow.Input(ui, config)
	}()

	select {
	case <-doneCh:
		t.Fatal("should block until finished")
	case <-time.After(10 * time.Millisecond):
	}

	// Call the real input
	realResult, realErr := real.Input(ui, config)
	if !realResult.Equal(returnConfig) {
		t.Fatalf("bad: %#v", realResult)
	}
	if realErr != nil {
		t.Fatalf("bad: %s", realErr)
	}

	// The shadow should finish now
	<-doneCh

	// Verify the shadow returned the same values
	if !actual.Equal(returnConfig) {
		t.Fatalf("bad: %#v", actual)
	}
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify we have no errors
	if err := shadow.CloseShadow(); err != nil {
		t.Fatalf("bad: %s", err)
	}
}

func TestShadowResourceProviderInput_badInput(t *testing.T) {
	mock := new(MockResourceProvider)
	real, shadow := newShadowResourceProvider(mock)

	// Test values
	ui := new(MockUIInput)
	config := testResourceConfig(t, map[string]interface{}{
		"foo": "bar",
	})
	configBad := testResourceConfig(t, map[string]interface{}{
		"foo": "nope",
	})

	// Call the real with one
	real.Input(ui, config)

	// Call the shadow with another
	_, err := shadow.Input(ui, configBad)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Verify we have an error
	if err := shadow.CloseShadow(); err == nil {
		t.Fatal("should have error")
	}
}

func TestShadowResourceProviderValidate(t *testing.T) {
	mock := new(MockResourceProvider)
	real, shadow := newShadowResourceProvider(mock)

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

func TestShadowResourceProviderValidate_badInput(t *testing.T) {
	mock := new(MockResourceProvider)
	real, shadow := newShadowResourceProvider(mock)

	// Test values
	config := testResourceConfig(t, map[string]interface{}{
		"foo": "bar",
	})
	configBad := testResourceConfig(t, map[string]interface{}{
		"foo": "nope",
	})

	// Call the real with one
	real.Validate(config)

	// Call the shadow with another
	shadow.Validate(configBad)

	// Verify we have an error
	if err := shadow.CloseShadow(); err == nil {
		t.Fatal("should have error")
	}
}

func TestShadowResourceProviderConfigure(t *testing.T) {
	mock := new(MockResourceProvider)
	real, shadow := newShadowResourceProvider(mock)

	// Test values
	config := testResourceConfig(t, map[string]interface{}{
		"foo": "bar",
	})
	returnErr := fmt.Errorf("bar")

	// Configure the mock
	mock.ConfigureReturnError = returnErr

	// Verify that it blocks until the real func is called
	var err error
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		err = shadow.Configure(config)
	}()

	select {
	case <-doneCh:
		t.Fatal("should block until finished")
	case <-time.After(10 * time.Millisecond):
	}

	// Call the real func
	realErr := real.Configure(config)
	if !reflect.DeepEqual(realErr, returnErr) {
		t.Fatalf("bad: %#v", realErr)
	}

	// The shadow should finish now
	<-doneCh

	// Verify the shadow returned the same values
	if !reflect.DeepEqual(err, returnErr) {
		t.Fatalf("bad: %#v", err)
	}

	// Verify we have no errors
	if err := shadow.CloseShadow(); err != nil {
		t.Fatalf("bad: %s", err)
	}
}

func TestShadowResourceProviderConfigure_badInput(t *testing.T) {
	mock := new(MockResourceProvider)
	real, shadow := newShadowResourceProvider(mock)

	// Test values
	config := testResourceConfig(t, map[string]interface{}{
		"foo": "bar",
	})
	configBad := testResourceConfig(t, map[string]interface{}{
		"foo": "nope",
	})

	// Call the real with one
	real.Configure(config)

	// Call the shadow with another
	shadow.Configure(configBad)

	// Verify we have an error
	if err := shadow.CloseShadow(); err == nil {
		t.Fatal("should have error")
	}
}

func TestShadowResourceProviderApply(t *testing.T) {
	mock := new(MockResourceProvider)
	real, shadow := newShadowResourceProvider(mock)

	// Test values
	info := &InstanceInfo{Id: "foo"}
	state := &InstanceState{ID: "foo"}
	diff := &InstanceDiff{Destroy: true}
	mockResult := &InstanceState{ID: "bar"}

	// Configure the mock
	mock.ApplyReturn = mockResult

	// Verify that it blocks until the real func is called
	var result *InstanceState
	var err error
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		result, err = shadow.Apply(info, state, diff)
	}()

	select {
	case <-doneCh:
		t.Fatal("should block until finished")
	case <-time.After(10 * time.Millisecond):
	}

	// Call the real func
	realResult, realErr := real.Apply(info, state, diff)
	if !realResult.Equal(mockResult) {
		t.Fatalf("bad: %#v", realResult)
	}
	if realErr != nil {
		t.Fatalf("bad: %#v", realErr)
	}

	// The shadow should finish now
	<-doneCh

	// Verify the shadow returned the same values
	if !result.Equal(mockResult) {
		t.Fatalf("bad: %#v", result)
	}
	if err != nil {
		t.Fatalf("bad: %#v", err)
	}

	// Verify we have no errors
	if err := shadow.CloseShadow(); err != nil {
		t.Fatalf("bad: %s", err)
	}
}
