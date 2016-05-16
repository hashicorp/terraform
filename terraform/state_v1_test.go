package terraform

import (
	"bufio"
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestReadUpgradeStateV1toV2(t *testing.T) {
	// ReadState should transparently detect the old version but will upgrade
	// it on Write.
	actual, err := ReadState(strings.NewReader(testV1State))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	buf := new(bytes.Buffer)
	if err := actual.WriteState(buf); err != nil {
		t.Fatalf("err: %s", err)
	}

	if actual.Version != 2 {
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

func TestReadUpgradeStateV1toV2_outputs(t *testing.T) {
	// ReadState should transparently detect the old version but will upgrade
	// it on Write.
	actual, err := ReadState(strings.NewReader(testV1StateWithOutputs))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	buf := new(bytes.Buffer)
	if err := actual.WriteState(buf); err != nil {
		t.Fatalf("err: %s", err)
	}

	if actual.Version != 2 {
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

func TestDowngradeStateV2ToV1_downgradableLossless(t *testing.T) {
	// Even though this is technically V1 state, the reader will upgrade it to V2.
	// The fact we have gone V1->V2 implies that it is possible to losslessly go
	// from V2->V1.
	source, err := ReadState(strings.NewReader(testV1FullStateWithOutputs))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	downgraded, lossy, err := source.downgradeToV1()

	if lossy {
		t.Fatalf("Conversion which should have been lossless was not:\nOriginal:\n%s\nDowngraded:\n%s\n",
			spew.Sdump(source), spew.Sdump(downgraded))
	}

	var stateV1Written bytes.Buffer
	writer := bufio.NewWriter(&stateV1Written)
	downgraded.WriteState(writer)
	writer.Flush()

	roundtripped, err := ReadState(bytes.NewReader(stateV1Written.Bytes()))
	if err != nil {
		t.Fatalf("Error reading roundtripped state: %s", err)
	}

	if !roundtripped.Equal(source) {
		t.Fatalf("Round tripped state is not equivalent: Source:\n%s\n\nRoundTripped:\n%s\n",
			spew.Sdump(source), spew.Sdump(downgraded))
	}
}

const testV1FullStateWithOutputs = `{
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
                    },
                    "deposed": [
                        {
                            "id": "bar",
                            "attributes": {
                                "foo": "bar",
                                "baz": "boo"
                            }
                        }
                    ],
                    "tainted": [
                        {
                            "id": "boo"
                        }
                    ]
                }
            },
            "depends_on": [
                "aws_instance.bar"
            ]
        }
    ]
}
`
