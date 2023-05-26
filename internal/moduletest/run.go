package moduletest

import (
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type Run struct {
	Name   string
	Status Status

	Diagnostics tfdiags.Diagnostics
}
