package moduletest

import (
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type File struct {
	Config *configs.TestFile

	Name   string
	Status Status

	Runs []*Run

	Diagnostics tfdiags.Diagnostics
}
