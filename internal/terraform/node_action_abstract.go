package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/providers"
)

// NodeAbstractAction represents an action that has no associated
// operations.
type NodeAbstractAction struct {
	Addr   addrs.ConfigAction
	Config configs.Action

	// The fields below will be automatically set using the Attach interfaces if
	// you're running those transforms, but also can be explicitly set if you
	// already have that information.

	// The address of the provider this resource will use
	ResolvedProvider addrs.AbsProviderConfig
	Schema           *providers.ActionSchema
}

// NewNodeAbstractAction creates an abstract action graph node for
// the given action config address.
func NewNodeAbstractAction(addr addrs.ConfigAction, config configs.Action) *NodeAbstractAction {
	return &NodeAbstractAction{
		Addr:   addr,
		Config: config, // we don't have an "attach action config" transformer
	}
}

var (
	_ GraphNodeModuleInstance     = (*NodeValidatableAction)(nil)
	_ GraphNodeExecutable         = (*NodeValidatableAction)(nil)
	_ GraphNodeReferenceable      = (*NodeValidatableAction)(nil)
	_ GraphNodeReferencer         = (*NodeValidatableAction)(nil)
	_ GraphNodeConfigAction       = (*NodeValidatableAction)(nil)
	_ GraphNodeAttachActionSchema = (*NodeValidatableAction)(nil)
	_ GraphNodeProviderConsumer   = (*NodeValidatableAction)(nil)
)

func (n NodeAbstractAction) Name() string {
	return n.Addr.String()
}

// ConcreteResourceNodeFunc is a callback type used to convert an
// abstract action to a concrete one of some type.
type ConcreteActionNodeFunc func(*NodeAbstractAction) dag.Vertex

func (n NodeAbstractAction) ActionAddr() addrs.ConfigAction {
	return n.Addr
}

func (n NodeAbstractAction) ModulePath() addrs.Module {
	return n.Addr.Module
}

func (n *NodeAbstractAction) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr.Action}
}

func (n *NodeAbstractAction) References() []*addrs.Reference {
	var result []*addrs.Reference
	c := n.Config

	refs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, c.Count)
	result = append(result, refs...)
	refs, _ = langrefs.ReferencesInExpr(addrs.ParseRef, c.ForEach)
	result = append(result, refs...)

	if n.Schema != nil {
		refs, _ = langrefs.ReferencesInBlock(addrs.ParseRef, c.Config, n.Schema.ConfigSchema)
		result = append(result, refs...)
	}

	return result
}

func (n *NodeAbstractAction) AttachActionSchema(schema *providers.ActionSchema) {
	n.Schema = schema
}

func (n *NodeAbstractAction) ProvidedBy() (addrs.ProviderConfig, bool) {
	if n.ResolvedProvider.Provider.Type != "" {
		return n.ResolvedProvider, true
	}

	relAddr := n.Config.ProviderConfigAddr()
	return addrs.LocalProviderConfig{
		LocalName: relAddr.LocalName,
		Alias:     relAddr.Alias,
	}, false
}

func (n *NodeAbstractAction) Provider() addrs.Provider {
	if n.ResolvedProvider.Provider.Type != "" {
		return n.ResolvedProvider.Provider
	}

	return addrs.ImpliedProviderForUnqualifiedType(n.Addr.Action.ImpliedProvider())
}

func (n *NodeAbstractAction) SetProvider(p addrs.AbsProviderConfig) {
	n.ResolvedProvider = p
}
