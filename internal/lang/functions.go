// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package lang

import (
	"fmt"

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
func (s *Scope) Functions() map[string]function.Function {
	s.funcsLock.Lock()
	if s.funcs == nil {
		s.funcs = baseFunctions(s.BaseDir)

		// If you're adding something here, please consider whether it meets
		// the criteria for either or both of the sets [filesystemFunctions]
		// and [templateFunctions] and add it there if so, to ensure that
		// functions relying on those classifications will behave correctly.
		coreFuncs := map[string]function.Function{
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
			"csvdecode":        stdlib.CSVDecodeFunc,
			"dirname":          funcs.DirnameFunc,
			"distinct":         stdlib.DistinctFunc,
			"element":          stdlib.ElementFunc,
			"endswith":         funcs.EndsWithFunc,
			"ephemeralasnull":  s.experimentalFunction(experiments.EphemeralValues, funcs.EphemeralAsNullFunc),
			"chunklist":        stdlib.ChunklistFunc,
			"file":             funcs.MakeFileFunc(s.BaseDir, false),
			"fileexists":       funcs.MakeFileExistsFunc(s.BaseDir),
			"fileset":          funcs.MakeFileSetFunc(s.BaseDir),
			"filebase64":       funcs.MakeFileFunc(s.BaseDir, true),
			"filebase64sha256": funcs.MakeFileBase64Sha256Func(s.BaseDir),
			"filebase64sha512": funcs.MakeFileBase64Sha512Func(s.BaseDir),
			"filemd5":          funcs.MakeFileMd5Func(s.BaseDir),
			"filesha1":         funcs.MakeFileSha1Func(s.BaseDir),
			"filesha256":       funcs.MakeFileSha256Func(s.BaseDir),
			"filesha512":       funcs.MakeFileSha512Func(s.BaseDir),
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
			"urldecode":        funcs.URLDecodeFunc,
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
			return s.funcs, filesystemFunctions, templateFunctions
		}
		coreFuncs["templatefile"] = funcs.MakeTemplateFileFunc(s.BaseDir, funcsFunc)
		coreFuncs["templatestring"] = funcs.MakeTemplateStringFunc(funcsFunc)

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
		s.funcs = make(map[string]function.Function, len(coreFuncs)*2)
		for name, fn := range coreFuncs {
			fn = funcs.WithDescription(name, fn)
			s.funcs[name] = fn
			s.funcs["core::"+name] = fn
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
		"csvdecode":        stdlib.CSVDecodeFunc,
		"dirname":          funcs.DirnameFunc,
		"distinct":         stdlib.DistinctFunc,
		"element":          stdlib.ElementFunc,
		"endswith":         funcs.EndsWithFunc,
		"chunklist":        stdlib.ChunklistFunc,
		"file":             funcs.MakeFileFunc(baseDir, false),
		"fileexists":       funcs.MakeFileExistsFunc(baseDir),
		"fileset":          funcs.MakeFileSetFunc(baseDir),
		"filebase64":       funcs.MakeFileFunc(baseDir, true),
		"filebase64sha256": funcs.MakeFileBase64Sha256Func(baseDir),
		"filebase64sha512": funcs.MakeFileBase64Sha512Func(baseDir),
		"filemd5":          funcs.MakeFileMd5Func(baseDir),
		"filesha1":         funcs.MakeFileSha1Func(baseDir),
		"filesha256":       funcs.MakeFileSha256Func(baseDir),
		"filesha512":       funcs.MakeFileSha512Func(baseDir),
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

	fs["templatefile"] = funcs.MakeTemplateFileFunc(baseDir, func() (map[string]function.Function, collections.Set[string], collections.Set[string]) {
		// The templatefile function prevents recursive calls to itself
		// by copying this map and overwriting the "templatefile" entry.
		return fs, filesystemFunctions, templateFunctions
	})

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
