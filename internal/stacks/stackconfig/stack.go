package stackconfig

import (
	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/terraform/internal/addrs"
)

// Stack represents a single stack, which can potentially call other
// "embedded stacks" in a similar manner to how Terraform modules can call
// other modules.
type Stack struct {
	SourceAddr sourceaddrs.Source

	// EmbeddedStacks are calls to other stack configurations that should
	// be treated as a part of the overall desired state produced from this
	// stack. These are declared with "stack" blocks in the stack language.
	EmbeddedStacks map[string]*EmbeddedStack

	// Components are calls to trees of Terraform modules that represent the
	// real infrastructure described by a stack.
	Components map[string]*Component

	// InputVariables, LocalValues, and OutputValues together represent all
	// of the "named values" in the stack configuration, which are just glue
	// to pass values between scopes or to factor out common expressions for
	// reuse in multiple locations.
	InputVariables map[string]*InputVariable
	LocalValues    map[string]*LocalValue
	OutputValues   map[string]*OutputValue

	// ProviderConfigs are the provider configurations declared in this
	// particular stack configuration. Other stack configurations in the
	// overall tree might have their own provider configurations.
	ProviderConfigs map[addrs.Provider]map[string]*ProviderConfig
}
