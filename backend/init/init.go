// Package init contains the list of backends that can be initialized and
// basic helper functions for initializing those backends.
package init

import (
	"sync"

	"github.com/hashicorp/terraform/backend"

	backendatlas "github.com/hashicorp/terraform/backend/atlas"
	backendlegacy "github.com/hashicorp/terraform/backend/legacy"
	backendlocal "github.com/hashicorp/terraform/backend/local"
	backendconsul "github.com/hashicorp/terraform/backend/remote-state/consul"
	backendinmem "github.com/hashicorp/terraform/backend/remote-state/inmem"
	backendS3 "github.com/hashicorp/terraform/backend/remote-state/s3"
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
		"atlas":  func() backend.Backend { return &backendatlas.Backend{} },
		"local":  func() backend.Backend { return &backendlocal.Local{} },
		"consul": func() backend.Backend { return backendconsul.New() },
		"inmem":  func() backend.Backend { return backendinmem.New() },
		"s3":     func() backend.Backend { return backendS3.New() },
	}

	// Add the legacy remote backends that haven't yet been convertd to
	// the new backend API.
	backendlegacy.Init(backends)
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
