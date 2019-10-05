package projects

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/projects/projectconfigs"
	"github.com/hashicorp/terraform/projects/projectlang"
)

type staticEvalData struct {
	config *projectconfigs.Config
}

func newStaticEvalData(config *projectconfigs.Config) *staticEvalData {
	return &staticEvalData{config: config}
}

var _ projectlang.StaticEvaluateData = (*staticEvalData)(nil)

func (d *staticEvalData) BaseDir() string {
	return d.config.ProjectRoot
}

func (d *staticEvalData) LocalValueExpr(addr addrs.LocalValue) hcl.Expression {
	return d.config.Locals[addr.Name].Value
}

func (d *staticEvalData) WorkspaceConfigForEachExpr(addr addrs.ProjectWorkspaceConfig) hcl.Expression {
	switch addr.Rel {
	case addrs.ProjectWorkspaceCurrent:
		return d.config.Workspaces[addr.Name].ForEach
	case addrs.ProjectWorkspaceUpstream:
		return d.config.Upstreams[addr.Name].ForEach
	default:
		panic(fmt.Sprintf("invalid workspace relationship"))
	}
}
