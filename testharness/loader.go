package testharness

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/tfdiags"
	lua "github.com/yuin/gopher-lua"
)

// LoadSpecFile reads spec source code from the given filename and constructs
// a Spec from it.
//
// This is a wrapper around LoadSpec, and behaves the same aside from first
// opening the given filename.
func LoadSpecFile(filename string) (*Spec, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	f, err := os.Open(filename)
	if err != nil {
		diags = diags.Append(fmt.Errorf("Failed to open %s: %s", filename, err))
		return nil, diags
	}

	return LoadSpec(f, filename)
}

// LoadSpecDir reads spec source code from files named with a *.tfspec
// extension in the given directory.
//
// This is a wrapper around LoadSpec, and behaves the same aside from
// enumerating all of the files in the given directory and opening them.
func LoadSpecDir(dir string) (*Spec, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// We use the same state for all of our specs so that we can forget
	// about the multiple separate files once we've finished loading.
	L := lua.NewState(lua.Options{
		SkipOpenLibs: true, // we bring our own Terraform-oriented API
	})

	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		diags = diags.Append(fmt.Errorf("Failed to list %s: %s", dir, err))
		return nil, diags
	}

	spec := &Spec{
		scenarios: map[string]*Scenario{},
		lstate:    L,
	}

	for _, info := range entries {
		if matches, _ := filepath.Match("*.tfspec", info.Name()); !matches {
			continue
		}

		filename := filepath.Join(dir, info.Name())

		f, err := os.Open(filename)
		if err != nil {
			diags = diags.Append(fmt.Errorf("Failed to open %s: %s", filename, err))
			continue
		}

		fileSpec, fileDiags := loadSpec(f, filename, L)
		diags = diags.Append(fileDiags)
		if fileDiags.HasErrors() {
			continue
		}

		for name, scenario := range fileSpec.scenarios {
			if existing, exists := spec.scenarios[name]; exists {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate test scenario",
					Detail:   fmt.Sprintf("A scenario named %q was already defined at %s.", name, existing.DefRange.StartString()),
					Subject:  scenario.DefRange.ToHCL().Ptr(),
				})
				continue
			}

			spec.scenarios[name] = scenario
		}

		spec.testers = append(spec.testers, fileSpec.testers...)
	}

	return spec, diags
}

// LoadSpec reads spec source code from the given reader, assumed to be reading
// from the given filename, and constructs a Spec from it.
//
// If the returned diagnostics has errors, the returned spec may be nil.
//
// If the reader is not reading from a file on disk, pass an empty string as
// the filename.
func LoadSpec(r io.Reader, filename string) (*Spec, tfdiags.Diagnostics) {
	L := lua.NewState(lua.Options{
		SkipOpenLibs: true, // we bring our own Terraform-oriented API
	})
	return loadSpec(r, filename, L)
}

func loadSpec(r io.Reader, filename string, L *lua.LState) (*Spec, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	fn, err := L.Load(r, filename)
	if err != nil {
		if err, isCompile := err.(*lua.CompileError); isCompile {
			// We borrow hcl's diagnostic type here in order to return
			// a rich error with source information.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Compilation error in test spec file",
				Detail:   err.Message,
				Subject: &hcl.Range{
					Filename: filename,
					Start:    hcl.Pos{Line: err.Line, Column: 1},
					End:      hcl.Pos{Line: err.Line, Column: 1},
				},
			})
			return nil, diags
		} else {
			diags = diags.Append(err)
			return nil, diags
		}
	}

	topEnv := L.NewTable()
	L.SetFEnv(fn, topEnv)

	builderDiags := &Diagnostics{}
	scenariosB := scenariosBuilder{
		Diags: builderDiags,
	}
	topEnv.RawSet(lua.LString("scenario"), L.NewFunction(scenariosB.luaScenarioFunc))

	L.Push(fn)
	err = L.PCall(0, 0, nil)
	if err != nil {
		diags = diags.Append(err)
		return nil, diags
	}

	diags = diags.Append(builderDiags.Diags)

	return &Spec{
		scenarios: scenariosB.Scenarios,
		lstate:    L,
	}, diags
}
