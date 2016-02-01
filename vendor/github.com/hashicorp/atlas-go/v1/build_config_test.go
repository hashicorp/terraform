package atlas

import (
	"bytes"
	"reflect"
	"testing"
)

func TestBuildConfig_slug(t *testing.T) {
	bc := &BuildConfig{User: "sethvargo", Name: "bacon"}
	expected := "sethvargo/bacon"
	if bc.Slug() != expected {
		t.Errorf("expected %q to be %q", bc.Slug(), expected)
	}
}

func TestBuildConfigVersion_slug(t *testing.T) {
	bc := &BuildConfigVersion{User: "sethvargo", Name: "bacon"}
	expected := "sethvargo/bacon"
	if bc.Slug() != expected {
		t.Errorf("expected %q to be %q", bc.Slug(), expected)
	}
}

func TestBuildConfig_fetches(t *testing.T) {
	server := newTestAtlasServer(t)
	defer server.Stop()

	client, err := NewClient(server.URL.String())
	if err != nil {
		t.Fatal(err)
	}

	actual, err := client.BuildConfig("hashicorp", "existing")
	if err != nil {
		t.Fatal(err)
	}

	expected := &BuildConfig{
		User: "hashicorp",
		Name: "existing",
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("%#v", actual)
	}
}

func TestCreateBuildConfig(t *testing.T) {
	server := newTestAtlasServer(t)
	defer server.Stop()

	client, err := NewClient(server.URL.String())
	if err != nil {
		t.Fatal(err)
	}

	user, name := "hashicorp", "new"
	bc, err := client.CreateBuildConfig(user, name)
	if err != nil {
		t.Fatal(err)
	}

	if bc.User != user {
		t.Errorf("expected %q to be %q", bc.User, user)
	}

	if bc.Name != name {
		t.Errorf("expected %q to be %q", bc.Name, name)
	}
}

func TestUploadBuildConfigVersion(t *testing.T) {
	server := newTestAtlasServer(t)
	defer server.Stop()

	client, err := NewClient(server.URL.String())
	if err != nil {
		t.Fatal(err)
	}

	bc := &BuildConfigVersion{
		User: "hashicorp",
		Name: "existing",
		Builds: []BuildConfigBuild{
			BuildConfigBuild{Name: "foo", Type: "ami"},
		},
	}
	metadata := map[string]interface{}{"testing": true}
	data := new(bytes.Buffer)
	err = client.UploadBuildConfigVersion(bc, metadata, data, int64(data.Len()))
	if err != nil {
		t.Fatal(err)
	}
}
