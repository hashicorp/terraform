package command

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/nomad/api"
)

type EvalStatusCommand struct {
	Meta
}

func (c *EvalStatusCommand) Help() string {
	helpText := `
Usage: nomad eval-status [options] <evaluation-id>

  Display information about evaluations. This command can be used to inspect the
  current status of an evaluation as well as determine the reason an evaluation
  did not place all allocations.

General Options:

  ` + generalOptionsUsage() + `

Eval Status Options:

  -monitor
    Monitor an outstanding evaluation

  -verbose
    Show full information.

  -json
    Output the evaluation in its JSON format.

  -t
    Format and display evaluation using a Go template.
`

	return strings.TrimSpace(helpText)
}

func (c *EvalStatusCommand) Synopsis() string {
	return "Display evaluation status and placement failure reasons"
}

func (c *EvalStatusCommand) Run(args []string) int {
	var monitor, verbose, json bool
	var tmpl string

	flags := c.Meta.FlagSet("eval-status", FlagSetClient)
	flags.Usage = func() { c.Ui.Output(c.Help()) }
	flags.BoolVar(&monitor, "monitor", false, "")
	flags.BoolVar(&verbose, "verbose", false, "")
	flags.BoolVar(&json, "json", false, "")
	flags.StringVar(&tmpl, "t", "", "")

	if err := flags.Parse(args); err != nil {
		return 1
	}

	// Check that we got exactly one evaluation ID
	args = flags.Args()

	// Get the HTTP client
	client, err := c.Meta.Client()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error initializing client: %s", err))
		return 1
	}

	// If args not specified but output format is specified, format and output the evaluations data list
	if len(args) == 0 {
		var format string
		if json && len(tmpl) > 0 {
			c.Ui.Error("Both -json and -t are not allowed")
			return 1
		} else if json {
			format = "json"
		} else if len(tmpl) > 0 {
			format = "template"
		}
		if len(format) > 0 {
			evals, _, err := client.Evaluations().List(nil)
			if err != nil {
				c.Ui.Error(fmt.Sprintf("Error querying evaluations: %v", err))
				return 1
			}
			// Return nothing if no evaluations found
			if len(evals) == 0 {
				return 0
			}

			f, err := DataFormat(format, tmpl)
			if err != nil {
				c.Ui.Error(fmt.Sprintf("Error getting formatter: %s", err))
				return 1
			}

			out, err := f.TransformData(evals)
			if err != nil {
				c.Ui.Error(fmt.Sprintf("Error formatting the data: %s", err))
				return 1
			}
			c.Ui.Output(out)
			return 0
		}
	}

	if len(args) != 1 {
		c.Ui.Error(c.Help())
		return 1
	}

	evalID := args[0]

	// Truncate the id unless full length is requested
	length := shortId
	if verbose {
		length = fullId
	}

	// Query the allocation info
	if len(evalID) == 1 {
		c.Ui.Error(fmt.Sprintf("Identifier must contain at least two characters."))
		return 1
	}
	if len(evalID)%2 == 1 {
		// Identifiers must be of even length, so we strip off the last byte
		// to provide a consistent user experience.
		evalID = evalID[:len(evalID)-1]
	}

	evals, _, err := client.Evaluations().PrefixList(evalID)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error querying evaluation: %v", err))
		return 1
	}
	if len(evals) == 0 {
		c.Ui.Error(fmt.Sprintf("No evaluation(s) with prefix or id %q found", evalID))
		return 1
	}

	if len(evals) > 1 {
		// Format the evals
		out := make([]string, len(evals)+1)
		out[0] = "ID|Priority|Triggered By|Status|Placement Failures"
		for i, eval := range evals {
			failures, _ := evalFailureStatus(eval)
			out[i+1] = fmt.Sprintf("%s|%d|%s|%s|%s",
				limit(eval.ID, length),
				eval.Priority,
				eval.TriggeredBy,
				eval.Status,
				failures,
			)
		}
		c.Ui.Output(fmt.Sprintf("Prefix matched multiple evaluations\n\n%s", formatList(out)))
		return 0
	}

	// If we are in monitor mode, monitor and exit
	if monitor {
		mon := newMonitor(c.Ui, client, length)
		return mon.monitor(evals[0].ID, true)
	}

	// Prefix lookup matched a single evaluation
	eval, _, err := client.Evaluations().Info(evals[0].ID, nil)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error querying evaluation: %s", err))
		return 1
	}

	// If output format is specified, format and output the data
	var format string
	if json {
		format = "json"
	} else if len(tmpl) > 0 {
		format = "template"
	}
	if len(format) > 0 {
		f, err := DataFormat(format, tmpl)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error getting formatter: %s", err))
			return 1
		}

		out, err := f.TransformData(eval)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error formatting the data: %s", err))
			return 1
		}
		c.Ui.Output(out)
		return 0
	}

	failureString, failures := evalFailureStatus(eval)
	triggerNoun, triggerSubj := getTriggerDetails(eval)
	statusDesc := eval.StatusDescription
	if statusDesc == "" {
		statusDesc = eval.Status
	}

	// Format the evaluation data
	basic := []string{
		fmt.Sprintf("ID|%s", limit(eval.ID, length)),
		fmt.Sprintf("Status|%s", eval.Status),
		fmt.Sprintf("Status Description|%s", statusDesc),
		fmt.Sprintf("Type|%s", eval.Type),
		fmt.Sprintf("TriggeredBy|%s", eval.TriggeredBy),
		fmt.Sprintf("%s|%s", triggerNoun, triggerSubj),
		fmt.Sprintf("Priority|%d", eval.Priority),
		fmt.Sprintf("Placement Failures|%s", failureString),
	}

	if verbose {
		// NextEval, PreviousEval, BlockedEval
		basic = append(basic,
			fmt.Sprintf("Previous Eval|%s", eval.PreviousEval),
			fmt.Sprintf("Next Eval|%s", eval.NextEval),
			fmt.Sprintf("Blocked Eval|%s", eval.BlockedEval))
	}
	c.Ui.Output(formatKV(basic))

	if failures {
		c.Ui.Output(c.Colorize().Color("\n[bold]Failed Placements[reset]"))
		sorted := sortedTaskGroupFromMetrics(eval.FailedTGAllocs)
		for _, tg := range sorted {
			metrics := eval.FailedTGAllocs[tg]

			noun := "allocation"
			if metrics.CoalescedFailures > 0 {
				noun += "s"
			}
			c.Ui.Output(fmt.Sprintf("Task Group %q (failed to place %d %s):", tg, metrics.CoalescedFailures+1, noun))
			c.Ui.Output(formatAllocMetrics(metrics, false, "  "))
			c.Ui.Output("")
		}

		if eval.BlockedEval != "" {
			c.Ui.Output(fmt.Sprintf("Evaluation %q waiting for additional capacity to place remainder",
				limit(eval.BlockedEval, length)))
		}
	}

	return 0
}

func sortedTaskGroupFromMetrics(groups map[string]*api.AllocationMetric) []string {
	tgs := make([]string, 0, len(groups))
	for tg, _ := range groups {
		tgs = append(tgs, tg)
	}
	sort.Strings(tgs)
	return tgs
}

func getTriggerDetails(eval *api.Evaluation) (noun, subject string) {
	switch eval.TriggeredBy {
	case "job-register", "job-deregister", "periodic-job", "rolling-update":
		return "Job ID", eval.JobID
	case "node-update":
		return "Node ID", eval.NodeID
	case "max-plan-attempts":
		return "Previous Eval", eval.PreviousEval
	default:
		return "", ""
	}
}
