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
}
