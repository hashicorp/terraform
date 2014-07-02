package aws

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/flatmap"
	"github.com/mitchellh/goamz/elb"
)

// Returns test configuration
func testConf() map[string]string {
	return map[string]string{
		"listener.#":                   "1",
		"listener.0.lb_port":           "80",
		"listener.0.lb_protocol":       "http",
		"listener.0.instance_port":     "8000",
		"listener.0.instance_protocol": "http",
	}
}

func Test_expandListeners(t *testing.T) {
	expanded := flatmap.Expand(testConf(), "listener").([]interface{})
	listeners := expandListeners(expanded)
	expected := elb.Listener{
		InstancePort:     8000,
		LoadBalancerPort: 80,
		InstanceProtocol: "http",
		Protocol:         "http",
	}

	if !reflect.DeepEqual(listeners[0], expected) {
		t.Fatalf(
			"Got:\n\n%#v\n\nExpected:\n\n%#v\n",
			listeners[0],
			expected)
	}

}
