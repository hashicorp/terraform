package webcommand

import (
	"context"
	"net/url"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// URLProvider is an optional interface that a backend can implement to support
// the "terraform web" command.
//
// Currently only the Terraform Cloud integration supports this interface and
// the UI code in command.WebCommand assumes that in its error messaging. If
// any other backend supports this in future that error messaging must be
// updated.
type URLProvider interface {
	WebURLForObject(ctx context.Context, workspaceName string, targetObject TargetObject) (*url.URL, tfdiags.Diagnostics)
}
