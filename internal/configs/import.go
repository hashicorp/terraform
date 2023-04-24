package configs

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
)

type Import struct {
	ID string
	To addrs.AbsResourceInstance

	DeclRange hcl.Range
}
