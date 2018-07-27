package registry

import (
	"fmt"

	"github.com/hashicorp/terraform/registry/regsrc"
)

type errModuleNotFound struct {
	addr *regsrc.Module
}

func (e *errModuleNotFound) Error() string {
	return fmt.Sprintf("module %s not found", e.addr)
}

// IsModuleNotFound returns true only if the given error is a "module not found"
// error. This allows callers to recognize this particular error condition
// as distinct from operational errors such as poor network connectivity.
func IsModuleNotFound(err error) bool {
	_, ok := err.(*errModuleNotFound)
	return ok
}

type errProviderNotFound struct {
	addr *regsrc.TerraformProvider
}

func (e *errProviderNotFound) Error() string {
	return fmt.Sprintf("provider %s not found", e.addr)
}

// IsProviderNotFound returns true only if the given error is a "provider not found"
// error. This allows callers to recognize this particular error condition
// as distinct from operational errors such as poor network connectivity.
func IsProviderNotFound(err error) bool {
	_, ok := err.(*errProviderNotFound)
	return ok
}
