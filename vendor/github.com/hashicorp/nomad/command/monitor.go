package command

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/mitchellh/cli"
)

const (
	// updateWait is the amount of time to wait between status
	// updates. Because the monitor is poll-based, we use this
	// delay to avoid overwhelming the API server.
	updateWait = time.Second
)

// evalState is used to store the current "state of the world"
// in the context of monitoring an evaluation.
type evalState struct {
	status string
	desc   string
	node   string
	job    string
	allocs map[string]*allocState
	wait   time.Duration
	index  uint64
}

// newEvalState creates and initializes a new monitorState
func newEvalState() *evalState {
	return &evalState{
		status: structs.EvalStatusPending,
		allocs: make(map[string]*allocState),
	}
}

// allocState is used to track the state of an allocation
type allocState struct {
	id          string
	group       string
	node        string
	desired     string
	desiredDesc string
	client      string
	clientDesc  string
	index       uint64

	// full is the allocation struct with full details. This
	// must be queried for explicitly so it is only included
	// if there is important error information inside.
	full *api.Allocation
}

// monitor wraps an evaluation monitor and holds metadata and
// state information.
type monitor struct {
	ui     cli.Ui
	client *api.Client
	state  *evalState

	// length determines the number of characters for identifiers in the ui.
	length int

	sync.Mutex
}

// newMonitor returns a new monitor. The returned monitor will
// write output information to the provided ui. The length parameter determines
// the number of characters for identifiers in the ui.
func newMonitor(ui cli.Ui, client *api.Client, length int) *monitor {
	mon := &monitor{
		ui: &cli.PrefixedUi{
			InfoPrefix:   "==> ",
			OutputPrefix: "    ",
			ErrorPrefix:  "==> ",
			Ui:           ui,
		},
		client: client,
		state:  newEvalState(),
		length: length,
	}
	return mon
}

// update is used to update our monitor with new state. It can be
// called whether the passed information is new or not, and will
// only dump update messages when state changes.
func (m *monitor) update(update *evalState) {
	m.Lock()
	defer m.Unlock()

	existing := m.state

	// Swap in the new state at the end
	defer func() {
		m.state = update
	}()

	// Check if the evaluation was triggered by a node
	if existing.node == "" && update.node != "" {
		m.ui.Output(fmt.Sprintf("Evaluation triggered by node %q",
			limit(update.node, m.length)))
	}

	// Check if the evaluation was triggered by a job
	if existing.job == "" && update.job != "" {
		m.ui.Output(fmt.Sprintf("Evaluation triggered by job %q", update.job))
	}

	// Check the allocations
	for allocID, alloc := range update.allocs {
		if existing, ok := existing.allocs[allocID]; !ok {
			switch {
			case alloc.index < update.index:
				// New alloc with create index lower than the eval
				// create index indicates modification
				m.ui.Output(fmt.Sprintf(
					"Allocation %q modified: node %q, group %q",
					limit(alloc.id, m.length), limit(alloc.node, m.length), alloc.group))

			case alloc.desired == structs.AllocDesiredStatusRun:
				// New allocation with desired status running
				m.ui.Output(fmt.Sprintf(
					"Allocation %q created: node %q, group %q",
					limit(alloc.id, m.length), limit(alloc.node, m.length), alloc.group))
			}
		} else {
			switch {
			case existing.client != alloc.client:
				description := ""
				if alloc.clientDesc != "" {
					description = fmt.Sprintf(" (%s)", alloc.clientDesc)
				}
				// Allocation status has changed
				m.ui.Output(fmt.Sprintf(
					"Allocation %q status changed: %q -> %q%s",
					limit(alloc.id, m.length), existing.client, alloc.client, description))
			}
		}
	}

	// Check if the status changed. We skip any transitions to pending status.
	if existing.status != "" &&
		update.status != structs.AllocClientStatusPending &&
		existing.status != update.status {
		m.ui.Output(fmt.Sprintf("Evaluation status changed: %q -> %q",
			existing.status, update.status))
	}
}

// monitor is used to start monitoring the given evaluation ID. It
// writes output directly to the monitor's ui, and returns the
// exit code for the command. If allowPrefix is false, monitor will only accept
// exact matching evalIDs.
//
// The return code will be 0 on successful evaluation. If there are
// problems scheduling the job (impossible constraints, resources
// exhausted, etc), then the return code will be 2. For any other
// failures (API connectivity, internal errors, etc), the return code
// will be 1.
func (m *monitor) monitor(evalID string, allowPrefix bool) int {
	// Track if we encounter a scheduling failure. This can only be
	// detected while querying allocations, so we use this bool to
	// carry that status into the return code.
	var schedFailure bool

	// The user may have specified a prefix as eval id. We need to lookup the
	// full id from the database first. Since we do this in a loop we need a
	// variable to keep track if we've already written the header message.
	var headerWritten bool

	// Add the initial pending state
	m.update(newEvalState())

	for {
		// Query the evaluation
		eval, _, err := m.client.Evaluations().Info(evalID, nil)
		if err != nil {
			if !allowPrefix {
				m.ui.Error(fmt.Sprintf("No evaluation with id %q found", evalID))
				return 1
			}
			if len(evalID) == 1 {
				m.ui.Error(fmt.Sprintf("Identifier must contain at least two characters."))
				return 1
			}
			if len(evalID)%2 == 1 {
				// Identifiers must be of even length, so we strip off the last byte
				// to provide a consistent user experience.
				evalID = evalID[:len(evalID)-1]
			}

			evals, _, err := m.client.Evaluations().PrefixList(evalID)
			if err != nil {
				m.ui.Error(fmt.Sprintf("Error reading evaluation: %s", err))
				return 1
			}
			if len(evals) == 0 {
				m.ui.Error(fmt.Sprintf("No evaluation(s) with prefix or id %q found", evalID))
				return 1
			}
			if len(evals) > 1 {
				// Format the evaluations
				out := make([]string, len(evals)+1)
				out[0] = "ID|Priority|Type|Triggered By|Status"
				for i, eval := range evals {
					out[i+1] = fmt.Sprintf("%s|%d|%s|%s|%s",
						limit(eval.ID, m.length),
						eval.Priority,
						eval.Type,
						eval.TriggeredBy,
						eval.Status)
				}
				m.ui.Output(fmt.Sprintf("Prefix matched multiple evaluations\n\n%s", formatList(out)))
				return 0
			}
			// Prefix lookup matched a single evaluation
			eval, _, err = m.client.Evaluations().Info(evals[0].ID, nil)
			if err != nil {
				m.ui.Error(fmt.Sprintf("Error reading evaluation: %s", err))
			}
		}

		if !headerWritten {
			m.ui.Info(fmt.Sprintf("Monitoring evaluation %q", limit(eval.ID, m.length)))
			headerWritten = true
		}

		// Create the new eval state.
		state := newEvalState()
		state.status = eval.Status
		state.desc = eval.StatusDescription
		state.node = eval.NodeID
		state.job = eval.JobID
		state.wait = eval.Wait
		state.index = eval.CreateIndex

		// Query the allocations associated with the evaluation
		allocs, _, err := m.client.Evaluations().Allocations(eval.ID, nil)
		if err != nil {
			m.ui.Error(fmt.Sprintf("Error reading allocations: %s", err))
			return 1
		}

		// Add the allocs to the state
		for _, alloc := range allocs {
			state.allocs[alloc.ID] = &allocState{
				id:          alloc.ID,
				group:       alloc.TaskGroup,
				node:        alloc.NodeID,
				desired:     alloc.DesiredStatus,
				desiredDesc: alloc.DesiredDescription,
				client:      alloc.ClientStatus,
				clientDesc:  alloc.ClientDescription,
				index:       alloc.CreateIndex,
			}
		}

		// Update the state
		m.update(state)

		switch eval.Status {
		case structs.EvalStatusComplete, structs.EvalStatusFailed, structs.EvalStatusCancelled:
			if len(eval.FailedTGAllocs) == 0 {
				m.ui.Info(fmt.Sprintf("Evaluation %q finished with status %q",
					limit(eval.ID, m.length), eval.Status))
			} else {
				// There were failures making the allocations
				schedFailure = true
				m.ui.Info(fmt.Sprintf("Evaluation %q finished with status %q but failed to place all allocations:",
					limit(eval.ID, m.length), eval.Status))

				// Print the failures per task group
				for tg, metrics := range eval.FailedTGAllocs {
					noun := "allocation"
					if metrics.CoalescedFailures > 0 {
						noun += "s"
					}
					m.ui.Output(fmt.Sprintf("Task Group %q (failed to place %d %s):", tg, metrics.CoalescedFailures+1, noun))
					metrics := formatAllocMetrics(metrics, false, "  ")
					for _, line := range strings.Split(metrics, "\n") {
						m.ui.Output(line)
					}
				}

				if eval.BlockedEval != "" {
					m.ui.Output(fmt.Sprintf("Evaluation %q waiting for additional capacity to place remainder",
						limit(eval.BlockedEval, m.length)))
				}
			}
		default:
			// Wait for the next update
			time.Sleep(updateWait)
			continue
		}

		// Monitor the next eval in the chain, if present
		if eval.NextEval != "" {
			if eval.Wait.Nanoseconds() != 0 {
				m.ui.Info(fmt.Sprintf(
					"Monitoring next evaluation %q in %s",
					limit(eval.NextEval, m.length), eval.Wait))

				// Skip some unnecessary polling
				time.Sleep(eval.Wait)
			}

			// Reset the state and monitor the new eval
			m.state = newEvalState()
			return m.monitor(eval.NextEval, allowPrefix)
		}
		break
	}

	// Treat scheduling failures specially using a dedicated exit code.
	// This makes it easier to detect failures from the CLI.
	if schedFailure {
		return 2
	}

	return 0
}

// dumpAllocStatus is a helper to generate a more user-friendly error message
// for scheduling failures, displaying a high level status of why the job
// could not be scheduled out.
func dumpAllocStatus(ui cli.Ui, alloc *api.Allocation, length int) {
	// Print filter stats
	ui.Output(fmt.Sprintf("Allocation %q status %q (%d/%d nodes filtered)",
		limit(alloc.ID, length), alloc.ClientStatus,
		alloc.Metrics.NodesFiltered, alloc.Metrics.NodesEvaluated))
	ui.Output(formatAllocMetrics(alloc.Metrics, true, "  "))
}

func formatAllocMetrics(metrics *api.AllocationMetric, scores bool, prefix string) string {
	// Print a helpful message if we have an eligibility problem
	var out string
	if metrics.NodesEvaluated == 0 {
		out += fmt.Sprintf("%s* No nodes were eligible for evaluation\n", prefix)
	}

	// Print a helpful message if the user has asked for a DC that has no
	// available nodes.
	for dc, available := range metrics.NodesAvailable {
		if available == 0 {
			out += fmt.Sprintf("%s* No nodes are available in datacenter %q\n", prefix, dc)
		}
	}

	// Print filter info
	for class, num := range metrics.ClassFiltered {
		out += fmt.Sprintf("%s* Class %q filtered %d nodes\n", prefix, class, num)
	}
	for cs, num := range metrics.ConstraintFiltered {
		out += fmt.Sprintf("%s* Constraint %q filtered %d nodes\n", prefix, cs, num)
	}

	// Print exhaustion info
	if ne := metrics.NodesExhausted; ne > 0 {
		out += fmt.Sprintf("%s* Resources exhausted on %d nodes\n", prefix, ne)
	}
	for class, num := range metrics.ClassExhausted {
		out += fmt.Sprintf("%s* Class %q exhausted on %d nodes\n", prefix, class, num)
	}
	for dim, num := range metrics.DimensionExhausted {
		out += fmt.Sprintf("%s* Dimension %q exhausted on %d nodes\n", prefix, dim, num)
	}

	// Print scores
	if scores {
		for name, score := range metrics.Scores {
			out += fmt.Sprintf("%s* Score %q = %f\n", prefix, name, score)
		}
	}

	out = strings.TrimSuffix(out, "\n")
	return out
}
