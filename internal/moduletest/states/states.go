package states

import (
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/states"
)

type TestRunState struct {
	File *moduletest.File
	Run  *moduletest.Run

	Manifest *TestRunManifest
	State    *states.State
}
