package terraform

import (
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
