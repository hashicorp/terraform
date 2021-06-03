package registry

import (
	"fmt"

	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/hashicorp/terraform/internal/registry/regsrc"
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

// IsServiceNotProvided returns true only if the given error is a "service not provided"
// error. This allows callers to recognize this particular error condition
// as distinct from operational errors such as poor network connectivity.
func IsServiceNotProvided(err error) bool {
	_, ok := err.(*disco.ErrServiceNotProvided)
	return ok
}

// ServiceUnreachableError Registry service is unreachable
type ServiceUnreachableError struct {
	err error
}

func (e *ServiceUnreachableError) Error() string {
	return e.err.Error()
}

// IsServiceUnreachable returns true if the registry/discovery service was unreachable
func IsServiceUnreachable(err error) bool {
	_, ok := err.(*ServiceUnreachableError)
	return ok
}
