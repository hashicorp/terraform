// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cliconfig

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// This is the directory where our test fixtures are.
const fixtureDir = "./testdata"

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

func TestLoadConfig_envSubst(t *testing.T) {
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

func TestLoadConfig_non_existing_file(t *testing.T) {
	tmpDir := os.TempDir()
	cliTmpFile := filepath.Join(tmpDir, "dev.tfrc")

	os.Setenv("TF_CLI_CONFIG_FILE", cliTmpFile)
	defer os.Unsetenv("TF_CLI_CONFIG_FILE")

	c, errs := LoadConfig()
	if errs.HasErrors() || c.Validate().HasErrors() {
		t.Fatalf("err: %s", errs)
	}

	hasOpenFileWarn := false
	for _, err := range errs {
		if err.Severity() == tfdiags.Warning && err.Description().Summary == "Unable to open CLI configuration file" {
			hasOpenFileWarn = true
			break
		}
	}

	if !hasOpenFileWarn {
		t.Fatal("expecting a warning message because of nonexisting CLI configuration file")
	}
}

func TestEnvConfig(t *testing.T) {
	tests := map[string]struct {
		env  map[string]string
		want *Config
	}{
		"no environment variables": {
			nil,
			&Config{},
		},
		"TF_PLUGIN_CACHE_DIR=boop": {
			map[string]string{
				"TF_PLUGIN_CACHE_DIR": "boop",
			},
			&Config{
				PluginCacheDir: "boop",
			},
		},
		"TF_PLUGIN_CACHE_MAY_BREAK_DEPENDENCY_LOCK_FILE=anything_except_zero": {
			map[string]string{
				"TF_PLUGIN_CACHE_MAY_BREAK_DEPENDENCY_LOCK_FILE": "anything_except_zero",
			},
			&Config{
				PluginCacheMayBreakDependencyLockFile: true,
			},
		},
		"TF_PLUGIN_CACHE_MAY_BREAK_DEPENDENCY_LOCK_FILE=0": {
			map[string]string{
				"TF_PLUGIN_CACHE_MAY_BREAK_DEPENDENCY_LOCK_FILE": "0",
			},
			&Config{},
		},
		"TF_PLUGIN_CACHE_DIR and TF_PLUGIN_CACHE_MAY_BREAK_DEPENDENCY_LOCK_FILE": {
			map[string]string{
				"TF_PLUGIN_CACHE_DIR":                            "beep",
				"TF_PLUGIN_CACHE_MAY_BREAK_DEPENDENCY_LOCK_FILE": "1",
			},
			&Config{
				PluginCacheDir:                        "beep",
				PluginCacheMayBreakDependencyLockFile: true,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := envConfig(test.env)
			want := test.want

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}

func TestMakeEnvMap(t *testing.T) {
	tests := map[string]struct {
		environ []string
		want    map[string]string
	}{
		"nil": {
			nil,
			nil,
		},
		"one": {
			[]string{
				"FOO=bar",
			},
			map[string]string{
				"FOO": "bar",
			},
		},
		"many": {
			[]string{
				"FOO=1",
				"BAR=2",
				"BAZ=3",
			},
			map[string]string{
				"FOO": "1",
				"BAR": "2",
				"BAZ": "3",
			},
		},
		"conflict": {
			[]string{
				"FOO=1",
				"BAR=1",
				"FOO=2",
			},
			map[string]string{
				"BAR": "1",
				"FOO": "2", // Last entry of each name wins
			},
		},
		"empty_val": {
			[]string{
				"FOO=",
			},
			map[string]string{
				"FOO": "",
			},
		},
		"no_equals": {
			[]string{
				"FOO=bar",
				"INVALID",
			},
			map[string]string{
				"FOO": "bar",
			},
		},
		"multi_equals": {
			[]string{
				"FOO=bar=baz=boop",
			},
			map[string]string{
				"FOO": "bar=baz=boop",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := makeEnvMap(test.environ)
			want := test.want

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}

}

func TestLoadConfig_hosts(t *testing.T) {
	got, diags := loadConfigFile(filepath.Join(fixtureDir, "hosts"))
	if len(diags) != 0 {
		t.Fatalf("%s", diags.Err())
	}

	want := &Config{
		Hosts: map[string]*ConfigHost{
			"example.com": {
				Services: map[string]interface{}{
					"modules.v1": "https://example.com/",
				},
			},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrong result\ngot:  %swant: %s", spew.Sdump(got), spew.Sdump(want))
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
		"host good": {
			&Config{
				Hosts: map[string]*ConfigHost{
					"example.com": {},
				},
			},
			0,
		},
		"host with bad hostname": {
			&Config{
				Hosts: map[string]*ConfigHost{
					"example..com": {},
				},
			},
			1, // host block has invalid hostname
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
		"provider_installation good none": {
			&Config{
				ProviderInstallation: nil,
			},
			0,
		},
		"provider_installation good one": {
			&Config{
				ProviderInstallation: []*ProviderInstallation{
					{},
				},
			},
			0,
		},
		"provider_installation too many": {
			&Config{
				ProviderInstallation: []*ProviderInstallation{
					{},
					{},
				},
			},
			1, // no more than one provider_installation block allowed
		},
		"plugin_cache_dir does not exist": {
			&Config{
				PluginCacheDir: "fake",
			},
			1, // The specified plugin cache dir %s cannot be opened
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
		Hosts: map[string]*ConfigHost{
			"example.com": {
				Services: map[string]interface{}{
					"modules.v1": "http://example.com/",
				},
			},
		},
		Credentials: map[string]map[string]interface{}{
			"foo": {
				"bar": "baz",
			},
		},
		CredentialsHelpers: map[string]*ConfigCredentialsHelper{
			"buz": {},
		},
		ProviderInstallation: []*ProviderInstallation{
			{
				Methods: []*ProviderInstallationMethod{
					{Location: ProviderInstallationFilesystemMirror("a")},
					{Location: ProviderInstallationFilesystemMirror("b")},
				},
			},
			{
				Methods: []*ProviderInstallationMethod{
					{Location: ProviderInstallationFilesystemMirror("c")},
				},
			},
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
		Hosts: map[string]*ConfigHost{
			"example.net": {
				Services: map[string]interface{}{
					"modules.v1": "https://example.net/",
				},
			},
		},
		Credentials: map[string]map[string]interface{}{
			"fee": {
				"bur": "bez",
			},
		},
		CredentialsHelpers: map[string]*ConfigCredentialsHelper{
			"biz": {},
		},
		ProviderInstallation: []*ProviderInstallation{
			{
				Methods: []*ProviderInstallationMethod{
					{Location: ProviderInstallationFilesystemMirror("d")},
				},
			},
		},
		PluginCacheMayBreakDependencyLockFile: true,
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
		Hosts: map[string]*ConfigHost{
			"example.com": {
				Services: map[string]interface{}{
					"modules.v1": "http://example.com/",
				},
			},
			"example.net": {
				Services: map[string]interface{}{
					"modules.v1": "https://example.net/",
				},
			},
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
		ProviderInstallation: []*ProviderInstallation{
			{
				Methods: []*ProviderInstallationMethod{
					{Location: ProviderInstallationFilesystemMirror("a")},
					{Location: ProviderInstallationFilesystemMirror("b")},
				},
			},
			{
				Methods: []*ProviderInstallationMethod{
					{Location: ProviderInstallationFilesystemMirror("c")},
				},
			},
			{
				Methods: []*ProviderInstallationMethod{
					{Location: ProviderInstallationFilesystemMirror("d")},
				},
			},
		},
		PluginCacheMayBreakDependencyLockFile: true,
	}

	actual := c1.Merge(c2)
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Fatalf("wrong result\n%s", diff)
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
