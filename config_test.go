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
	c, err := loadConfigFile(filepath.Join(fixtureDir, "config"))
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

	c, err := loadConfigFile(filepath.Join(fixtureDir, "config-env"))
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
	got, err := loadConfigFile(filepath.Join(fixtureDir, "credentials"))
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

func TestConfigValidate(t *testing.T) {
	tests := map[string]struct {
		Config    *Config
		DiagCount int
	}{
		"nil": {
			nil,
			0,
		},
		"empty": {
			&Config{},
			0,
		},
		"credentials good": {
			&Config{
				Credentials: map[string]map[string]interface{}{
					"example.com": map[string]interface{}{
						"token": "foo",
					},
				},
			},
			0,
		},
		"credentials with bad hostname": {
			&Config{
				Credentials: map[string]map[string]interface{}{
					"example..com": map[string]interface{}{
						"token": "foo",
					},
				},
			},
			1, // credentials block has invalid hostname
		},
		"credentials helper good": {
			&Config{
				CredentialsHelpers: map[string]*ConfigCredentialsHelper{
					"foo": {},
				},
			},
			0,
		},
		"credentials helper too many": {
			&Config{
				CredentialsHelpers: map[string]*ConfigCredentialsHelper{
					"foo": {},
					"bar": {},
				},
			},
			1, // no more than one credentials_helper block allowed
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			diags := test.Config.Validate()
			if len(diags) != test.DiagCount {
				t.Errorf("wrong number of diagnostics %d; want %d", len(diags), test.DiagCount)
				for _, diag := range diags {
					t.Logf("- %#v", diag.Description())
				}
			}
		})
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
		Credentials: map[string]map[string]interface{}{
			"foo": {
				"bar": "baz",
			},
		},
		CredentialsHelpers: map[string]*ConfigCredentialsHelper{
			"buz": {},
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
		Credentials: map[string]map[string]interface{}{
			"fee": {
				"bur": "bez",
			},
		},
		CredentialsHelpers: map[string]*ConfigCredentialsHelper{
			"biz": {},
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
		Credentials: map[string]map[string]interface{}{
			"foo": {
				"bar": "baz",
			},
			"fee": {
				"bur": "bez",
			},
		},
		CredentialsHelpers: map[string]*ConfigCredentialsHelper{
			"buz": {},
			"biz": {},
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
