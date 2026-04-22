// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package lang

import (
	"fmt"
	"sync"

	"github.com/hashicorp/hcl/v2/ext/tryfunc"
	ctyyaml "github.com/zclconf/go-cty-yaml"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"

	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/experiments"
	"github.com/hashicorp/terraform/internal/lang/funcs"
)

// cachedCoreFuncs holds the pre-computed base function table with descriptions
// and "core::" prefixes already applied. This avoids re-creating ~180 function
// objects on every Scope.Functions() call. The cached table is immutable once
// initialized; callers must clone it before making modifications.
var (
	cachedCoreFuncsOnce sync.Once
	cachedCoreFuncs     map[string]function.Function
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

// coreFunctionsTable returns the cached, immutable base function table with
// descriptions and "core::" prefixes already applied. This is computed once
// and shared across all Scope instances. Callers must NOT modify the returned
// map; clone it first if modifications are needed.
func coreFunctionsTable() map[string]function.Function {
	cachedCoreFuncsOnce.Do(func() {
		// baseFunctions uses "." as the baseDir since that's what the
		// evaluator always passes. The filesystem functions will be
		// overridden per-scope anyway.
		base := baseFunctions(".")

		// Apply descriptions and build the core:: namespace, just like the
		// original Functions() method did on every call.
		cachedCoreFuncs = make(map[string]function.Function, len(base)*2)
		for name, fn := range base {
			fn = funcs.WithDescription(name, fn)
			cachedCoreFuncs[name] = fn
			cachedCoreFuncs["core::"+name] = fn
		}
	})
	return cachedCoreFuncs
}

// Functions returns the set of functions that should be used to when evaluating
// expressions in the receiving scope.
func (s *Scope) Functions() map[string]function.Function {
	// For backwards compatibility, filesystem functions are allowed to return
	// inconsistent results when called from within a provider configuration, so
	// here we override the checks with a noop wrapper. This misbehavior was
	// found to be used by a number of configurations, which took advantage of
	// it to create the equivalent of ephemeral values before they formally
	// existed in the language.
	immutableResults := immutableResults
	if s.ForProvider {
		immutableResults = filesystemNoopWrapper
	}

	s.funcsLock.Lock()
	if s.funcs == nil {
		// Start from the cached core functions table (immutable, shared).
		// Clone it so we can apply per-scope overrides.
		cached := coreFunctionsTable()
		s.funcs = make(map[string]function.Function, len(cached)+20)
		for k, v := range cached {
			s.funcs[k] = v
		}

		// Override filesystem functions that need scope-specific wrappers
		// for checking consistent results between plan and apply.
		overrideWithDesc := func(name string, fn function.Function) {
			fn = funcs.WithDescription(name, fn)
			s.funcs[name] = fn
			s.funcs["core::"+name] = fn
		}
		overrideWithDesc("file", funcs.MakeFileFunc(s.BaseDir, false, immutableResults("file", s.FunctionResults)))
		overrideWithDesc("fileexists", funcs.MakeFileExistsFunc(s.BaseDir, immutableResults("fileexists", s.FunctionResults)))
		overrideWithDesc("fileset", funcs.MakeFileSetFunc(s.BaseDir, immutableResults("fileset", s.FunctionResults)))
		overrideWithDesc("filebase64", funcs.MakeFileFunc(s.BaseDir, true, immutableResults("filebase64", s.FunctionResults)))
		overrideWithDesc("filebase64sha256", funcs.MakeFileBase64Sha256Func(s.BaseDir, immutableResults("filebase64sha256", s.FunctionResults)))
		overrideWithDesc("filebase64sha512", funcs.MakeFileBase64Sha512Func(s.BaseDir, immutableResults("filebase64sha512", s.FunctionResults)))
		overrideWithDesc("filemd5", funcs.MakeFileMd5Func(s.BaseDir, immutableResults("filemd5", s.FunctionResults)))
		overrideWithDesc("filesha1", funcs.MakeFileSha1Func(s.BaseDir, immutableResults("filesha1", s.FunctionResults)))
		overrideWithDesc("filesha256", funcs.MakeFileSha256Func(s.BaseDir, immutableResults("filesha256", s.FunctionResults)))
		overrideWithDesc("filesha512", funcs.MakeFileSha512Func(s.BaseDir, immutableResults("filesha512", s.FunctionResults)))

		// Our two template-rendering functions want to be able to call
		// all of the other functions themselves, but we pass them indirectly
		// via a callback to avoid chicken/egg problems while initializing
		// the functions table.
		funcsFunc := func() (funcs map[string]function.Function, fsFuncs collections.Set[string], templateFuncs collections.Set[string]) {
			// The templatefile and templatestring functions prevent recursive
			// calls to themselves and each other by copying this map and
			// overwriting the relevant entries.
			return s.funcs, filesystemFunctions, templateFunctions
		}
		overrideWithDesc("templatefile", funcs.MakeTemplateFileFunc(s.BaseDir, funcsFunc, immutableResults("templatefile", s.FunctionResults)))
		overrideWithDesc("templatestring", funcs.MakeTemplateStringFunc(funcsFunc))

		if s.ConsoleMode {
			// The type function is only available in terraform console.
			overrideWithDesc("type", funcs.TypeFunc)
		}

		if !s.ConsoleMode {
			// The plantimestamp function doesn't make sense in the terraform
			// console.
			overrideWithDesc("plantimestamp", funcs.MakeStaticTimestampFunc(s.PlanTimestamp))
		}

		if s.PureOnly {
			// Force our few impure functions to return unknown so that we
			// can defer evaluating them until a later pass.
			for _, name := range impureFunctions {
				fn := function.Unpredictable(s.funcs[name])
				s.funcs[name] = fn
				s.funcs["core::"+name] = fn
			}
		}

		// We'll also bring in any external functions that the caller provided
		// when constructing this scope. For now, that's just
		// provider-contributed functions, under a "provider::NAME::" namespace
		// where NAME is the local name of the provider in the current module.
		for providerLocalName, funcs := range s.ExternalFuncs.Provider {
			for funcName, fn := range funcs {
				name := fmt.Sprintf("provider::%s::%s", providerLocalName, funcName)
				s.funcs[name] = fn
			}
		}
	}
	s.funcsLock.Unlock()

	return s.funcs
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
		"index":            funcs.IndexFunc, // stdlib.IndexFunc is not compatible
		"join":             stdlib.JoinFunc,
		"jsondecode":       stdlib.JSONDecodeFunc,
		"jsonencode":       stdlib.JSONEncodeFunc,
		"keys":             stdlib.KeysFunc,
		"length":           funcs.LengthFunc,
		"list":             funcs.ListFunc,
		"log":              stdlib.LogFunc,
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
		"pow":              stdlib.PowFunc,
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
