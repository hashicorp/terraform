package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

// This is the directory where our test fixtures are.
const fixtureDir = "./test-fixtures"

func TestLoadConfig(t *testing.T) {
	c, err := LoadConfig(filepath.Join(fixtureDir, "config"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := &Config{
		Providers: map[string]string{
			"aws": "foo",
			"do":  "bar",
		},
	}

	if !reflect.DeepEqual(c, expected) {
		t.Fatalf("bad: %#v", c)
	}
}

func TestLoadConfig_env(t *testing.T) {
	defer os.Unsetenv("TFTEST")
	os.Setenv("TFTEST", "hello")

	c, err := LoadConfig(filepath.Join(fixtureDir, "config-env"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := &Config{
		Providers: map[string]string{
			"aws":    "hello",
			"google": "bar",
		},
		Provisioners: map[string]string{
			"local": "hello",
		},
	}

	if !reflect.DeepEqual(c, expected) {
		t.Fatalf("bad: %#v", c)
	}
}

func TestLoadConfig_credentials(t *testing.T) {
	got, err := LoadConfig(filepath.Join(fixtureDir, "credentials"))
	if err != nil {
		t.Fatal(err)
	}

	want := &Config{
		Credentials: map[string]map[string]interface{}{
			"example.com": map[string]interface{}{
				"token": "foo the bar baz",
			},
			"example.net": map[string]interface{}{
				"username": "foo",
				"password": "baz",
			},
		},
		CredentialsHelpers: map[string]*ConfigCredentialsHelper{
			"foo": &ConfigCredentialsHelper{
				Args: []string{"bar", "baz"},
			},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrong result\ngot:  %swant: %s", spew.Sdump(got), spew.Sdump(want))
	}
}

func TestConfig_Merge(t *testing.T) {
	c1 := &Config{
		Providers: map[string]string{
			"foo": "bar",
			"bar": "blah",
		},
		Provisioners: map[string]string{
			"local":  "local",
			"remote": "bad",
		},
	}

	c2 := &Config{
		Providers: map[string]string{
			"bar": "baz",
			"baz": "what",
		},
		Provisioners: map[string]string{
			"remote": "remote",
		},
	}

	expected := &Config{
		Providers: map[string]string{
			"foo": "bar",
			"bar": "baz",
			"baz": "what",
		},
		Provisioners: map[string]string{
			"local":  "local",
			"remote": "remote",
		},
	}

	actual := c1.Merge(c2)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestConfig_Merge_disableCheckpoint(t *testing.T) {
	c1 := &Config{
		DisableCheckpoint: true,
	}

	c2 := &Config{}

	expected := &Config{
		Providers:         map[string]string{},
		Provisioners:      map[string]string{},
		DisableCheckpoint: true,
	}

	actual := c1.Merge(c2)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestConfig_Merge_disableCheckpointSignature(t *testing.T) {
	c1 := &Config{
		DisableCheckpointSignature: true,
	}

	c2 := &Config{}

	expected := &Config{
		Providers:                  map[string]string{},
		Provisioners:               map[string]string{},
		DisableCheckpointSignature: true,
	}

	actual := c1.Merge(c2)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}
