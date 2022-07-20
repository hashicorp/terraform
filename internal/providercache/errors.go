package providercache

import "github.com/hashicorp/terraform/internal/getproviders"

// ErrProviderChecksumMiss is an error type used to indicate a provider
// installation failed due to a mismatch in the terraform provider lock file.
type ErrProviderChecksumMiss struct {
	Meta getproviders.PackageMeta
	Msg  string
}

func (err ErrProviderChecksumMiss) Error() string {
	return err.Msg
}
