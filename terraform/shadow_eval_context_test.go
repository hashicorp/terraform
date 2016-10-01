package terraform

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestShadowEvalContext_impl(t *testing.T) {
	var _ EvalContext = new(shadowEvalContextReal)
	var _ EvalContext = new(shadowEvalContextShadow)
}

func TestShadowEvalContextInitProvider(t *testing.T) {
	mock := new(MockEvalContext)
	real, shadow := NewShadowEvalContext(mock)

	// Args, results
	name := "foo"
	mockResult := new(MockResourceProvider)

	// Configure the mock
	mock.InitProviderProvider = mockResult

	// Verify that it blocks until the real func is called
	var result ResourceProvider
	var err error
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		result, err = shadow.InitProvider(name)
	}()

	select {
	case <-doneCh:
		t.Fatal("should block until finished")
	case <-time.After(10 * time.Millisecond):
	}

	// Call the real func
	realResult, realErr := real.InitProvider(name)
	if realErr != nil {
		t.Fatalf("bad: %#v", realErr)
	}
	realResult.Configure(nil)
	if !mockResult.ConfigureCalled {
		t.Fatalf("bad: %#v", realResult)
	}
	mockResult.ConfigureCalled = false

	// The shadow should finish now
	<-doneCh

	// Verify the shadow returned the same values
	if err != nil {
		t.Fatalf("bad: %#v", err)
	}

	// Verify that the returned value is a shadow. Calling one function
	// shouldn't affect the other.
	result.Configure(nil)
	if mockResult.ConfigureCalled {
		t.Fatal("close should not be called")
	}

	// And doing some work should result in that value
	mockErr := fmt.Errorf("yo")
	mockResult.ConfigureReturnError = mockErr
	realResult.Configure(nil)
	if err := result.Configure(nil); !reflect.DeepEqual(err, mockErr) {
		t.Fatalf("bad: %#v", err)
	}

	// Verify we have no errors
	if err := shadow.CloseShadow(); err != nil {
		t.Fatalf("bad: %s", err)
	}
}

func TestShadowEvalContextInitProvider_doubleInit(t *testing.T) {
	mock := new(MockEvalContext)
	real, shadow := NewShadowEvalContext(mock)

	// Args, results
	name := "foo"
	mockResult := new(MockResourceProvider)

	// Configure the mock
	mock.InitProviderProvider = mockResult

	// Call the real func
	real.InitProvider(name)

	// Get the provider twice
	shadow.InitProvider(name)
	p, err := shadow.InitProvider(name)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if p == nil {
		t.Fatal("should return provider")
	}

	if err := shadow.CloseShadow(); err == nil {
		t.Fatal("should error")
	}
}

func TestShadowEvalContextProvider(t *testing.T) {
	mock := new(MockEvalContext)
	real, shadow := NewShadowEvalContext(mock)

	// Args, results
	name := "foo"
	mockResult := new(MockResourceProvider)

	// Configure the mock
	mock.InitProviderProvider = mockResult

	// Call the real func
	real.InitProvider(name)
	shadow.InitProvider(name)

	// Get the provider twice
	p := shadow.Provider(name)
	if p == nil {
		t.Fatal("should return provider")
	}

	if err := shadow.CloseShadow(); err != nil {
		t.Fatalf("bad: %s", err)
	}
}

func TestShadowEvalContextProvider_noInit(t *testing.T) {
	mock := new(MockEvalContext)
	real, shadow := NewShadowEvalContext(mock)

	// Args, results
	name := "foo"
	mockResult := new(MockResourceProvider)

	// Configure the mock
	mock.InitProviderProvider = mockResult

	// Call the real func
	real.InitProvider(name)

	// Get the provider w/o calling init
	p := shadow.Provider(name)
	if p == nil {
		t.Fatal("should return provider")
	}

	if err := shadow.CloseShadow(); err == nil {
		t.Fatal("should error")
	}
}

func TestShadowEvalContextCloseProvider(t *testing.T) {
	mock := new(MockEvalContext)
	real, shadow := NewShadowEvalContext(mock)

	// Args, results
	name := "foo"
	mockResult := new(MockResourceProvider)

	// Configure the mock
	mock.InitProviderProvider = mockResult

	// Call the real func
	real.InitProvider(name)
	shadow.InitProvider(name)

	// Get the provider twice
	if err := shadow.CloseProvider(name); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := shadow.CloseShadow(); err != nil {
		t.Fatalf("bad: %s", err)
	}
}

func TestShadowEvalContextCloseProvider_doubleClose(t *testing.T) {
	mock := new(MockEvalContext)
	real, shadow := NewShadowEvalContext(mock)

	// Args, results
	name := "foo"
	mockResult := new(MockResourceProvider)

	// Configure the mock
	mock.InitProviderProvider = mockResult

	// Call the real func
	real.InitProvider(name)
	shadow.InitProvider(name)

	// Close the provider twice
	if err := shadow.CloseProvider(name); err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := shadow.CloseProvider(name); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := shadow.CloseShadow(); err == nil {
		t.Fatal("should error")
	}
}

func TestShadowEvalContextCloseProvider_noInitClose(t *testing.T) {
	mock := new(MockEvalContext)
	real, shadow := NewShadowEvalContext(mock)

	// Args, results
	name := "foo"
	mockResult := new(MockResourceProvider)

	// Configure the mock
	mock.InitProviderProvider = mockResult

	// Call the real func
	real.InitProvider(name)

	// Close the provider
	if err := shadow.CloseProvider(name); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := shadow.CloseShadow(); err == nil {
		t.Fatal("should error")
	}
}

func TestShadowEvalContextCloseProvider_noCreate(t *testing.T) {
	mock := new(MockEvalContext)
	_, shadow := NewShadowEvalContext(mock)

	// Args, results
	name := "foo"
	mockResult := new(MockResourceProvider)

	// Configure the mock
	mock.InitProviderProvider = mockResult

	// Close the provider
	if err := shadow.CloseProvider(name); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := shadow.CloseShadow(); err == nil {
		t.Fatal("should error")
	}
}
