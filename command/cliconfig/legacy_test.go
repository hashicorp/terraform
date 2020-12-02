package cliconfig

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
)

// This is the directory where our test fixtures are.
const fixtureDir = "./testdata"

func TestLegacyLoadConfig(t *testing.T) {
	c, err := legacyLoadConfigFile(filepath.Join(fixtureDir, "config"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := &LegacyConfig{
		Providers: map[string]string{
			"aws": "foo",
			"do":  "bar",
		},
	}

	if !reflect.DeepEqual(c, expected) {
		t.Fatalf("bad: %#v", c)
	}
}

func TestLegacyLoadConfig_env(t *testing.T) {
	defer os.Unsetenv("TFTEST")
	os.Setenv("TFTEST", "hello")

	c, err := legacyLoadConfigFile(filepath.Join(fixtureDir, "config-env"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := &LegacyConfig{
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

func TestLegacyLoadConfig_hosts(t *testing.T) {
	got, diags := legacyLoadConfigFile(filepath.Join(fixtureDir, "hosts"))
	if len(diags) != 0 {
		t.Fatalf("%s", diags.Err())
	}

	want := &LegacyConfig{
		Hosts: map[string]*LegacyConfigHost{
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

func TestLegacyLoadConfig_credentials(t *testing.T) {
	got, err := legacyLoadConfigFile(filepath.Join(fixtureDir, "credentials"))
	if err != nil {
		t.Fatal(err)
	}

	want := &LegacyConfig{
		Credentials: map[string]map[string]interface{}{
			"example.com": map[string]interface{}{
				"token": "foo the bar baz",
			},
			"example.net": map[string]interface{}{
				"username": "foo",
				"password": "baz",
			},
		},
		CredentialsHelpers: map[string]*LegacyConfigCredentialsHelper{
			"foo": &LegacyConfigCredentialsHelper{
				Args: []string{"bar", "baz"},
			},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrong result\ngot:  %swant: %s", spew.Sdump(got), spew.Sdump(want))
	}
}

func TestLegacyConfigValidate(t *testing.T) {
	tests := map[string]struct {
		Config    *LegacyConfig
		DiagCount int
	}{
		"nil": {
			nil,
			0,
		},
		"empty": {
			&LegacyConfig{},
			0,
		},
		"host good": {
			&LegacyConfig{
				Hosts: map[string]*LegacyConfigHost{
					"example.com": {},
				},
			},
			0,
		},
		"host with bad hostname": {
			&LegacyConfig{
				Hosts: map[string]*LegacyConfigHost{
					"example..com": {},
				},
			},
			1, // host block has invalid hostname
		},
		"credentials good": {
			&LegacyConfig{
				Credentials: map[string]map[string]interface{}{
					"example.com": map[string]interface{}{
						"token": "foo",
					},
				},
			},
			0,
		},
		"credentials with bad hostname": {
			&LegacyConfig{
				Credentials: map[string]map[string]interface{}{
					"example..com": map[string]interface{}{
						"token": "foo",
					},
				},
			},
			1, // credentials block has invalid hostname
		},
		"credentials helper good": {
			&LegacyConfig{
				CredentialsHelpers: map[string]*LegacyConfigCredentialsHelper{
					"foo": {},
				},
			},
			0,
		},
		"credentials helper too many": {
			&LegacyConfig{
				CredentialsHelpers: map[string]*LegacyConfigCredentialsHelper{
					"foo": {},
					"bar": {},
				},
			},
			1, // no more than one credentials_helper block allowed
		},
		"provider_installation good none": {
			&LegacyConfig{
				ProviderInstallation: nil,
			},
			0,
		},
		"provider_installation good one": {
			&LegacyConfig{
				ProviderInstallation: []*LegacyProviderInstallation{
					{},
				},
			},
			0,
		},
		"provider_installation too many": {
			&LegacyConfig{
				ProviderInstallation: []*LegacyProviderInstallation{
					{},
					{},
				},
			},
			1, // no more than one provider_installation block allowed
		},
		"plugin_cache_dir does not exist": {
			&LegacyConfig{
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

func TestLegacyConfig_Merge(t *testing.T) {
	c1 := &LegacyConfig{
		Providers: map[string]string{
			"foo": "bar",
			"bar": "blah",
		},
		Provisioners: map[string]string{
			"local":  "local",
			"remote": "bad",
		},
		Hosts: map[string]*LegacyConfigHost{
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
		CredentialsHelpers: map[string]*LegacyConfigCredentialsHelper{
			"buz": {},
		},
		ProviderInstallation: []*LegacyProviderInstallation{
			{
				Methods: []*LegacyProviderInstallationMethod{
					{Location: ProviderInstallationFilesystemMirror("a")},
					{Location: ProviderInstallationFilesystemMirror("b")},
				},
			},
			{
				Methods: []*LegacyProviderInstallationMethod{
					{Location: ProviderInstallationFilesystemMirror("c")},
				},
			},
		},
	}

	c2 := &LegacyConfig{
		Providers: map[string]string{
			"bar": "baz",
			"baz": "what",
		},
		Provisioners: map[string]string{
			"remote": "remote",
		},
		Hosts: map[string]*LegacyConfigHost{
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
		CredentialsHelpers: map[string]*LegacyConfigCredentialsHelper{
			"biz": {},
		},
		ProviderInstallation: []*LegacyProviderInstallation{
			{
				Methods: []*LegacyProviderInstallationMethod{
					{Location: ProviderInstallationFilesystemMirror("d")},
				},
			},
		},
	}

	expected := &LegacyConfig{
		Providers: map[string]string{
			"foo": "bar",
			"bar": "baz",
			"baz": "what",
		},
		Provisioners: map[string]string{
			"local":  "local",
			"remote": "remote",
		},
		Hosts: map[string]*LegacyConfigHost{
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
		CredentialsHelpers: map[string]*LegacyConfigCredentialsHelper{
			"buz": {},
			"biz": {},
		},
		ProviderInstallation: []*LegacyProviderInstallation{
			{
				Methods: []*LegacyProviderInstallationMethod{
					{Location: ProviderInstallationFilesystemMirror("a")},
					{Location: ProviderInstallationFilesystemMirror("b")},
				},
			},
			{
				Methods: []*LegacyProviderInstallationMethod{
					{Location: ProviderInstallationFilesystemMirror("c")},
				},
			},
			{
				Methods: []*LegacyProviderInstallationMethod{
					{Location: ProviderInstallationFilesystemMirror("d")},
				},
			},
		},
	}

	actual := c1.Merge(c2)
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Fatalf("wrong result\n%s", diff)
	}
}

func TestLegacyConfig_Merge_disableCheckpoint(t *testing.T) {
	c1 := &LegacyConfig{
		DisableCheckpoint: true,
	}

	c2 := &LegacyConfig{}

	expected := &LegacyConfig{
		Providers:         map[string]string{},
		Provisioners:      map[string]string{},
		DisableCheckpoint: true,
	}

	actual := c1.Merge(c2)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestLegacyConfig_Merge_disableCheckpointSignature(t *testing.T) {
	c1 := &LegacyConfig{
		DisableCheckpointSignature: true,
	}

	c2 := &LegacyConfig{}

	expected := &LegacyConfig{
		Providers:                  map[string]string{},
		Provisioners:               map[string]string{},
		DisableCheckpointSignature: true,
	}

	actual := c1.Merge(c2)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}
