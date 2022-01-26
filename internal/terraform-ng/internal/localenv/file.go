package localenv

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// DefinitionFile represents the content of an environment definition file.
//
// Definitions for local environments (but _not_ for remote environments)
// live in the local filesystem in .tfenv.hcl files. The definition of a
// local environment consists of a location of a Terraform configuration
// definining zero or more components, values for each of the input variables
// required by that configuration, and configuration for where to store
// persistent shared data such as the Terraform state for each component.
//
// This type represents an in-memory snapshot of what was in the file at the
// most recent time the object was synchronized from disk. Concurrent
// modifications of the file while an instance of this type exists will result
// in data loss.
type DefinitionFile struct {
	// All accessors must lock this mutex across any reads or writes of other
	// fields, and ensure that the "raw"-prefixed fields are always in sync
	// with the other fields at any instant when this lock is not held.
	mu sync.Mutex

	// sourceAddr is the currently-specified location of the configuration for
	// this environment. This is slightly misusing the addrs.ModuleSource
	// type for the sake of prototyping: whereas ModuleSource normally refers
	// to a directory containing .tf files defining a module, this particular
	// use of ModuleSource actually refers to a .tfcomponents.hcl file which
	// will then in turn refer to actual module source addresses.
	sourceAddr addrs.ModuleSource

	// variableValues are the currently-specified values for the input
	// variables of the configuration specified in sourceAddr.
	variableValues map[string]cty.Value

	// storageConfig is the currently-specified state storage configuration
	// for this environment. This defines a systematic rule for deciding how
	// to store persistent state for each component declared in the
	// configuration specified by sourceAddr.
	storageConfig *Storage

	// filePath is the path to the on-disk representation of this environment
	// definition. This is both where the data was originally read from and
	// where we'll write it back to if a caller uses the Save() method.
	filePath string

	// rawContents is a representation of the direct syntax of the underlying
	// file, which we'll modify in-place with any call to a setter method
	// so that a subsequent Save call will write the updated settings to disk.
	rawContents *hclwrite.File

	// rawVariablesBlock is a block guaranteed to belong to the file stored in
	// rawContents, representing the the nested "variables" block, if any.
	// If the file doesn't yet have a variables block then this field is nil.
	rawVariablesBlock *hclwrite.Block

	// rawStorageBlock is a block guaranteed to belong to the file stored
	// in rawContents, representing the nested "storage" block, if any.
	// If the file doesn't yet have such a block then this field is nil.
	rawStorageBlock *hclwrite.Block
}

func ValidEnvironmentFilename(filename string) bool {
	filename = filepath.Clean(filename)
	return strings.HasSuffix(filename, ".tfenv.hcl")
}

func NewDefinitionFile(filename string, configAddr addrs.ModuleSource) (*DefinitionFile, error) {
	// FIXME: If we do something like this in real (i.e. non-prototype) code
	// then we should be careful to avoid implicitly overwriting an existing
	// file here, so that there's a clear distinction between creating a new
	// environment vs. using and possibly updating an existing one.

	ret := &DefinitionFile{
		sourceAddr:  configAddr,
		filePath:    filepath.Clean(filename),
		rawContents: hclwrite.NewEmptyFile(),
	}

	ret.rawContents.Body().SetAttributeValue("config", cty.StringVal(configAddr.ForDisplay()))

	err := ret.Save()
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func OpenDefinitionFile(filename string) (*DefinitionFile, error) {
	filename = filepath.Clean(filename)
	src, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Here we use both hclsyntax to get the meaning of the file and hclwrite
	// to capture its raw syntax, so that we can both understand the values
	// currently saved and make surgical updates to those values when needed.
	rf, diags := hclsyntax.ParseConfig(src, filename, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags // diagnostics packaged as an error
	}
	wf, diags := hclwrite.ParseConfig(src, filename, hcl.InitialPos)
	if diags.HasErrors() {
		// It would be weird to get here because hclwrite should encounter
		// no more errors than hclsyntax does, but we'll handle it anyway.
		return nil, diags // diagnostics packaged as an error
	}

	content, diags := rf.Body.Content(&hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "config", Required: true},
		},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "variables"},
			{Type: "storage", LabelNames: []string{"type"}},
		},
	})
	if diags.HasErrors() {
		return nil, diags // diagnostics packaged as an error
	}

	ret := &DefinitionFile{
		filePath: filename,
	}

	configVal, diags := content.Attributes["config"].Expr.Value(nil)
	if diags.HasErrors() {
		return nil, diags // diagnostics packaged as an error
	}
	var sourceAddrRaw string
	err = gocty.FromCtyValue(configVal, &sourceAddrRaw)
	if err != nil {
		// FIXME: A diagnostic referring to the expression would be better
		return nil, fmt.Errorf("invalid config source: %w", err)
	}
	ret.sourceAddr, err = addrs.ParseModuleSource(sourceAddrRaw)
	if err != nil {
		// FIXME: A diagnostic referring to the expression would be better
		return nil, fmt.Errorf("invalid config source: %w", err)
	}

	for _, block := range content.Blocks {
		switch block.Type {
		case "variables":
			if ret.variableValues != nil {
				return nil, fmt.Errorf("duplicate variables block at %s", block.DefRange)
			}
			attrs, diags := block.Body.JustAttributes()
			if diags.HasErrors() {
				return nil, diags // diagnostics packaged as an error
			}
			ret.variableValues = make(map[string]cty.Value, len(attrs))
			for name, attr := range attrs {
				v, diags := attr.Expr.Value(nil)
				if diags.HasErrors() {
					return nil, diags // diagnostics packaged as an error
				}
				ret.variableValues[name] = v
			}

		case "storage":
			if ret.storageConfig != nil {
				return nil, fmt.Errorf("duplicate storage block at %s", block.DefRange)
			}
			typeAddr := block.Labels[0]
			attrs, diags := block.Body.JustAttributes()
			if diags.HasErrors() {
				return nil, diags // diagnostics packaged as an error
			}
			config := make(map[string]cty.Value, len(attrs))
			for name, attr := range attrs {
				v, diags := attr.Expr.Value(nil)
				if diags.HasErrors() {
					return nil, diags // diagnostics packaged as an error
				}
				config[name] = v
			}
			ret.storageConfig = NewStorage(typeAddr, config)

		default:
			// We didn't declare any other block types in our schema, so we
			// shouldn't be able to get here.
			panic(fmt.Sprintf("undeclared block type %q", block.Type))
		}
	}

	ret.rawContents = wf
	for _, block := range wf.Body().Blocks() {
		switch block.Type() {
		case "variables":
			ret.rawVariablesBlock = block
		case "storage":
			ret.rawStorageBlock = block
		}
	}

	return ret, nil
}

func (f *DefinitionFile) Filename() string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.filePath
}

func (f *DefinitionFile) Save() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// For now we'll just overwrite the file directly. If we were to do
	// something this in a real (not prototype) implementation then we should
	// make some extra effort to ensure the file appears to update atomicly,
	// and retains its exact previous value on any error.
	return os.WriteFile(f.filePath, f.rawContents.Bytes(), os.ModePerm)
}

func (f *DefinitionFile) ConfigSourceAddr() addrs.ModuleSource {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sourceAddr
}

func (f *DefinitionFile) SetConfigSourceAddr(new addrs.ModuleSource) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.sourceAddr = new
	f.rawContents.Body().SetAttributeValue("config", cty.StringVal(new.ForDisplay()))
}

func (f *DefinitionFile) VariablesVal() cty.Value {
	f.mu.Lock()
	defer f.mu.Unlock()
	return cty.ObjectVal(f.variableValues)
}

func (f *DefinitionFile) SetVariables(new map[string]cty.Value) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.rawVariablesBlock == nil {
		f.rawVariablesBlock = f.rawContents.Body().AppendNewBlock("variables", nil)
	}

	varNames := make([]string, 0, len(new))
	f.variableValues = make(map[string]cty.Value, len(new))
	for k, v := range new {
		f.variableValues[k] = v
		varNames = append(varNames, k)
	}
	sort.Strings(varNames)

	// We take a kinda arduous path here with the intent of preserving any
	// comments and manual ordering of pre-existing declarations, surgically
	// modifying them in-place, while appending to the end any new ones,
	// rather than just emptying the block and starting over.
	for _, k := range varNames {
		v := new[k]

		f.rawVariablesBlock.Body().SetAttributeValue(k, v)
	}
	for k := range f.rawVariablesBlock.Body().Attributes() {
		if _, declared := new[k]; !declared {
			f.rawVariablesBlock.Body().RemoveAttribute(k)
		}
	}
}

func (f *DefinitionFile) SetVariable(name string, value cty.Value) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.rawVariablesBlock == nil {
		f.rawVariablesBlock = f.rawContents.Body().AppendNewBlock("variables", nil)
	}
	f.rawVariablesBlock.Body().SetAttributeValue(name, value)
}

func (f *DefinitionFile) StorageConfig() *Storage {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.storageConfig
}

func (f *DefinitionFile) SetStorageConfig(new *Storage) {
	if new == nil {
		if f.rawStorageBlock != nil {
			f.rawContents.Body().RemoveBlock(f.rawStorageBlock)
			f.rawStorageBlock = nil
		}
		return
	}

	newLabels := []string{new.typeAddr}
	if f.rawStorageBlock == nil {
		f.rawStorageBlock = f.rawContents.Body().AppendNewBlock("storage", newLabels)
	} else {
		f.rawStorageBlock.SetLabels(newLabels)
		f.rawStorageBlock.Body().Clear()
	}

	for k, v := range new.config {
		f.rawStorageBlock.Body().SetAttributeValue(k, v)
	}
}
