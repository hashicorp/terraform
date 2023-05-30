package stackeval

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type withDiagnostics[T any] struct {
	Result      T
	Diagnostics tfdiags.Diagnostics
}

// taskSelfDependencyDiagnostics transforms a [promising.ErrSelfDependent]
// error into one or more error diagnostics suitable for returning to an
// end user, after first trying to discover user-friendly names for each
// of the promises involved using the .
func taskSelfDependencyDiagnostics(err promising.ErrSelfDependent, root namedPromiseReporter) tfdiags.Diagnostics {

	promiseNames := collectPromiseNames(root)
	distinctPromises := make(map[promising.PromiseID]struct{})
	for _, id := range err {
		distinctPromises[id] = struct{}{}
	}

	var diags tfdiags.Diagnostics
	switch len(distinctPromises) {
	case 0:
		// Should not get here; there can't be a promise cycle without any
		// promises involved in it.
		panic("promising.ErrSelfDependent without any promises")
	case 1:
		const diagSummary = "Object depends on itself"
		var promiseID promising.PromiseID
		for id := range distinctPromises {
			promiseID = id
		}
		name, ok := promiseNames[promiseID]
		if ok {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				diagSummary,
				fmt.Sprintf("The object %s depends on its own results, so there is no correct order of operations.", name),
			))
		} else {
			// This is the worst case to report, since something depended on
			// itself but we don't actually know its name. We can't really say
			// anything useful here, so we'll treat this as a bug and then
			// we can add whatever promise name was missing in order to fix
			// that bug.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				diagSummary,
				"One of the objects in your configuration refers to its own results, but Terraform was not able to detect which one. The fact that Terraform cannot name the object is a bug; please report it!",
			))
		}
	default:
		// If we have more than one promise involved then it's non-deterministic
		// which one we'll detect, since it depends on how the tasks get
		// scheduled by the Go runtime. To return a deterministic-ish result
		// anyway we'll arbitrarily descide to report whichever promise has
		// the lexically-least name as defined by Go's own less than operator
		// when applied to strings.
		selectedIdx := 0
		selectedName := promiseNames[err[0]]
		for i, id := range err {
			if selectedName == "" {
				// If we don't have a name yet then we'll take whatever we get
				selectedIdx = i
				selectedName = promiseNames[id]
				continue
			}
			candidateName := promiseNames[id]
			if candidateName != "" && candidateName < selectedName {
				selectedIdx = i
				selectedName = candidateName
			}
		}
		// Now we'll rotate the list of promise IDs so that the one we selected
		// appears first.
		ids := make([]promising.PromiseID, 0, len(err))
		ids = append(ids, err[selectedIdx:]...)
		ids = append(ids, err[:selectedIdx]...)
		var nameList strings.Builder
		for _, id := range ids {
			name := promiseNames[id]
			if name == "" {
				// We should minimize the number of unnamed promises so that
				// we can typically say at least something useful about what
				// objects are involved.
				name = "(...)"
			}
			fmt.Fprintf(&nameList, "\n  - %s", name)
		}
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Objects depend on themselves",
			fmt.Sprintf(
				"The following objects in your configuration form a circular dependency chain through their references:%s\n\nTerraform uses references to decide a suitable order for visiting objects, so objects may not refer to their own results either directly or indirectly.",
				nameList.String(),
			),
		))

	}
	return diags
}

// namedPromiseReporter is an interface implemented by the types in this
// package that perform asynchronous work using the promises model implemented
// by package promising, allowing discovery of user-friendly names for promises
// involved in a particular operation.
//
// We handle this as an out-of-band action so we can avoid the overhead of
// maintaining this metadata in the common case, and instead deal with it
// retroactively only in the rare case that there's a self-dependency problem
// that exhibits as a promise resolution error.
type namedPromiseReporter interface {
	// reportNamedPromises calls the given callback for each promise that
	// the caller is responsible for, giving a user-friendly name for
	// whatever data or action that promise was responsible for.
	//
	// reportNamedPromises should also delegate to the same method on any
	// directly-nested objects that might themselves have promises, so that
	// collectPromiseNames can walk the whole tree. This should be done only
	// in situations where the original reciever's implementation is itself
	// acting as the physical container for the child objects, and not just
	// when an object is _logically_ nested within another object.
	reportNamedPromises(func(id promising.PromiseID, name string))
}

func collectPromiseNames(r namedPromiseReporter) map[promising.PromiseID]string {
	ret := make(map[promising.PromiseID]string)
	r.reportNamedPromises(func(id promising.PromiseID, name string) {
		if id != promising.NoPromise {
			ret[id] = name
		}
	})
	return ret
}
