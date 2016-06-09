package terraform

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

// TestReadUpgradeStateV1toV3 tests the state upgrade process from the V1 state
// to the current version, and needs editing each time. This means it tests the
// entire pipeline of upgrades (which migrate version to version).
func TestReadUpgradeStateV1toV3(t *testing.T) {
	// ReadState should transparently detect the old version but will upgrade
	// it on Write.
	actual, err := ReadState(strings.NewReader(testV1State))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	buf := new(bytes.Buffer)
	if err := WriteState(actual, buf); err != nil {
		t.Fatalf("err: %s", err)
	}

	if actual.Version != 3 {
		t.Fatalf("bad: State version not incremented; is %d", actual.Version)
	}

	roundTripped, err := ReadState(buf)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(actual, roundTripped) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestReadUpgradeStateV1toV3_outputs(t *testing.T) {
	// ReadState should transparently detect the old version but will upgrade
	// it on Write.
	actual, err := ReadState(strings.NewReader(testV1StateWithOutputs))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	buf := new(bytes.Buffer)
	if err := WriteState(actual, buf); err != nil {
		t.Fatalf("err: %s", err)
	}

	if actual.Version != 3 {
		t.Fatalf("bad: State version not incremented; is %d", actual.Version)
	}

	roundTripped, err := ReadState(buf)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(actual, roundTripped) {
		spew.Config.DisableMethods = true
		t.Fatalf("bad:\n%s\n\nround tripped:\n%s\n", spew.Sdump(actual), spew.Sdump(roundTripped))
		spew.Config.DisableMethods = false
	}
}

const testV1State = `{
    "version": 1,
    "serial": 9,
    "remote": {
        "type": "http",
        "config": {
            "url": "http://my-cool-server.com/"
        }
    },
    "modules": [
        {
            "path": [
                "root"
            ],
            "outputs": null,
            "resources": {
                "foo": {
                    "type": "",
                    "primary": {
                        "id": "bar"
                    }
                }
            },
            "depends_on": [
                "aws_instance.bar"
            ]
        }
    ]
}
`

const testV1StateWithOutputs = `{
    "version": 1,
    "serial": 9,
    "remote": {
        "type": "http",
        "config": {
            "url": "http://my-cool-server.com/"
        }
    },
    "modules": [
        {
            "path": [
                "root"
            ],
            "outputs": {
            	"foo": "bar",
            	"baz": "foo"
            },
            "resources": {
                "foo": {
                    "type": "",
                    "primary": {
                        "id": "bar"
                    }
                }
            },
            "depends_on": [
                "aws_instance.bar"
            ]
        }
    ]
}
`
