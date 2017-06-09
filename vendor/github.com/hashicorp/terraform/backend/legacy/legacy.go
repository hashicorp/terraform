// Package legacy contains a backend implementation that can be used
// with the legacy remote state clients.
package legacy

import (
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state/remote"
)

// Init updates the backend/init package map of initializers to support
// all the remote state types.
//
// If a type is already in the map, it will not be added. This will allow
// us to slowly convert the legacy types to first-class backends.
func Init(m map[string]func() backend.Backend) {
	for k, _ := range remote.BuiltinClients {
		if _, ok := m[k]; !ok {
			// Copy the "k" value since the variable "k" is reused for
			// each key (address doesn't change).
			typ := k

			// Build the factory function to return a backend of typ
			m[k] = func() backend.Backend {
				return &Backend{Type: typ}
			}
		}
	}
}
