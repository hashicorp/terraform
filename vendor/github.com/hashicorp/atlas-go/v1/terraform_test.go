package atlas

import (
	"bytes"
	"reflect"
	"testing"
)

func TestTerraformConfigLatest(t *testing.T) {
	server := newTestAtlasServer(t)
	defer server.Stop()

	client, err := NewClient(server.URL.String())
	if err != nil {
		t.Fatal(err)
	}

	actual, err := client.TerraformConfigLatest("hashicorp", "existing")
	if err != nil {
		t.Fatal(err)
	}

	expected := &TerraformConfigVersion{
		Version:   5,
		Metadata:  map[string]string{"foo": "bar"},
		Variables: map[string]string{"foo": "bar"},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("%#v", actual)
	}
}

func TestCreateTerraformConfigVersion(t *testing.T) {
	server := newTestAtlasServer(t)
	defer server.Stop()

	client, err := NewClient(server.URL.String())
	if err != nil {
		t.Fatal(err)
	}

	v := &TerraformConfigVersion{
		Version:  5,
		Metadata: map[string]string{"foo": "bar"},
	}

	data := new(bytes.Buffer)
	vsn, err := client.CreateTerraformConfigVersion(
		"hashicorp", "existing", v, data, int64(data.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if vsn != 5 {
		t.Fatalf("bad: %v", vsn)
	}
}
