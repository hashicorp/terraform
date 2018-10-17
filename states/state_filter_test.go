package states

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/addrs"
)

func TestFilterFilter(t *testing.T) {
	cases := map[string]struct {
		State    *State
		Filters  []string
		Expected []string
	}{
		"all": {
			testStateSmall(),
			[]string{},
			[]string{
				"*states.Resource: aws_key_pair.onprem",
				"*states.ResourceInstance: aws_key_pair.onprem",
				"*states.Module: module.boot",
				"*states.Resource: module.boot.aws_route53_record.oasis-consul-boot-a",
				"*states.ResourceInstance: module.boot.aws_route53_record.oasis-consul-boot-a",
				"*states.Resource: module.boot.aws_route53_record.oasis-consul-boot-ns",
				"*states.ResourceInstance: module.boot.aws_route53_record.oasis-consul-boot-ns",
				"*states.Resource: module.boot.aws_route53_zone.oasis-consul-boot",
				"*states.ResourceInstance: module.boot.aws_route53_zone.oasis-consul-boot",
			},
		},

		"single resource": {
			testStateSmall(),
			[]string{"aws_key_pair.onprem"},
			[]string{
				"*states.Resource: aws_key_pair.onprem",
				"*states.ResourceInstance: aws_key_pair.onprem",
			},
		},

		"single resource from minimal state": {
			testStateSingleMinimal(),
			[]string{"aws_instance.web"},
			[]string{
				"*states.Resource: aws_instance.web",
				"*states.ResourceInstance: aws_instance.web",
			},
		},

		"single resource with similar names": {
			testStateSmallTestInstance(),
			[]string{"test_instance.foo"},
			[]string{
				"*states.Resource: test_instance.foo",
				"*states.ResourceInstance: test_instance.foo",
			},
		},

		"module filter": {
			testStateComplete(),
			[]string{"module.boot"},
			[]string{
				"*states.Module: module.boot",
				"*states.Resource: module.boot.aws_route53_record.oasis-consul-boot-a",
				"*states.ResourceInstance: module.boot.aws_route53_record.oasis-consul-boot-a",
				"*states.Resource: module.boot.aws_route53_record.oasis-consul-boot-ns",
				"*states.ResourceInstance: module.boot.aws_route53_record.oasis-consul-boot-ns",
				"*states.Resource: module.boot.aws_route53_zone.oasis-consul-boot",
				"*states.ResourceInstance: module.boot.aws_route53_zone.oasis-consul-boot",
			},
		},

		"resource in module": {
			testStateComplete(),
			[]string{"module.boot.aws_route53_zone.oasis-consul-boot"},
			[]string{
				"*states.Resource: module.boot.aws_route53_zone.oasis-consul-boot",
				"*states.ResourceInstance: module.boot.aws_route53_zone.oasis-consul-boot",
			},
		},

		"resource in module 2": {
			testStateResourceInModule(),
			[]string{"module.foo.aws_instance.foo"},
			[]string{},
		},

		"single count index": {
			testStateComplete(),
			[]string{"module.consul.aws_instance.consul-green[0]"},
			[]string{
				"*states.ResourceInstance: module.consul.aws_instance.consul-green[0]",
			},
		},

		"no count index": {
			testStateComplete(),
			[]string{"module.consul.aws_instance.consul-green"},
			[]string{
				"*states.Resource: module.consul.aws_instance.consul-green",
				"*states.ResourceInstance: module.consul.aws_instance.consul-green[0]",
				"*states.ResourceInstance: module.consul.aws_instance.consul-green[1]",
				"*states.ResourceInstance: module.consul.aws_instance.consul-green[2]",
			},
		},

		"nested modules": {
			testStateNestedModules(),
			[]string{"module.outer"},
			[]string{
				"*states.Module: module.outer",
				"*states.Module: module.outer.module.child1",
				"*states.Resource: module.outer.module.child1.aws_instance.foo",
				"*states.ResourceInstance: module.outer.module.child1.aws_instance.foo",
				"*states.Module: module.outer.module.child2",
				"*states.Resource: module.outer.module.child2.aws_instance.foo",
				"*states.ResourceInstance: module.outer.module.child2.aws_instance.foo",
			},
		},
	}

	for n, tc := range cases {
		// Create the filter
		filter := &Filter{State: tc.State}

		// Filter!
		results, err := filter.Filter(tc.Filters...)
		if err != nil {
			t.Fatalf("%q: err: %s", n, err)
		}

		actual := make([]string, len(results))
		for i, result := range results {
			actual[i] = result.String()
		}

		if !reflect.DeepEqual(actual, tc.Expected) {
			t.Fatalf("%q: expected, then actual\n\n%#v\n\n%#v", n, tc.Expected, actual)
		}
	}
}

// testStateComplete returns a test State structure.
func testStateComplete() *State {
	root := addrs.RootModuleInstance
	boot, _ := addrs.ParseModuleInstanceStr("module.boot")
	consul, _ := addrs.ParseModuleInstanceStr("module.consul")
	vault, _ := addrs.ParseModuleInstanceStr("module.vault")

	return BuildState(func(s *SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_key_pair",
				Name: "onprem",
			}.Instance(addrs.NoKey).Absolute(root),
			&ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id": "1234567890"}`),
				Status:    ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(root),
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_route53_record",
				Name: "oasis-consul-boot-a",
			}.Instance(addrs.NoKey).Absolute(boot),
			&ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id": "1234567890"}`),
				Status:    ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(boot),
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_route53_record",
				Name: "oasis-consul-boot-ns",
			}.Instance(addrs.NoKey).Absolute(boot),
			&ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id": "1234567890"}`),
				Status:    ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(boot),
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_route53_zone",
				Name: "oasis-consul-boot",
			}.Instance(addrs.NoKey).Absolute(boot),
			&ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id": "1234567890"}`),
				Status:    ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(boot),
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "consul-green",
			}.Instance(addrs.IntKey(0)).Absolute(consul),
			&ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id": "1234567890"}`),
				Status:    ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(consul),
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "consul-green",
			}.Instance(addrs.IntKey(1)).Absolute(consul),
			&ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id": "1234567890"}`),
				Status:    ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(consul),
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "consul-green",
			}.Instance(addrs.IntKey(2)).Absolute(consul),
			&ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id": "1234567890"}`),
				Status:    ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(consul),
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_security_group",
				Name: "consul",
			}.Instance(addrs.NoKey).Absolute(consul),
			&ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id": "1234567890"}`),
				Status:    ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(consul),
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_elb",
				Name: "vault",
			}.Instance(addrs.NoKey).Absolute(vault),
			&ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id": "1234567890"}`),
				Status:    ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(vault),
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "vault",
			}.Instance(addrs.IntKey(0)).Absolute(vault),
			&ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id": "1234567890"}`),
				Status:    ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(vault),
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "vault",
			}.Instance(addrs.IntKey(1)).Absolute(vault),
			&ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id": "1234567890"}`),
				Status:    ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(vault),
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "vault",
			}.Instance(addrs.IntKey(2)).Absolute(vault),
			&ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id": "1234567890"}`),
				Status:    ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(vault),
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_security_group",
				Name: "vault",
			}.Instance(addrs.NoKey).Absolute(vault),
			&ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id": "1234567890"}`),
				Status:    ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(vault),
		)
	})
}

// testStateNestedModules returns a test State structure.
func testStateNestedModules() *State {
	outer, _ := addrs.ParseModuleInstanceStr("module.outer")
	child1, _ := addrs.ParseModuleInstanceStr("module.outer.module.child1")
	child2, _ := addrs.ParseModuleInstanceStr("module.outer.module.child2")

	state := BuildState(func(s *SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(child1),
			&ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id": "1234567890"}`),
				Status:    ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(child1),
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(child2),
			&ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id": "1234567890"}`),
				Status:    ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(child2),
		)
	})

	state.Modules[outer.String()] = NewModule(outer)
	return state
}

// testStateSingleMinimal returns a test State structure.
func testStateSingleMinimal() *State {
	return BuildState(func(s *SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "web",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id": "1234567890"}`),
				Status:    ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(addrs.RootModuleInstance),
		)
	})
}

// testStateSmall returns a test State structure.
func testStateSmall() *State {
	root := addrs.RootModuleInstance
	boot, _ := addrs.ParseModuleInstanceStr("module.boot")

	state := BuildState(func(s *SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_key_pair",
				Name: "onprem",
			}.Instance(addrs.NoKey).Absolute(root),
			&ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id": "1234567890"}`),
				Status:    ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(root),
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_route53_record",
				Name: "oasis-consul-boot-a",
			}.Instance(addrs.NoKey).Absolute(boot),
			&ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id": "1234567890"}`),
				Status:    ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(boot),
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_route53_record",
				Name: "oasis-consul-boot-ns",
			}.Instance(addrs.NoKey).Absolute(boot),
			&ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id": "1234567890"}`),
				Status:    ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(boot),
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_route53_zone",
				Name: "oasis-consul-boot",
			}.Instance(addrs.NoKey).Absolute(boot),
			&ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id": "1234567890"}`),
				Status:    ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(boot),
		)
	})
	// fmt.Printf("mods: %#v\n", state.Modules)
	// fmt.Printf("boot: %#+v\n", state.Modules["module.boot"])
	return state
}

// testStateSmallTestInstance returns a test State structure.
func testStateSmallTestInstance() *State {
	return BuildState(func(s *SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id": "1234567890"}`),
				Status:    ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "test",
			}.Absolute(addrs.RootModuleInstance),
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "bar",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id": "1234567890"}`),
				Status:    ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "test",
			}.Absolute(addrs.RootModuleInstance),
		)
	})
}

// testStateResourceInModule returns a test State structure.
func testStateResourceInModule() *State {
	foo, _ := addrs.ParseModuleInstanceStr("module.foo")

	return BuildState(func(s *SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "bar",
			}.Instance(addrs.NoKey).Absolute(foo),
			&ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id": "1234567890"}`),
				Status:    ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(foo),
		)
	})
}
