package terraform

import (
	backendInit "github.com/hashicorp/terraform/internal/backend/init"
)

func init() {
	// Initialize the backends
	backendInit.Init(nil)
}
