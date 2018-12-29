package statemgr

import (
	"fmt"

	uuid "github.com/hashicorp/go-uuid"

	"github.com/hashicorp/terraform/states/statefile"
	"github.com/hashicorp/terraform/version"
)

// NewLineage generates a new lineage identifier string. A lineage identifier
// is an opaque string that is intended to be unique in space and time, chosen
// when state is recorded at a location for the first time and then preserved
// afterwards to allow Terraform to recognize when one state snapshot is a
// predecessor or successor of another.
func NewLineage() string {
	lineage, err := uuid.GenerateUUID()
	if err != nil {
		panic(fmt.Errorf("Failed to generate lineage: %v", err))
	}
	return lineage
}

// NewStateFile creates a new statefile.File object, with a newly-minted
// lineage identifier and serial 0, and returns a pointer to it.
func NewStateFile() *statefile.File {
	return &statefile.File{
		Lineage:          NewLineage(),
		TerraformVersion: version.SemVer,
	}
}
