package init

import (
	"os"
	"reflect"
	"testing"

	backendLocal "github.com/hashicorp/terraform/backend/local"
)

func TestInit_backend(t *testing.T) {
	// Initialize the backends map
	Init(nil)

	backends := []struct {
		Name string
		Type string
	}{
		{"local", "*local.Local"},
		{"atlas", "*atlas.Backend"},
		{"azurerm", "*azure.Backend"},
		{"consul", "*consul.Backend"},
		{"etcdv3", "*etcd.Backend"},
		{"gcs", "*gcs.Backend"},
		{"inmem", "*inmem.Backend"},
		{"manta", "*manta.Backend"},
		{"s3", "*s3.Backend"},
		{"swift", "*swift.Backend"},
		{"azure", "init.deprecatedBackendShim"},
	}

	// Make sure we get the requested backend
	for _, b := range backends {
		t.Run(b.Name, func(t *testing.T) {
			f := Backend(b.Name)
			if f == nil {
				t.Fatalf("backend %q is not present; should be", b.Name)
			}
			bType := reflect.TypeOf(f()).String()
			if bType != b.Type {
				t.Fatalf("expected backend %q to be %q, got: %q", b.Name, b.Type, bType)
			}
		})
	}
}

func TestInit_forceLocalBackend(t *testing.T) {
	// Initialize the backends map
	Init(nil)

	enhancedBackends := []struct {
		Name string
		Type string
	}{
		{"local", "nil"},
	}

	// Set the TF_FORCE_LOCAL_BACKEND flag so all enhanced backends will
	// return a local.Local backend with themselves as embedded backend.
	if err := os.Setenv("TF_FORCE_LOCAL_BACKEND", "1"); err != nil {
		t.Fatalf("error setting environment variable TF_FORCE_LOCAL_BACKEND: %v", err)
	}
	defer os.Unsetenv("TF_FORCE_LOCAL_BACKEND")

	// Make sure we always get the local backend.
	for _, b := range enhancedBackends {
		f := Backend(b.Name)

		local, ok := f().(*backendLocal.Local)
		if !ok {
			t.Fatalf("expected backend %q to be \"*local.Local\", got: %T", b.Name, f())
		}

		bType := "nil"
		if local.Backend != nil {
			bType = reflect.TypeOf(local.Backend).String()
		}

		if bType != b.Type {
			t.Fatalf("expected local.Backend to be %s, got: %s", b.Type, bType)
		}
	}
}
