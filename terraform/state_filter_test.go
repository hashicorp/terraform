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
