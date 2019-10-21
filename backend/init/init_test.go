package init

import (
	"reflect"
	"testing"
)

func TestInit_backend(t *testing.T) {
	// Initialize the backends map
	Init(nil)

	backends := []struct {
		Name string
		Type string
	}{
		{"local", "*local.Local"},
		{"remote", "*remote.Remote"},
		{"atlas", "*atlas.Backend"},
		{"azurerm", "*azure.Backend"},
		{"consul", "*consul.Backend"},
		{"etcdv3", "*etcd.Backend"},
		{"gcs", "*gcs.Backend"},
		{"inmem", "*inmem.Backend"},
		{"manta", "*manta.Backend"},
		{"pg", "*pg.Backend"},
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
