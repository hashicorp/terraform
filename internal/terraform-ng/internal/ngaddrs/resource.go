package ngaddrs

import (
	"github.com/hashicorp/terraform/internal/addrs"
)

type AbsResourceInstance struct {
	Component    AbsComponent
	ResourceInst addrs.AbsResourceInstance
}
