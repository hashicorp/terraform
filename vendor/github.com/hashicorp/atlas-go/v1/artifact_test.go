package atlas

import (
	"bytes"
	"testing"
)

func TestArtifact_fetchesArtifact(t *testing.T) {
	server := newTestAtlasServer(t)
	defer server.Stop()

	client, err := NewClient(server.URL.String())
	if err != nil {
		t.Fatal(err)
	}

	art, err := client.Artifact("hashicorp", "existing")
	if err != nil {
		t.Fatal(err)
	}

	if art.User != "hashicorp" {
		t.Errorf("expected %q to be %q", art.User, "hashicorp")
	}

	if art.Name != "existing" {
		t.Errorf("expected %q to be %q", art.Name, "existing")
	}
}

func TestArtifact_returnsErrorNoArtifact(t *testing.T) {
	server := newTestAtlasServer(t)
	defer server.Stop()

	client, err := NewClient(server.URL.String())
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.App("hashicorp", "newproject")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}
}

func TestArtifactSearch_fetches(t *testing.T) {
	server := newTestAtlasServer(t)
	defer server.Stop()

	client, err := NewClient(server.URL.String())
	if err != nil {
		t.Fatal(err)
	}

	vs, err := client.ArtifactSearch(&ArtifactSearchOpts{
		User: "hashicorp",
		Name: "existing1",
		Type: "amazon-ami",
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(vs) != 1 {
		t.Fatalf("bad: %#v", vs)
	}
}

func TestArtifactSearch_metadata(t *testing.T) {
	server := newTestAtlasServer(t)
	defer server.Stop()

	client, err := NewClient(server.URL.String())
	if err != nil {
		t.Fatal(err)
	}

	vs, err := client.ArtifactSearch(&ArtifactSearchOpts{
		User: "hashicorp",
		Name: "existing2",
		Type: "amazon-ami",
		Metadata: map[string]string{
			"foo": "bar",
			"bar": MetadataAnyValue,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(vs) != 1 {
		t.Fatalf("bad: %#v", vs)
	}
}

func TestArtifactFileURL(t *testing.T) {
	server := newTestAtlasServer(t)
	defer server.Stop()

	client, err := NewClient(server.URL.String())
	if err != nil {
		t.Fatal(err)
	}

	v := &ArtifactVersion{
		User: "foo",
		Name: "bar",
		Type: "vagrant-box",
		File: true,
	}

	u, err := client.ArtifactFileURL(v)
	if err != nil {
		t.Fatal(err)
	}

	expected := *server.URL
	expected.Path = "/api/v1/artifacts/foo/bar/vagrant-box/file"
	if u.String() != expected.String() {
		t.Fatalf("unexpected: %s\n\nexpected: %s", u, expected.String())
	}
}

func TestArtifactFileURL_nil(t *testing.T) {
	server := newTestAtlasServer(t)
	defer server.Stop()

	client, err := NewClient(server.URL.String())
	if err != nil {
		t.Fatal(err)
	}

	v := &ArtifactVersion{
		User: "foo",
		Name: "bar",
		Type: "vagrant-box",
	}

	u, err := client.ArtifactFileURL(v)
	if err != nil {
		t.Fatal(err)
	}
	if u != nil {
		t.Fatal("should be nil")
	}
}

func TestUploadArtifact(t *testing.T) {
	server := newTestAtlasServer(t)
	defer server.Stop()

	client, err := NewClient(server.URL.String())
	if err != nil {
		t.Fatal(err)
	}

	data := new(bytes.Buffer)
	_, err = client.UploadArtifact(&UploadArtifactOpts{
		User: "hashicorp",
		Name: "existing",
		Type: "amazon-ami",
		File: data,
	})
	if err != nil {
		t.Fatal(err)
	}
}
