package configupgrade

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/tfdiags"

	"github.com/hashicorp/hcl/v2"
	hcl2syntax "github.com/hashicorp/hcl/v2/hclsyntax"

	version "github.com/hashicorp/go-version"
)

type ModuleSources map[string][]byte

// LoadModule looks for Terraform configuration files in the given directory
// and loads each of them into memory as source code, in preparation for
// further analysis and conversion.
//
// At this stage the files are not parsed at all. Instead, we just read the
// raw bytes from the file so that they can be passed into a parser in a
// separate step.
//
// If the given directory or any of the files cannot be read, an error is
// returned. It is not safe to proceed with processing in that case because
// we cannot "see" all of the source code for the configuration.
func LoadModule(dir string) (ModuleSources, error) {
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	ret := make(ModuleSources)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			continue
		}
		if configs.IsIgnoredFile(name) {
			continue
		}
		ext := fileExt(name)
		if ext == "" {
			continue
		}

		fullPath := filepath.Join(dir, name)
		src, err := ioutil.ReadFile(fullPath)
		if err != nil {
			return nil, err
		}

		ret[name] = src
	}

	return ret, nil
}

// UnusedFilename finds a filename that isn't already used by a file in
// the receiving sources and returns it.
//
// The given "proposed" name is returned verbatim if it isn't already used.
// Otherwise, the function will try appending incrementing integers to the
// proposed name until an unused name is found. Callers should propose names
// that they do not expect to already be in use so that numeric suffixes are
// only used in rare cases.
//
// The proposed name must end in either ".tf" or ".tf.json" because a
// ModuleSources only has visibility into such files. This function will
// panic if given a file whose name does not end with one of these
// extensions.
//
// A ModuleSources only works on one directory at a time, so the proposed
// name must not contain any directory separator characters.
func (ms ModuleSources) UnusedFilename(proposed string) string {
	ext := fileExt(proposed)
	if ext == "" {
		panic(fmt.Errorf("method UnusedFilename used with invalid proposal %q", proposed))
	}

	if _, exists := ms[proposed]; !exists {
		return proposed
	}

	base := proposed[:len(proposed)-len(ext)]
	for i := 1; ; i++ {
		try := fmt.Sprintf("%s-%d%s", base, i, ext)
		if _, exists := ms[try]; !exists {
			return try
		}
	}
}

// MaybeAlreadyUpgraded is a heuristic to see if a given module may have
// already been upgraded by this package.
//
// The heuristic used is to look for a Terraform Core version constraint in
// any of the given sources that seems to be requiring a version greater than
// or equal to v0.12.0. If true is returned then the source range of the found
// version constraint is returned in case the caller wishes to present it to
// the user as context for a warning message. The returned range is not
// meaningful if false is returned.
func (ms ModuleSources) MaybeAlreadyUpgraded() (bool, tfdiags.SourceRange) {
	for name, src := range ms {
		f, diags := hcl2syntax.ParseConfig(src, name, hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			// If we can't parse at all then that's a reasonable signal that
			// we _haven't_ been upgraded yet, but we'll continue checking
			// other files anyway.
			continue
		}

		content, _, diags := f.Body.PartialContent(&hcl.BodySchema{
			Blocks: []hcl.BlockHeaderSchema{
				{
					Type: "terraform",
				},
			},
		})
		if diags.HasErrors() {
			// Suggests that the file has an invalid "terraform" block, such
			// as one with labels.
			continue
		}

		for _, block := range content.Blocks {
			content, _, diags := block.Body.PartialContent(&hcl.BodySchema{
				Attributes: []hcl.AttributeSchema{
					{
						Name: "required_version",
					},
				},
			})
			if diags.HasErrors() {
				continue
			}

			attr, present := content.Attributes["required_version"]
			if !present {
				continue
			}

			constraintVal, diags := attr.Expr.Value(nil)
			if diags.HasErrors() {
				continue
			}
			if constraintVal.Type() != cty.String || constraintVal.IsNull() {
				continue
			}

			constraints, err := version.NewConstraint(constraintVal.AsString())
			if err != nil {
				continue
			}

			// The go-version package doesn't actually let us see the details
			// of the parsed constraints here, so we now need a bit of an
			// abstraction inversion to decide if any of the given constraints
			// match our heuristic. However, we do at least get to benefit
			// from go-version's ability to extract multiple constraints from
			// a single string and the fact that it's already validated each
			// constraint to match its expected pattern.
		Constraints:
			for _, constraint := range constraints {
				str := strings.TrimSpace(constraint.String())
				// Want to match >, >= and ~> here.
				if !(strings.HasPrefix(str, ">") || strings.HasPrefix(str, "~>")) {
					continue
				}

				// Try to find something in this string that'll parse as a version.
				for i := 1; i < len(str); i++ {
					candidate := str[i:]
					v, err := version.NewVersion(candidate)
					if err != nil {
						continue
					}

					if v.Equal(firstVersionWithNewParser) || v.GreaterThan(firstVersionWithNewParser) {
						// This constraint appears to be preventing the old
						// parser from being used, so we suspect it was
						// already upgraded.
						return true, tfdiags.SourceRangeFromHCL(attr.Range)
					}

					// If we fall out here then we _did_ find something that
					// parses as a version, so we'll stop and move on to the
					// next constraint. (Otherwise we'll pass by 0.7.0 and find
					// 7.0, which is also a valid version.)
					continue Constraints
				}
			}
		}
	}
	return false, tfdiags.SourceRange{}
}

var firstVersionWithNewParser = version.Must(version.NewVersion("0.12.0"))

// fileExt returns the Terraform configuration extension of the given
// path, or a blank string if it is not a recognized extension.
func fileExt(path string) string {
	if strings.HasSuffix(path, ".tf") {
		return ".tf"
	} else if strings.HasSuffix(path, ".tf.json") {
		return ".tf.json"
	} else {
		return ""
	}
}
