// Package init contains the list of backends that can be initialized and
// basic helper functions for initializing those backends.
package init

import (
	"sync"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/tfdiags"

	backendatlas "github.com/hashicorp/terraform/backend/atlas"
	backendlocal "github.com/hashicorp/terraform/backend/local"
	backendartifactory "github.com/hashicorp/terraform/backend/remote-state/artifactory"
	backendAzure "github.com/hashicorp/terraform/backend/remote-state/azure"
	backendconsul "github.com/hashicorp/terraform/backend/remote-state/consul"
	backendetcdv2 "github.com/hashicorp/terraform/backend/remote-state/etcdv2"
	backendetcdv3 "github.com/hashicorp/terraform/backend/remote-state/etcdv3"
	backendGCS "github.com/hashicorp/terraform/backend/remote-state/gcs"
	backendhttp "github.com/hashicorp/terraform/backend/remote-state/http"
	backendinmem "github.com/hashicorp/terraform/backend/remote-state/inmem"
	backendManta "github.com/hashicorp/terraform/backend/remote-state/manta"
	backendS3 "github.com/hashicorp/terraform/backend/remote-state/s3"
	backendSwift "github.com/hashicorp/terraform/backend/remote-state/swift"

	"github.com/zclconf/go-cty/cty"
)

// backends is the list of available backends. This is a global variable
// because backends are currently hardcoded into Terraform and can't be
// modified without recompilation.
//
// To read an available backend, use the Backend function. This ensures
// safe concurrent read access to the list of built-in backends.
//
// Backends are hardcoded into Terraform because the API for backends uses
// complex structures and supporting that over the plugin system is currently
// prohibitively difficult. For those wanting to implement a custom backend,
// they can do so with recompilation.
var backends map[string]func() backend.Backend
var backendsLock sync.Mutex

func init() {
	// Our hardcoded backends. We don't need to acquire a lock here
	// since init() code is serial and can't spawn goroutines.
	backends = map[string]func() backend.Backend{
		"artifactory": func() backend.Backend { return backendartifactory.New() },
		"atlas":       func() backend.Backend { return &backendatlas.Backend{} },
		"http":        func() backend.Backend { return backendhttp.New() },
		"local":       func() backend.Backend { return &backendlocal.Local{} },
		"consul":      func() backend.Backend { return backendconsul.New() },
		"inmem":       func() backend.Backend { return backendinmem.New() },
		"swift":       func() backend.Backend { return backendSwift.New() },
		"s3":          func() backend.Backend { return backendS3.New() },
		"azurerm":     func() backend.Backend { return backendAzure.New() },
		"etcd":        func() backend.Backend { return backendetcdv2.New() },
		"etcdv3":      func() backend.Backend { return backendetcdv3.New() },
		"gcs":         func() backend.Backend { return backendGCS.New() },
		"manta":       func() backend.Backend { return backendManta.New() },

		"azure": func() backend.Backend {
			return deprecateBackend(
				backendAzure.New(),
				`Warning: "azure" name is deprecated, please use "azurerm"`,
			)
		},
	}
}

// Backend returns the initialization factory for the given backend, or
// nil if none exists.
func Backend(name string) func() backend.Backend {
	backendsLock.Lock()
	defer backendsLock.Unlock()
	return backends[name]
}

// Set sets a new backend in the list of backends. If f is nil then the
// backend will be removed from the map. If this backend already exists
// then it will be overwritten.
//
// This method sets this backend globally and care should be taken to do
// this only before Terraform is executing to prevent odd behavior of backends
// changing mid-execution.
func Set(name string, f func() backend.Backend) {
	backendsLock.Lock()
	defer backendsLock.Unlock()

	if f == nil {
		delete(backends, name)
		return
	}

	backends[name] = f
}

// deprecatedBackendShim is used to wrap a backend and inject a deprecation
// warning into the Validate method.
type deprecatedBackendShim struct {
	backend.Backend
	Message string
}

// ValidateConfig delegates to the wrapped backend to validate its config
// and then appends shim's deprecation warning.
func (b deprecatedBackendShim) ValidateConfig(obj cty.Value) tfdiags.Diagnostics {
	diags := b.Backend.ValidateConfig(obj)
	return diags.Append(tfdiags.SimpleWarning(b.Message))
}

// DeprecateBackend can be used to wrap a backend to retrun a deprecation
// warning during validation.
func deprecateBackend(b backend.Backend, message string) backend.Backend {
	// Since a Backend wrapped by deprecatedBackendShim can no longer be
	// asserted as an Enhanced or Local backend, disallow those types here
	// entirely.  If something other than a basic backend.Backend needs to be
	// deprecated, we can add that functionality to schema.Backend or the
	// backend itself.
	if _, ok := b.(backend.Enhanced); ok {
		panic("cannot use DeprecateBackend on an Enhanced Backend")
	}

	if _, ok := b.(backend.Local); ok {
		panic("cannot use DeprecateBackend on a Local Backend")
	}

	return deprecatedBackendShim{
		Backend: b,
		Message: message,
	}
}
