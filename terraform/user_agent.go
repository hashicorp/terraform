package terraform

import (
	"github.com/hashicorp/terraform/httpclient"
)

// Generate a UserAgent string
//
// Deprecated: Use httpclient.UserAgent(version) instead
func UserAgentString() string {
	return httpclient.UserAgentString()
}
