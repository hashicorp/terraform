package command

import (
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/cloud"
)

const failedToLoadSchemasMessage = `
Terraform failed to load schemas, which will in turn affect its ability to generate the
external JSON state file. This will not have any adverse effects on Terraforms ability
to maintain state information, but may have adverse effects on any external integrations
relying on this format. The file should be created on the next successful "terraform apply"
however, historic state information may be missing if the affected integration relies on that

%s
`

func isCloudMode(b backend.Enhanced) bool {
	_, ok := b.(*cloud.Cloud)

	return ok
}
