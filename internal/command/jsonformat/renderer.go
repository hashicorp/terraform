package jsonformat

import (
	"github.com/mitchellh/colorstring"

	"github.com/hashicorp/terraform/internal/command/jsonplan"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/terminal"
)

type Plan struct {
	OutputChanges   map[string]jsonplan.Change        `json:"output_changes"`
	ResourceChanges []jsonplan.ResourceChange         `json:"resource_changes"`
	ResourceDrift   []jsonplan.ResourceChange         `json:"resource_drift"`
	ProviderSchemas map[string]*jsonprovider.Provider `json:"provider_schemas"`
}

type Renderer struct {
	Streams  *terminal.Streams
	Colorize *colorstring.Colorize
}

func (r Renderer) RenderPlan(plan Plan) {
	panic("not implemented")
}

func (r Renderer) RenderLog(message map[string]interface{}) {
	panic("not implemented")
}
