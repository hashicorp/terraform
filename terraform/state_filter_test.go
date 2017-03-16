package terraform

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStateFilterFilter(t *testing.T) {
	cases := map[string]struct {
		State    string
		Filters  []string
		Expected []string
	}{
		"all": {
			"small.tfstate",
			[]string{},
			[]string{
				"*terraform.ResourceState: aws_key_pair.onprem",
				"*terraform.InstanceState: aws_key_pair.onprem",
				"*terraform.ModuleState: module.bootstrap",
				"*terraform.ResourceState: module.bootstrap.aws_route53_record.oasis-consul-bootstrap-a",
				"*terraform.InstanceState: module.bootstrap.aws_route53_record.oasis-consul-bootstrap-a",
				"*terraform.ResourceState: module.bootstrap.aws_route53_record.oasis-consul-bootstrap-ns",
				"*terraform.InstanceState: module.bootstrap.aws_route53_record.oasis-consul-bootstrap-ns",
				"*terraform.ResourceState: module.bootstrap.aws_route53_zone.oasis-consul-bootstrap",
				"*terraform.InstanceState: module.bootstrap.aws_route53_zone.oasis-consul-bootstrap",
			},
		},

		"single resource": {
			"small.tfstate",
			[]string{"aws_key_pair.onprem"},
			[]string{
				"*terraform.ResourceState: aws_key_pair.onprem",
				"*terraform.InstanceState: aws_key_pair.onprem",
			},
		},

		"single resource from minimal state": {
			"single-minimal-resource.tfstate",
			[]string{"aws_instance.web"},
			[]string{
				"*terraform.ResourceState: aws_instance.web",
				"*terraform.InstanceState: aws_instance.web",
			},
		},

		"single resource with similar names": {
			"small_test_instance.tfstate",
			[]string{"test_instance.foo"},
			[]string{
				"*terraform.ResourceState: test_instance.foo",
				"*terraform.InstanceState: test_instance.foo",
			},
		},

		"single instance": {
			"small.tfstate",
			[]string{"aws_key_pair.onprem.primary"},
			[]string{
				"*terraform.InstanceState: aws_key_pair.onprem",
			},
		},

		"module filter": {
			"complete.tfstate",
			[]string{"module.bootstrap"},
			[]string{
				"*terraform.ModuleState: module.bootstrap",
				"*terraform.ResourceState: module.bootstrap.aws_route53_record.oasis-consul-bootstrap-a",
				"*terraform.InstanceState: module.bootstrap.aws_route53_record.oasis-consul-bootstrap-a",
				"*terraform.ResourceState: module.bootstrap.aws_route53_record.oasis-consul-bootstrap-ns",
				"*terraform.InstanceState: module.bootstrap.aws_route53_record.oasis-consul-bootstrap-ns",
				"*terraform.ResourceState: module.bootstrap.aws_route53_zone.oasis-consul-bootstrap",
				"*terraform.InstanceState: module.bootstrap.aws_route53_zone.oasis-consul-bootstrap",
			},
		},

		"resource in module": {
			"complete.tfstate",
			[]string{"module.bootstrap.aws_route53_zone.oasis-consul-bootstrap"},
			[]string{
				"*terraform.ResourceState: module.bootstrap.aws_route53_zone.oasis-consul-bootstrap",
				"*terraform.InstanceState: module.bootstrap.aws_route53_zone.oasis-consul-bootstrap",
			},
		},

		"resource in module 2": {
			"resource-in-module-2.tfstate",
			[]string{"module.foo.aws_instance.foo"},
			[]string{},
		},

		"single count index": {
			"complete.tfstate",
			[]string{"module.consul.aws_instance.consul-green[0]"},
			[]string{
				"*terraform.ResourceState: module.consul.aws_instance.consul-green[0]",
				"*terraform.InstanceState: module.consul.aws_instance.consul-green[0]",
			},
		},

		"no count index": {
			"complete.tfstate",
			[]string{"module.consul.aws_instance.consul-green"},
			[]string{
				"*terraform.ResourceState: module.consul.aws_instance.consul-green[0]",
				"*terraform.InstanceState: module.consul.aws_instance.consul-green[0]",
				"*terraform.ResourceState: module.consul.aws_instance.consul-green[1]",
				"*terraform.InstanceState: module.consul.aws_instance.consul-green[1]",
				"*terraform.ResourceState: module.consul.aws_instance.consul-green[2]",
				"*terraform.InstanceState: module.consul.aws_instance.consul-green[2]",
			},
		},

		"nested modules": {
			"nested-modules.tfstate",
			[]string{"module.outer"},
			[]string{
				"*terraform.ModuleState: module.outer",
				"*terraform.ModuleState: module.outer.module.child1",
				"*terraform.ResourceState: module.outer.module.child1.aws_instance.foo",
				"*terraform.InstanceState: module.outer.module.child1.aws_instance.foo",
				"*terraform.ModuleState: module.outer.module.child2",
				"*terraform.ResourceState: module.outer.module.child2.aws_instance.foo",
				"*terraform.InstanceState: module.outer.module.child2.aws_instance.foo",
			},
		},
	}

	for n, tc := range cases {
		// Load our state
		f, err := os.Open(filepath.Join("./test-fixtures", "state-filter", tc.State))
		if err != nil {
			t.Fatalf("%q: err: %s", n, err)
		}

		state, err := ReadState(f)
		f.Close()
		if err != nil {
			t.Fatalf("%q: err: %s", n, err)
		}

		// Create the filter
		filter := &StateFilter{State: state}

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
