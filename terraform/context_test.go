package terraform

import (
	"fmt"
	"reflect"
	"testing"
)

func TestContextValidate(t *testing.T) {
	config := testConfig(t, "validate-good")
	c := testContext(t, &ContextOpts{
		Config: config,
	})

	w, e := c.Validate()
	if len(w) > 0 {
		t.Fatalf("bad: %#v", w)
	}
	if len(e) > 0 {
		t.Fatalf("bad: %#v", e)
	}
}

func TestContextValidate_badVar(t *testing.T) {
	config := testConfig(t, "validate-bad-var")
	c := testContext(t, &ContextOpts{
		Config: config,
	})

	w, e := c.Validate()
	if len(w) > 0 {
		t.Fatalf("bad: %#v", w)
	}
	if len(e) == 0 {
		t.Fatalf("bad: %#v", e)
	}
}

func TestContextValidate_requiredVar(t *testing.T) {
	config := testConfig(t, "validate-required-var")
	c := testContext(t, &ContextOpts{
		Config: config,
	})

	w, e := c.Validate()
	if len(w) > 0 {
		t.Fatalf("bad: %#v", w)
	}
	if len(e) == 0 {
		t.Fatalf("bad: %#v", e)
	}
}

func TestContextRefresh(t *testing.T) {
	p := testProvider("aws")
	c := testConfig(t, "refresh-basic")
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	p.RefreshFn = nil
	p.RefreshReturn = &ResourceState{
		ID: "foo",
	}

	s, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if !p.RefreshCalled {
		t.Fatal("refresh should be called")
	}
	if p.RefreshState.ID != "" {
		t.Fatalf("bad: %#v", p.RefreshState)
	}
	if !reflect.DeepEqual(s.Resources["aws_instance.web"], p.RefreshReturn) {
		t.Fatalf("bad: %#v", s.Resources["aws_instance.web"])
	}

	for _, r := range s.Resources {
		if r.Type == "" {
			t.Fatalf("no type: %#v", r)
		}
	}
}

func TestContextRefresh_hook(t *testing.T) {
	h := new(MockHook)
	p := testProvider("aws")
	c := testConfig(t, "refresh-basic")
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Hooks:  []Hook{h},
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	if _, err := ctx.Refresh(); err != nil {
		t.Fatalf("err: %s", err)
	}
	if !h.PreRefreshCalled {
		t.Fatal("should be called")
	}
	if h.PreRefreshState.Type != "aws_instance" {
		t.Fatalf("bad: %#v", h.PreRefreshState)
	}
	if !h.PostRefreshCalled {
		t.Fatal("should be called")
	}
	if h.PostRefreshState.Type != "aws_instance" {
		t.Fatalf("bad: %#v", h.PostRefreshState)
	}
}

func TestContextRefresh_state(t *testing.T) {
	p := testProvider("aws")
	c := testConfig(t, "refresh-basic")
	state := &State{
		Resources: map[string]*ResourceState{
			"aws_instance.web": &ResourceState{
				ID: "bar",
			},
		},
	}
	ctx := testContext(t, &ContextOpts{
		Config: c,
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
		State: state,
	})

	p.RefreshFn = nil
	p.RefreshReturn = &ResourceState{
		ID: "foo",
	}

	s, err := ctx.Refresh()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if !p.RefreshCalled {
		t.Fatal("refresh should be called")
	}
	if !reflect.DeepEqual(p.RefreshState, state.Resources["aws_instance.web"]) {
		t.Fatalf("bad: %#v", p.RefreshState)
	}
	if !reflect.DeepEqual(s.Resources["aws_instance.web"], p.RefreshReturn) {
		t.Fatalf("bad: %#v", s.Resources)
	}
}

func testContext(t *testing.T, opts *ContextOpts) *Context {
	return NewContext(opts)
}

func testProvider(prefix string) *MockResourceProvider {
	p := new(MockResourceProvider)
	p.RefreshFn = func(s *ResourceState) (*ResourceState, error) {
		return s, nil
	}
	p.ResourcesReturn = []ResourceType{
		ResourceType{
			Name: fmt.Sprintf("%s_instance", prefix),
		},
	}

	return p
}
