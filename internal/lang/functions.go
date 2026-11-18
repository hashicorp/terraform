// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package lang

import (
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/hashicorp/hcl/v2/ext/tryfunc"
	ctyyaml "github.com/zclconf/go-cty-yaml"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"

	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/experiments"
	"github.com/hashicorp/terraform/internal/lang/funcs"
)

var impureFunctions = []string{
	"bcrypt",
	"timestamp",
	"uuid",
}

// filesystemFunctions are the functions that allow interacting with arbitrary
// paths in the local filesystem, and which can therefore have their results
// vary based on something other than their arguments, and might allow template
// rendering to expose details about the system where Terraform is running.
var filesystemFunctions = collections.NewSetCmp[string](
	"file",
	"fileexists",
	"fileset",
	"filebase64",
	"filebase64sha256",
	"filebase64sha512",
	"filemd5",
	"filesha1",
	"filesha256",
	"filesha512",
	"templatefile",
)

// templateFunctions are functions that render nested templates. These are
// callable from module code but not from within the templates they are
// rendering.
var templateFunctions = collections.NewSetCmp[string](
	"templatefile",
	"templatestring",
)

// Functions returns the set of functions that should be used to when evaluating
// expressions in the receiving scope.
//
// The returned map must not be modified, because it may be shared with other
// scopes.
func (s *Scope) Functions() map[string]function.Function {
	s.funcsLock.Lock()
	defer s.funcsLock.Unlock()

	if s.funcs == nil {
		s.funcs = s.buildFunctions()
	}
	return s.funcs
}

// buildFunctions returns the table of functions for the receiving scope,
// either from s.FunctionTables or by building a new table.
func (s *Scope) buildFunctions() map[string]function.Function {
	if s.FunctionTables == nil {
		table := s.coreFunctions()
		s.addExternalFunctions(table)
		return table
	}

	core := s.FunctionTables.get(s.functionTableKey(), s.coreFunctions)
	if !s.ExternalFuncs.hasFunctions() {
		return core
	}

	// External functions differ between scopes, so they must be added to a
	// copy of the shared table. The template functions must be able to call
	// the external functions too, so they must refer to the copy.
	table := maps.Clone(core)
	s.addTemplateFunctions(table)
	s.addExternalFunctions(table)
	return table
}

// coreFunctions builds a new table containing all of the built-in functions
// available in the receiving scope, both with and without the "core::"
// namespace prefix.
func (s *Scope) coreFunctions() map[string]function.Function {
	immutableResults := s.immutableResultsWrapper()
	coreFuncs := baseFunctions(s.BaseDir)

	// Modify the functions which behave slightly differently in core.
	// Filesystem functions will check for consistent results, and the
	// template functions are replaced below to close over the correct
	// versions of the filesystem functions.
	coreFuncs["file"] = funcs.MakeFileFunc(s.BaseDir, false, immutableResults("file", s.FunctionResults))
	coreFuncs["fileexists"] = funcs.MakeFileExistsFunc(s.BaseDir, immutableResults("fileexists", s.FunctionResults))
	coreFuncs["fileset"] = funcs.MakeFileSetFunc(s.BaseDir, immutableResults("fileset", s.FunctionResults))
	coreFuncs["filebase64"] = funcs.MakeFileFunc(s.BaseDir, true, immutableResults("filebase64", s.FunctionResults))
	coreFuncs["filebase64sha256"] = funcs.MakeFileBase64Sha256Func(s.BaseDir, immutableResults("filebase64sha256", s.FunctionResults))
	coreFuncs["filebase64sha512"] = funcs.MakeFileBase64Sha512Func(s.BaseDir, immutableResults("filebase64sha512", s.FunctionResults))
	coreFuncs["filemd5"] = funcs.MakeFileMd5Func(s.BaseDir, immutableResults("filemd5", s.FunctionResults))
	coreFuncs["filesha1"] = funcs.MakeFileSha1Func(s.BaseDir, immutableResults("filesha1", s.FunctionResults))
	coreFuncs["filesha256"] = funcs.MakeFileSha256Func(s.BaseDir, immutableResults("filesha256", s.FunctionResults))
	coreFuncs["filesha512"] = funcs.MakeFileSha512Func(s.BaseDir, immutableResults("filesha512", s.FunctionResults))

	if s.ConsoleMode {
		// The type function is only available in terraform console.
		coreFuncs["type"] = funcs.TypeFunc
	}

	if !s.ConsoleMode {
		// The plantimestamp function doesn't make sense in the terraform
		// console.
		coreFuncs["plantimestamp"] = funcs.MakeStaticTimestampFunc(s.PlanTimestamp)
	}

	if s.PureOnly {
		// Force our few impure functions to return unknown so that we
		// can defer evaluating them until a later pass.
		for _, name := range impureFunctions {
			coreFuncs[name] = function.Unpredictable(coreFuncs[name])
		}
	}

	// All of the built-in functions are also available under the "core::"
	// namespace, to distinguish from the "provider::" and "module::"
	// namespaces that can serve as external extension points.
	table := make(map[string]function.Function, len(coreFuncs)*2)
	for name, fn := range coreFuncs {
		if templateFunctions.Has(name) {
			// replaced by addTemplateFunctions below
			continue
		}
		addCoreFunction(table, name, fn)
	}
	s.addTemplateFunctions(table)
	return table
}

// addTemplateFunctions adds the template rendering functions to the given
// table of functions, such that templates can call the functions in that
// table.
func (s *Scope) addTemplateFunctions(table map[string]function.Function) {
	immutableResults := s.immutableResultsWrapper()

	// Our two template-rendering functions want to be able to call
	// all of the other functions themselves, but we pass them indirectly
	// via a callback to avoid chicken/egg problems while initializing
	// the functions table.
	funcsFunc := func() (funcs map[string]function.Function, fsFuncs collections.Set[string], templateFuncs collections.Set[string]) {
		// The templatefile and templatestring functions prevent recursive
		// calls to themselves and each other by copying this map and
		// overwriting the relevant entries.
		return table, filesystemFunctions, templateFunctions
	}
	addCoreFunction(table, "templatefile", funcs.MakeTemplateFileFunc(s.BaseDir, funcsFunc, immutableResults("templatefile", s.FunctionResults)))
	addCoreFunction(table, "templatestring", funcs.MakeTemplateStringFunc(funcsFunc))
}

// addExternalFunctions adds any external functions that the caller provided
// when constructing the receiving scope to the given table of functions. For
// now, that's just provider-contributed functions, under a
// "provider::NAME::" namespace where NAME is the local name of the provider in
// the current module.
func (s *Scope) addExternalFunctions(table map[string]function.Function) {
	for providerLocalName, funcs := range s.ExternalFuncs.Provider {
		for funcName, fn := range funcs {
			name := fmt.Sprintf("provider::%s::%s", providerLocalName, funcName)
			table[name] = fn
		}
	}
}

// immutableResultsWrapper returns the function used to wrap the
// implementations of functions which must return consistent results.
func (s *Scope) immutableResultsWrapper() func(name string, priorResults *FunctionResults) func(fn function.ImplFunc) function.ImplFunc {
	// For backwards compatibility, filesystem functions are allowed to return
	// inconsistent results when called from within a provider configuration, so
	// here we override the checks with a noop wrapper. This misbehavior was
	// found to be used by a number of configurations, which took advantage of
	// it to create the equivalent of ephemeral values before they formally
	// existed in the language.
	if s.ForProvider {
		return filesystemNoopWrapper
	}
	return immutableResults
}

// addCoreFunction adds the given built-in function to the given table of
// functions, both with and without the "core::" namespace prefix.
func addCoreFunction(table map[string]function.Function, name string, fn function.Function) {
	fn = funcs.WithDescription(name, fn)
	table[name] = fn
	table["core::"+name] = fn
}

// TestingFunctions returns the set of functions available to the testing
// framework. Generally, the testing framework doesn't have access to a specific
// state or plan when executing these functions so some of the functions
// available normally are not available during tests.
func TestingFunctions() map[string]function.Function {
	// The baseDir is always the current directory during the tests.
	fs := baseFunctions(".")

	// Add a description to each function and parameter based on the
	// contents of descriptionList.
	// One must create a matching description entry whenever a new
	// function is introduced.
	for name, f := range fs {
		fs[name] = funcs.WithDescription(name, f)
	}

	return fs
}

// baseFunctions loads the set of functions that are used in both the testing
// framework and the main Terraform operations.
func baseFunctions(baseDir string) map[string]function.Function {
	// Some of our functions are just directly the cty stdlib functions.
	// Others are implemented in the subdirectory "funcs" here in this
	// repository. New functions should generally start out their lives
	// in the "funcs" directory and potentially graduate to cty stdlib
	// later if the functionality seems to be something domain-agnostic
	// that would be useful to all applications using cty functions.
	//
	// If you're adding something here, please consider whether it meets
	// the criteria for either or both of the sets [filesystemFunctions]
	// and [templateFunctions] and add it there if so, to ensure that
	// functions relying on those classifications will behave correctly.
	fs := map[string]function.Function{
		"abs":              stdlib.AbsoluteFunc,
		"abspath":          funcs.AbsPathFunc,
		"alltrue":          funcs.AllTrueFunc,
		"anytrue":          funcs.AnyTrueFunc,
		"basename":         funcs.BasenameFunc,
		"base64decode":     funcs.Base64DecodeFunc,
		"base64encode":     funcs.Base64EncodeFunc,
		"base64gzip":       funcs.Base64GzipFunc,
		"base64sha256":     funcs.Base64Sha256Func,
		"base64sha512":     funcs.Base64Sha512Func,
		"bcrypt":           funcs.BcryptFunc,
		"can":              tryfunc.CanFunc,
		"ceil":             stdlib.CeilFunc,
		"chomp":            stdlib.ChompFunc,
		"cidrhost":         funcs.CidrHostFunc,
		"cidrnetmask":      funcs.CidrNetmaskFunc,
		"cidrsubnet":       funcs.CidrSubnetFunc,
		"cidrsubnets":      funcs.CidrSubnetsFunc,
		"coalesce":         funcs.CoalesceFunc,
		"coalescelist":     stdlib.CoalesceListFunc,
		"compact":          stdlib.CompactFunc,
		"concat":           stdlib.ConcatFunc,
		"contains":         stdlib.ContainsFunc,
		"convert":          funcs.ConvertFunc,
		"csvdecode":        stdlib.CSVDecodeFunc,
		"dirname":          funcs.DirnameFunc,
		"distinct":         stdlib.DistinctFunc,
		"element":          stdlib.ElementFunc,
		"endswith":         funcs.EndsWithFunc,
		"ephemeralasnull":  funcs.EphemeralAsNullFunc,
		"chunklist":        stdlib.ChunklistFunc,
		"file":             funcs.MakeFileFunc(baseDir, false, noopWrapper),
		"fileexists":       funcs.MakeFileExistsFunc(baseDir, noopWrapper),
		"fileset":          funcs.MakeFileSetFunc(baseDir, noopWrapper),
		"filebase64":       funcs.MakeFileFunc(baseDir, true, noopWrapper),
		"filebase64sha256": funcs.MakeFileBase64Sha256Func(baseDir, noopWrapper),
		"filebase64sha512": funcs.MakeFileBase64Sha512Func(baseDir, noopWrapper),
		"filemd5":          funcs.MakeFileMd5Func(baseDir, noopWrapper),
		"filesha1":         funcs.MakeFileSha1Func(baseDir, noopWrapper),
		"filesha256":       funcs.MakeFileSha256Func(baseDir, noopWrapper),
		"filesha512":       funcs.MakeFileSha512Func(baseDir, noopWrapper),
		"flatten":          stdlib.FlattenFunc,
		"floor":            stdlib.FloorFunc,
		"format":           stdlib.FormatFunc,
		"formatdate":       stdlib.FormatDateFunc,
		"formatlist":       stdlib.FormatListFunc,
		"indent":           stdlib.IndentFunc,
		"index":            funcs.IndexFunc,
		"join":             stdlib.JoinFunc,
		"jsondecode":       stdlib.JSONDecodeFunc,
		"jsonencode":       stdlib.JSONEncodeFunc,
		"keys":             stdlib.KeysFunc,
		"length":           funcs.LengthFunc,
		"list":             funcs.ListFunc,
		"log":              funcs.LogFunc,
		"lookup":           funcs.LookupFunc,
		"lower":            stdlib.LowerFunc,
		"map":              funcs.MapFunc,
		"matchkeys":        funcs.MatchkeysFunc,
		"max":              stdlib.MaxFunc,
		"md5":              funcs.Md5Func,
		"merge":            stdlib.MergeFunc,
		"min":              stdlib.MinFunc,
		"one":              funcs.OneFunc,
		"parseint":         stdlib.ParseIntFunc,
		"pathexpand":       funcs.PathExpandFunc,
		"pow":              funcs.PowFunc,
		"range":            stdlib.RangeFunc,
		"regex":            stdlib.RegexFunc,
		"regexall":         stdlib.RegexAllFunc,
		"replace":          funcs.ReplaceFunc,
		"reverse":          stdlib.ReverseListFunc,
		"rsadecrypt":       funcs.RsaDecryptFunc,
		"sensitive":        funcs.SensitiveFunc,
		"nonsensitive":     funcs.NonsensitiveFunc,
		"issensitive":      funcs.IssensitiveFunc,
		"setintersection":  stdlib.SetIntersectionFunc,
		"setproduct":       stdlib.SetProductFunc,
		"setsubtract":      stdlib.SetSubtractFunc,
		"setunion":         stdlib.SetUnionFunc,
		"sha1":             funcs.Sha1Func,
		"sha256":           funcs.Sha256Func,
		"sha512":           funcs.Sha512Func,
		"signum":           stdlib.SignumFunc,
		"slice":            stdlib.SliceFunc,
		"sort":             stdlib.SortFunc,
		"split":            stdlib.SplitFunc,
		"startswith":       funcs.StartsWithFunc,
		"strcontains":      funcs.StrContainsFunc,
		"strrev":           stdlib.ReverseFunc,
		"substr":           stdlib.SubstrFunc,
		"sum":              funcs.SumFunc,
		"textdecodebase64": funcs.TextDecodeBase64Func,
		"textencodebase64": funcs.TextEncodeBase64Func,
		"timestamp":        funcs.TimestampFunc,
		"timeadd":          stdlib.TimeAddFunc,
		"timecmp":          funcs.TimeCmpFunc,
		"title":            stdlib.TitleFunc,
		"tostring":         funcs.MakeToFunc(cty.String),
		"tonumber":         funcs.MakeToFunc(cty.Number),
		"tobool":           funcs.MakeToFunc(cty.Bool),
		"toset":            funcs.MakeToFunc(cty.Set(cty.DynamicPseudoType)),
		"tolist":           funcs.MakeToFunc(cty.List(cty.DynamicPseudoType)),
		"tomap":            funcs.MakeToFunc(cty.Map(cty.DynamicPseudoType)),
		"transpose":        funcs.TransposeFunc,
		"trim":             stdlib.TrimFunc,
		"trimprefix":       stdlib.TrimPrefixFunc,
		"trimspace":        stdlib.TrimSpaceFunc,
		"trimsuffix":       stdlib.TrimSuffixFunc,
		"try":              tryfunc.TryFunc,
		"upper":            stdlib.UpperFunc,
		"urlencode":        funcs.URLEncodeFunc,
		"uuid":             funcs.UUIDFunc,
		"uuidv5":           funcs.UUIDV5Func,
		"values":           stdlib.ValuesFunc,
		"yamldecode":       ctyyaml.YAMLDecodeFunc,
		"yamlencode":       ctyyaml.YAMLEncodeFunc,
		"zipmap":           stdlib.ZipmapFunc,
	}

	// Our two template-rendering functions want to be able to call
	// all of the other functions themselves, but we pass them indirectly
	// via a callback to avoid chicken/egg problems while initializing
	// the functions table.
	funcsFunc := func() (funcs map[string]function.Function, fsFuncs collections.Set[string], templateFuncs collections.Set[string]) {
		// The templatefile and templatestring functions prevent recursive
		// calls to themselves and each other by copying this map and
		// overwriting the relevant entries.
		return fs, filesystemFunctions, templateFunctions
	}

	fs["templatefile"] = funcs.MakeTemplateFileFunc(baseDir, funcsFunc, noopWrapper)
	fs["templatestring"] = funcs.MakeTemplateStringFunc(funcsFunc)

	return fs
}

// experimentalFunction checks whether the given experiment is enabled for
// the recieving scope. If so, it will return the given function verbatim.
// If not, it will return a placeholder function that just returns an
// error explaining that the function requires the experiment to be enabled.
//
//lint:ignore U1000 Ignore unused function error for now
func (s *Scope) experimentalFunction(experiment experiments.Experiment, fn function.Function) function.Function {
	if s.activeExperiments.Has(experiment) {
		return fn
	}

	err := fmt.Errorf(
		"this function is experimental and available only when the experiment keyword %s is enabled for the current module",
		experiment.Keyword(),
	)

	return function.New(&function.Spec{
		Params:   fn.Params(),
		VarParam: fn.VarParam(),
		Type: func(args []cty.Value) (cty.Type, error) {
			return cty.DynamicPseudoType, err
		},
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			// It would be weird to get here because the Type function always
			// fails, but we'll return an error here too anyway just to be
			// robust.
			return cty.DynamicVal, err
		},
	})
}

// ExternalFuncs represents functions defined by extension components outside
// of Terraform Core.
//
// This package expects the caller to provide ready-to-use function.Function
// instances for each function, which themselves perform whatever adaptations
// are necessary to translate a call into a form suitable for the external
// component that's contributing the function, and to translate the results
// to conform to the expected function return value conventions.
type ExternalFuncs struct {
	Provider map[string]map[string]function.Function
}

// hasFunctions returns true if there are any external functions.
func (e ExternalFuncs) hasFunctions() bool {
	for _, funcs := range e.Provider {
		if len(funcs) != 0 {
			return true
		}
	}
	return false
}

// FunctionTables is a concurrency-safe cache of tables of built-in functions,
// which can be shared between the scopes used for an operation so that each
// scope doesn't need to build its own table.
//
// A table is only shared between scopes whose settings would otherwise result
// in identical tables. Scopes with external functions use a copy of the
// shared table.
type FunctionTables struct {
	mu     sync.Mutex
	tables map[functionTableKey]map[string]function.Function
}

// functionTableKey represents all of the settings of a Scope which affect its
// table of built-in functions.
type functionTableKey struct {
	baseDir         string
	pureOnly        bool
	consoleMode     bool
	forProvider     bool
	planTimestamp   time.Time
	functionResults *FunctionResults
}

func NewFunctionTables() *FunctionTables {
	return &FunctionTables{
		tables: make(map[functionTableKey]map[string]function.Function),
	}
}

// get returns the table for the given key, calling build to create it if it
// doesn't exist yet.
func (t *FunctionTables) get(key functionTableKey, build func() map[string]function.Function) map[string]function.Function {
	t.mu.Lock()
	defer t.mu.Unlock()

	table, ok := t.tables[key]
	if !ok {
		table = build()
		t.tables[key] = table
	}
	return table
}

func (s *Scope) functionTableKey() functionTableKey {
	return functionTableKey{
		baseDir:         s.BaseDir,
		pureOnly:        s.PureOnly,
		consoleMode:     s.ConsoleMode,
		forProvider:     s.ForProvider,
		planTimestamp:   s.PlanTimestamp,
		functionResults: s.FunctionResults,
	}
}

// immutableResults is a wrapper for cty function implementations which may
// otherwise not return consistent results because they depends on data outside
// of Terraform. Due to the fact that the cty functions are a concrete type, and
// the implementation is hidden within a private struct field, we need to pass
// along these closures to get the data to the actual call site.
func immutableResults(name string, priorResults *FunctionResults) func(fn function.ImplFunc) function.ImplFunc {
	if priorResults == nil {
		return func(fn function.ImplFunc) function.ImplFunc {
			return fn
		}
	}
	return func(fn function.ImplFunc) function.ImplFunc {
		return func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			res, err := fn(args, retType)
			if err != nil {
				return res, err
			}
			err = priorResults.CheckPrior(name, args, res)
			if err != nil {
				return cty.UnknownVal(retType), err
			}
			return res, err
		}
	}
}

func filesystemNoopWrapper(name string, priorResults *FunctionResults) func(fn function.ImplFunc) function.ImplFunc {
	return noopWrapper
}

func noopWrapper(fn function.ImplFunc) function.ImplFunc {
	return fn
}
