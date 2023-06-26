package moduletest

import (
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type Run struct {
	Config *configs.TestRun

	Name   string
	Status Status

	Diagnostics tfdiags.Diagnostics
}
