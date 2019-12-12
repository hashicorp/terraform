// Package resources contains shared functionality for processing resource
// configurations.
package resources

import (
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/lang"
)

// ResourceDependencies analyzes a resource configuration and finds the
// dependencies both implied by references in expressions and defined
// explicitly using "depends_on".
func ResourceDependencies(c *configs.Resource, schema *configschema.Block, provisionerSchemas map[string]*configschema.Block) []addrs.Referenceable {
	var result []addrs.Referenceable

	for _, traversal := range c.DependsOn {
		ref, diags := addrs.ParseRef(traversal)
		if diags.HasErrors() {
			// We ignore this here, because this isn't a suitable place to return
			// errors. This situation should be caught and rejected during
			// validation.
			log.Printf("[ERROR] Can't parse %#v from depends_on as reference: %s", traversal, diags.Err())
			continue
		}

		result = append(result, ref.Subject)
	}

	// We intentionally ignore errors here because detecting them is the
	// responsibility of the validation step. In case of errors, we'll return
	// the subset of the dependencies that are valid.
	refs, _ := lang.ReferencesInExpr(c.Count)
	for _, ref := range refs {
		result = append(result, ref.Subject)
	}
	refs, _ = lang.ReferencesInExpr(c.ForEach)
	for _, ref := range refs {
		result = append(result, ref.Subject)
	}
	refs, _ = lang.ReferencesInBlock(c.Config, schema)
	for _, ref := range refs {
		result = append(result, ref.Subject)
	}
	if c.Managed != nil {
		for _, p := range c.Managed.Provisioners {
			if p.When != configs.ProvisionerWhenCreate {
				continue
			}
			if p.Connection != nil {
				refs, _ = lang.ReferencesInBlock(p.Connection.Config, connectionBlockSupersetSchema)
				for _, ref := range refs {
					result = append(result, ref.Subject)
				}
			}

			schema := provisionerSchemas[p.Type]
			if schema == nil {
				log.Printf("[WARN] no schema for provisioner %q is available, so provisioner block references cannot be detected", p.Type)
			}
			refs, _ = lang.ReferencesInBlock(p.Config, schema)
			for _, ref := range refs {
				result = append(result, ref.Subject)
			}
		}
	}
	return result
}
