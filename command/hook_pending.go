package command

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
)

type pendingEvent struct {
	// Kind of operation that started, or "" if the operation ended.
	op string
	// Human ID of the node being operated on.
	id string
}

type PendingHook struct {
	terraform.NilHook

	Colorize *colorstring.Colorize
	Ui       cli.Ui

	events chan *pendingEvent
}

func (h *PendingHook) PreApply(
	n *terraform.InstanceInfo,
	s *terraform.InstanceState,
	d *terraform.InstanceDiff) (terraform.HookAction, error) {
	op := "modifying"
	if d.Destroy {
		op = "destroying"
	} else if s.ID == "" {
		op = "creating"
	}

	h.events <- &pendingEvent{
		op: op,
		id: n.HumanId(),
	}

	return terraform.HookActionContinue, nil
}

func (h *PendingHook) PostApply(
	n *terraform.InstanceInfo,
	s *terraform.InstanceState,
	applyerr error) (terraform.HookAction, error) {
	h.events <- &pendingEvent{
		id: n.HumanId(),
	}

	return terraform.HookActionContinue, nil
}

// We sometimes get multiple PreApply with the same ID (eg, if destroying
// multiple old versions of something?). Keep reference counts.
type resource struct {
	op   string
	refs int
}

func (h *PendingHook) ShowPendingOperationsInBackground(doneCh <-chan struct{}) {
	h.events = make(chan *pendingEvent)
	pendingById := make(map[string]*resource)

	go func() {
		for {
			select {
			case <-doneCh:
				// The apply is done; nothing more to print.
				return

			case event := <-h.events:
				// Something happened! Update our internal state and restart the timer.
				if event.op == "" {
					// PostApply. Note that this can get called even if there's a no-op diff.
					if r, found := pendingById[event.id]; found {
						if r.refs == 1 {
							delete(pendingById, event.id)
						} else {
							r.refs--
						}
					}
					// ... otherwise ignore.
				} else {
					// PreApply.  Note that it's possible for this to be called more than
					// once for the same resource ID.
					if r, found := pendingById[event.id]; found {
						r.refs++
					} else {
						pendingById[event.id] = &resource{op: event.op, refs: 1}
					}
				}

			case <-time.After(10 * time.Second):
				// It's been a while. Print something.
				h.outputPending(pendingById)
			}
		}
	}()
}

func (h *PendingHook) outputPending(pendingById map[string]*resource) {
	if len(pendingById) == 0 {
		return
	}

	type opData struct {
		descriptions []string
		count        int
	}
	opToOpData := make(map[string]*opData)
	count := 0
	for id, resource := range pendingById {
		description := id
		if resource.refs > 1 {
			description = fmt.Sprintf("%s (x%d)", description, resource.refs)
		}
		if _, found := opToOpData[resource.op]; !found {
			opToOpData[resource.op] = &opData{}
		}
		opToOpData[resource.op].descriptions =
			append(opToOpData[resource.op].descriptions, description)
		opToOpData[resource.op].count += resource.refs
		count += resource.refs
	}

	var descriptions []string
	for op, opData := range opToOpData {
		// Canonicalize message ordering.
		sort.Strings(opData.descriptions)
		descriptions = append(descriptions, fmt.Sprintf(
			"%s %d (%s)",
			op, opData.count, strings.Join(opData.descriptions, ", ")))
	}
	// Canonicalize message ordering.
	sort.Strings(descriptions)

	operations := "operations"
	if count == 1 {
		operations = "operation"
	}
	h.Ui.Output(h.Colorize.Color(fmt.Sprintf(
		"[reset][bold]%d %s pending[reset_bold]: %s",
		count, operations,
		strings.Join(descriptions, "; "))))
}
