package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type InitGraphBuilder struct {
	Config *configs.Config

	RootVariableValues InputValues

	Walker configs.ModuleWalker
}

func (b *InitGraphBuilder) Build(path addrs.ModuleInstance) (*Graph, tfdiags.Diagnostics) {
	log.Printf("[TRACE] building graph for terraform dependencies")
	return (&BasicGraphBuilder{
		Steps: b.Steps(),
		Name:  "InitGraphBuilder",
	}).Build(path)
}

func (b *InitGraphBuilder) Steps() []GraphTransformer {
	steps := []GraphTransformer{}

	if b.Config.Parent == nil {
		steps = append(steps, &RootVariableTransformer{
			Config:    b.Config,
			RawValues: b.RootVariableValues,
		})
	} else {
		steps = append(steps, &ModuleVariableTransformer{
			Config:     b.Config,
			ModuleOnly: true,
		})
	}

	steps = append(steps, []GraphTransformer{
		&ModuleTransformer{
			Config:    b.Config,
			Installer: b.Walker,
		},

		&LocalTransformer{
			Config: b.Config,
		},

		&ReferenceTransformer{},

		&RootTransformer{},

		&TransitiveReductionTransformer{},
	}...)

	return steps
}
