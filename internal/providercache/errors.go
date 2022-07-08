package providercache

import "github.com/hashicorp/terraform/internal/getproviders"

type ErrChecksumMiss struct {
	Meta getproviders.PackageMeta
	Msg  string
}

func (err ErrChecksumMiss) Error() string {
	return err.Msg
}
