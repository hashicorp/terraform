package terraform

import (
	"fmt"
	"runtime"

	"github.com/hashicorp/terraform/version"
)

// The standard Terraform User-Agent format
const UserAgent = "Terraform %s (%s)"

// Generate a UserAgent string
func UserAgentString() string {
	return fmt.Sprintf(UserAgent, version.String(), runtime.Version())
}
