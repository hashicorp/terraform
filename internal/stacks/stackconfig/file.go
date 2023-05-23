package stackconfig

import "github.com/hashicorp/go-slug/sourceaddrs"

// File represents the content of a single .tfstack.hcl or .tfstack.json file
// before it's been merged with its siblings in the same directory to produce
// the overall [Stack] object.
type File struct {
	// SourceAddr is the source location for this particular file, meaning
	// that the "sub-path" portion of the address should always be populated
	// and refer to a particular file rather than to a directory.
	SourceAddr sourceaddrs.Source

	// The remaining fields in here correspond to the fields of the same name in
	// [Stack].
	Components      map[string]*Component
	EmbeddedStacks  map[string]*EmbeddedStack
	InputVariables  map[string]*InputVariable
	LocalValues     map[string]*LocalValue
	OutputValues    map[string]*OutputValue
	ProviderConfigs map[string]*ProviderConfig
}
