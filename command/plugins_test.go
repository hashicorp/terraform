package command

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/hashicorp/terraform/terraform"
)

func TestMultiVersionProviderResolver(t *testing.T) {
	available := make(discovery.PluginMetaSet)
	available.Add(discovery.PluginMeta{
		Name:    "plugin",
		Version: "1.0.0",
		Path:    "test-fixtures/empty-file",
	})

	resolver := &multiVersionProviderResolver{
		Internal: map[string]terraform.ResourceProviderFactory{
			"internal": func() (terraform.ResourceProvider, error) {
				return &terraform.MockResourceProvider{
					ResourcesReturn: []terraform.ResourceType{
						{
							Name: "internal_foo",
						},
					},
				}, nil
			},
		},
		Available: available,
	}

	t.Run("plugin matches", func(t *testing.T) {
		reqd := discovery.PluginRequirements{
			"plugin": &discovery.PluginConstraints{
				Versions: discovery.ConstraintStr("1.0.0").MustParse(),
			},
		}
		got, err := resolver.ResolveProviders(reqd)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if ct := len(got); ct != 1 {
			t.Errorf("wrong number of results %d; want 1", ct)
		}
		if _, exists := got["plugin"]; !exists {
			t.Errorf("provider \"plugin\" not in result")
		}
	})
	t.Run("plugin doesn't match", func(t *testing.T) {
		reqd := discovery.PluginRequirements{
			"plugin": &discovery.PluginConstraints{
				Versions: discovery.ConstraintStr("2.0.0").MustParse(),
			},
		}
		_, err := resolver.ResolveProviders(reqd)
		if err == nil {
			t.Errorf("resolved successfully, but want error")
		}
	})
	t.Run("internal matches", func(t *testing.T) {
		reqd := discovery.PluginRequirements{
			"internal": &discovery.PluginConstraints{
				Versions: discovery.AllVersions,
			},
		}
		got, err := resolver.ResolveProviders(reqd)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if ct := len(got); ct != 1 {
			t.Errorf("wrong number of results %d; want 1", ct)
		}
		if _, exists := got["internal"]; !exists {
			t.Errorf("provider \"internal\" not in result")
		}
	})
	t.Run("internal with version constraint", func(t *testing.T) {
		// Version constraints are not permitted for internal providers
		reqd := discovery.PluginRequirements{
			"internal": &discovery.PluginConstraints{
				Versions: discovery.ConstraintStr("2.0.0").MustParse(),
			},
		}
		_, err := resolver.ResolveProviders(reqd)
		if err == nil {
			t.Errorf("resolved successfully, but want error")
		}
	})
}

func TestPluginPath(t *testing.T) {
	td := testTempDir(t)
	defer testChdir(t, td)()

	pluginPath := []string{"a", "b", "c"}

	m := Meta{}
	if err := m.storePluginPath(pluginPath); err != nil {
		t.Fatal(err)
	}

	restoredPath, err := m.loadPluginPath()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(pluginPath, restoredPath) {
		t.Fatalf("expected plugin path %#v, got %#v", pluginPath, restoredPath)
	}
}

func TestInternalProviders(t *testing.T) {
	m := Meta{}
	internal := m.internalProviders()
	tfProvider, err := internal["terraform"]()
	if err != nil {
		t.Fatal(err)
	}

	dataSources := tfProvider.DataSources()
	found := false
	for _, ds := range dataSources {
		if ds.Name == "terraform_remote_state" {
			found = true
		}
	}
	if !found {
		t.Errorf("didn't find terraform_remote_state in internal \"terraform\" provider")
	}
}

// mockProviderInstaller is a discovery.PluginInstaller implementation that
// is a mock for discovery.ProviderInstaller.
type mockProviderInstaller struct {
	// A map of provider names to available versions.
	// The tests expect the versions to be in order from newest to oldest.
	Providers map[string][]string

	Dir               string
	PurgeUnusedCalled bool
}

func (i *mockProviderInstaller) FileName(provider, version string) string {
	return fmt.Sprintf("terraform-provider-%s_v%s_x4", provider, version)
}

func (i *mockProviderInstaller) Get(provider string, req discovery.Constraints) (discovery.PluginMeta, error) {
	noMeta := discovery.PluginMeta{}
	versions := i.Providers[provider]
	if len(versions) == 0 {
		return noMeta, fmt.Errorf("provider %q not found", provider)
	}

	err := os.MkdirAll(i.Dir, 0755)
	if err != nil {
		return noMeta, fmt.Errorf("error creating plugins directory: %s", err)
	}

	for _, v := range versions {
		version, err := discovery.VersionStr(v).Parse()
		if err != nil {
			panic(err)
		}

		if req.Allows(version) {
			// provider filename
			name := i.FileName(provider, v)
			path := filepath.Join(i.Dir, name)
			f, err := os.Create(path)
			if err != nil {
				return noMeta, fmt.Errorf("error fetching provider: %s", err)
			}
			f.Close()
			return discovery.PluginMeta{
				Name:    provider,
				Version: discovery.VersionStr(v),
				Path:    path,
			}, nil
		}
	}

	return noMeta, fmt.Errorf("no suitable version for provider %q found with constraints %s", provider, req)
}

func (i *mockProviderInstaller) PurgeUnused(map[string]discovery.PluginMeta) (discovery.PluginMetaSet, error) {
	i.PurgeUnusedCalled = true
	ret := make(discovery.PluginMetaSet)
	ret.Add(discovery.PluginMeta{
		Name:    "test",
		Version: "0.0.0",
		Path:    "mock-test",
	})
	return ret, nil
}

type callbackPluginInstaller func(provider string, req discovery.Constraints) (discovery.PluginMeta, error)

func (cb callbackPluginInstaller) Get(provider string, req discovery.Constraints) (discovery.PluginMeta, error) {
	return cb(provider, req)
}

func (cb callbackPluginInstaller) PurgeUnused(map[string]discovery.PluginMeta) (discovery.PluginMetaSet, error) {
	// does nothing
	return make(discovery.PluginMetaSet), nil
}
