package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/tfdiags"
)

// FlagStringKV is a flag.Value implementation for parsing user variables
// from the command-line in the format of '-var key=value', where value is
// only ever a primitive.
type FlagStringKV map[string]string

func (v *FlagStringKV) String() string {
	return ""
}

func (v *FlagStringKV) Set(raw string) error {
	idx := strings.Index(raw, "=")
	if idx == -1 {
		return fmt.Errorf("No '=' value in arg: %s", raw)
	}

	if *v == nil {
		*v = make(map[string]string)
	}

	key, value := raw[0:idx], raw[idx+1:]
	(*v)[key] = value
	return nil
}

// FlagStringSlice is a flag.Value implementation for parsing targets from the
// command line, e.g. -target=aws_instance.foo -target=aws_vpc.bar
type FlagStringSlice []string

func (v *FlagStringSlice) String() string {
	return ""
}
func (v *FlagStringSlice) Set(raw string) error {
	*v = append(*v, raw)

	return nil
}

// FlagTargetSlice is a flag.Value implementation for parsing target addresses
// from the command line, such as -target=aws_instance.foo -target=aws_vpc.bar .
type FlagTargetSlice []addrs.Targetable

func (v *FlagTargetSlice) String() string {
	return ""
}

func (v *FlagTargetSlice) Set(raw string) error {
	// FIXME: This is not an ideal way to deal with this because it requires
	// us to do parsing in a context where we can't nicely return errors
	// to the user.

	var diags tfdiags.Diagnostics
	synthFilename := fmt.Sprintf("-target=%q", raw)
	traversal, syntaxDiags := hclsyntax.ParseTraversalAbs([]byte(raw), synthFilename, hcl.Pos{Line: 1, Column: 1})
	diags = diags.Append(syntaxDiags)
	if syntaxDiags.HasErrors() {
		return diags.Err()
	}

	target, targetDiags := addrs.ParseTarget(traversal)
	diags = diags.Append(targetDiags)
	if targetDiags.HasErrors() {
		return diags.Err()
	}

	*v = append(*v, target.Subject)
	return nil
}
