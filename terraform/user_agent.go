package terraform

import (
	"fmt"
	"runtime"
)

// The standard Terraform User-Agent format
const UserAgent = "Terraform %s (%s)"

// Generate a UserAgent string
func UserAgentString() string {
	return fmt.Sprintf(UserAgent, VersionString(), runtime.Version())
}
