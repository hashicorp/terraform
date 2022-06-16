package views

import (
	"bufio"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// The Checks view renders either one or all check results.
type Checks interface {
	CurrentResults(results *states.CheckResults, opts ChecksResultOptions) tfdiags.Diagnostics
	Diagnostics(diags tfdiags.Diagnostics)
}

// NewChecks returns an initialized Checks implementation for the given ViewType.
func NewChecks(vt arguments.ViewType, view *View) Checks {
	switch vt {
	case arguments.ViewHuman:
		return &ChecksHuman{view: view}
	default:
		panic(fmt.Sprintf("unsupported view type %v", vt))
	}
}

// ChecksResultOptions are some options for the methods of Checks that
// render aggregations of check results.
type ChecksResultOptions struct {
	// PreferShowAll indicates that the user requested that we show all
	// checks, rather than filtering only to non-passing checks.
	//
	// Some views always show all checks regardless of this setting,
	// so only the "true" value of this flag is actually meaningful.
	PreferShowAll bool
}

// The ChecksHuman implementation renders checks in a concise form intended
// for human consumption.
type ChecksHuman struct {
	view *View
}

func (v *ChecksHuman) CurrentResults(results *states.CheckResults, opts ChecksResultOptions) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if results == nil {
		v.view.streams.Eprintln(format.WordWrap(
			"The latest state snapshot for this workspace doesn't include any check results, probably because it was created by a different Terraform version that didn't support checks yet.\n\nCreate and apply a plan with this version of Terraform in order to evaluate any checks in your configuration and record the results for viewing with this command.",
			v.view.errorColumns(),
		))
		return diags
	}

	if results.ConfigResults.Len() == 0 {
		v.view.streams.Eprintln(format.WordWrap(
			"The latest state snapshot was created from a configuration that didn't define any checks, so there are no check results to report.\n\nYou can define checks by associating preconditions and postconditions with resources and output values in your configuration.",
			v.view.errorColumns(),
		))
		return diags
	}

	// We'll construct ourselves a view-oriented data structure of all of the
	// results first, before trying to render anything, because we'll want to
	// sort these into a consistent order before we show them.
	type uiCheckResult struct {
		// DisplayAddr is an address whose string representation we'll show
		// in the UI to identify this result.
		//
		// Ideally this is the actual address of a dynamic object, but for
		// configuration objects that ended up having zero dynamic objects
		// for any reason we'll put in here a _synthetic_ addrs.Checkable
		// that is derived from the addrs.ConfigCheckable by leaving all
		// of the instance keys set to addrs.NoKey, just to give us a nice
		// normalized result to sort and render with.
		DisplayAddr addrs.Checkable

		// Status is the status we'll indicate against this result.
		Status checks.Status

		// Messages are to be shown as a nested list under this item, if any.
		Messages []string
	}
	var uiResults []uiCheckResult

	for _, configElem := range results.ConfigResults.Elems {
		configAddr := configElem.Key

		if configElem.Value.ObjectResults.Len() == 0 {
			// For a config object that has no associated dynamic objects,
			// we'll report the configuration object itself as a single
			// item. This will always be either passing or unknown, because
			// we don't run checks at all if there are no objects to run against.
			//
			// In this case we construct a synthetic addrs.Checkable to use
			// for display, just so our logic below doesn't have to handle
			// both addrs.Checkable and addrs.ConfigCheckable addresses.

			var synthAddr addrs.Checkable
			switch addr := configAddr.(type) {
			case addrs.ConfigResource:
				synthAddr = addr.Resource.Instance(addrs.NoKey).Absolute(addr.Module.UnkeyedInstanceShim())
			case addrs.ConfigOutputValue:
				synthAddr = addr.OutputValue.Absolute(addr.Module.UnkeyedInstanceShim())
			default:
				panic(fmt.Sprintf("unsupported checkable address type %T", addr))
			}

			uiResults = append(uiResults, uiCheckResult{
				DisplayAddr: synthAddr,
				Status:      configElem.Value.Status,
			})
		} else {
			// If we _do_ have at least one dynamic object then we'll keep
			// the output relatively concise by not mentioning the config
			// object at all and only mentioning its objects.
			for _, objectElem := range configElem.Value.ObjectResults.Elems {
				objectAddr := objectElem.Key
				uiResults = append(uiResults, uiCheckResult{
					DisplayAddr: objectAddr,
					Status:      objectElem.Value.Status,
					Messages:    objectElem.Value.FailureMessages,
				})
			}
		}
	}

	sort.Slice(uiResults, func(i, j int) bool {
		addrI := uiResults[i].DisplayAddr
		addrJ := uiResults[j].DisplayAddr
		_, isOutputI := addrI.(addrs.AbsOutputValue)
		_, isOutputJ := addrJ.(addrs.AbsOutputValue)

		if isOutputI != isOutputJ {
			// We always push all of the output values to the end of our
			// sort, after all of the resource instances.
			return isOutputJ
		}

		// If we get here then we know that both addresses have the same
		// type, and can compare on that basis.
		switch addrI := addrI.(type) {
		case addrs.AbsResourceInstance:
			addrJ := addrJ.(addrs.AbsResourceInstance)
			return addrI.Less(addrJ)
		case addrs.AbsOutputValue:
			addrJ := addrJ.(addrs.AbsOutputValue)
			if !addrs.Equivalent(addrI.Module, addrJ.Module) {
				return addrI.Module.Less(addrJ.Module)
			}
			return addrI.OutputValue.Name < addrJ.OutputValue.Name
		default:
			panic(fmt.Sprintf("unsupported checkable address type %T", addrI))
		}
	})

	unknownCount := 0
	failErrorCount := 0
	renderedCount := 0
	for _, uiResult := range uiResults {
		var buf strings.Builder

		switch uiResult.Status {
		case checks.StatusPass:
			if !opts.PreferShowAll {
				continue
			}
			buf.WriteString(v.view.colorize.Color("[bold][green]✅[reset] "))
		case checks.StatusFail:
			buf.WriteString(v.view.colorize.Color("[bold][red]❌[reset] "))
			failErrorCount++
		case checks.StatusError:
			buf.WriteString(v.view.colorize.Color("[bold][red]❗[reset] "))
			failErrorCount++
		case checks.StatusUnknown:
			buf.WriteString(v.view.colorize.Color("[bold][light_gray]➖[reset] "))
			unknownCount++
		}

		buf.WriteString(fmt.Sprintf(v.view.colorize.Color("[bold]%s"), uiResult.DisplayAddr.String()))
		renderedCount++

		switch uiResult.Status {
		case checks.StatusPass:
			buf.WriteString(" passed")
		case checks.StatusFail:
			buf.WriteString(" failed")
		case checks.StatusError:
			buf.WriteString(" was invalid")
		case checks.StatusUnknown:
			buf.WriteString(" was not checked")
		}

		v.view.streams.Println(buf.String())

		for _, msg := range uiResult.Messages {
			var buf strings.Builder

			wrapped := format.WordWrap(msg, v.view.outputColumns()-4)
			sc := bufio.NewScanner(strings.NewReader(wrapped))
			for i := 0; sc.Scan(); i++ {
				if i == 0 {
					buf.WriteString("  - ")
				} else {
					buf.WriteString("\n    ")
				}
				buf.WriteString(sc.Text())
			}
			v.view.streams.Println(buf.String())
		}
	}

	if unknownCount == 0 && failErrorCount == 0 {
		if renderedCount > 0 {
			v.view.streams.Println("")
		}
		v.view.streams.Println(
			v.view.colorize.Color(
				"[bold][green]All checks passed![reset]\nThe custom conditions declared in this configuration were all met on the most recent Terraform run.",
			),
		)
	}
	if unknownCount > 0 {
		v.view.streams.Println(format.WordWrap(
			"\nTerraform did not perform all of the configured checks on the most recent run.\n\nThis is typically caused by an error which interrupted the apply step before it completed, but can also be caused by excluding checked objects from a run using the -target=... planning option.",
			v.view.outputColumns(),
		))
	}

	return diags
}

func (v *ChecksHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}
