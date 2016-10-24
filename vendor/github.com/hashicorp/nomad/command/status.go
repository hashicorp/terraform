package command

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/nomad/structs"
)

const (
	// maxFailedTGs is the maximum number of task groups we show failure reasons
	// for before defering to eval-status
	maxFailedTGs = 5
)

type StatusCommand struct {
	Meta
	length  int
	evals   bool
	verbose bool
}

func (c *StatusCommand) Help() string {
	helpText := `
Usage: nomad status [options] <job>

  Display status information about jobs. If no job ID is given,
  a list of all known jobs will be dumped.

General Options:

  ` + generalOptionsUsage() + `

Status Options:

  -short
    Display short output. Used only when a single job is being
    queried, and drops verbose information about allocations.

  -evals
    Display the evaluations associated with the job.

  -verbose
    Display full information.
`
	return strings.TrimSpace(helpText)
}

func (c *StatusCommand) Synopsis() string {
	return "Display status information about jobs"
}

func (c *StatusCommand) Run(args []string) int {
	var short bool

	flags := c.Meta.FlagSet("status", FlagSetClient)
	flags.Usage = func() { c.Ui.Output(c.Help()) }
	flags.BoolVar(&short, "short", false, "")
	flags.BoolVar(&c.evals, "evals", false, "")
	flags.BoolVar(&c.verbose, "verbose", false, "")

	if err := flags.Parse(args); err != nil {
		return 1
	}

	// Check that we either got no jobs or exactly one.
	args = flags.Args()
	if len(args) > 1 {
		c.Ui.Error(c.Help())
		return 1
	}

	// Truncate the id unless full length is requested
	c.length = shortId
	if c.verbose {
		c.length = fullId
	}

	// Get the HTTP client
	client, err := c.Meta.Client()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error initializing client: %s", err))
		return 1
	}

	// Invoke list mode if no job ID.
	if len(args) == 0 {
		jobs, _, err := client.Jobs().List(nil)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error querying jobs: %s", err))
			return 1
		}

		if len(jobs) == 0 {
			// No output if we have no jobs
			c.Ui.Output("No running jobs")
		} else {
			c.Ui.Output(createStatusListOutput(jobs))
		}
		return 0
	}

	// Try querying the job
	jobID := args[0]
	jobs, _, err := client.Jobs().PrefixList(jobID)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error querying job: %s", err))
		return 1
	}
	if len(jobs) == 0 {
		c.Ui.Error(fmt.Sprintf("No job(s) with prefix or id %q found", jobID))
		return 1
	}
	if len(jobs) > 1 && strings.TrimSpace(jobID) != jobs[0].ID {
		c.Ui.Output(fmt.Sprintf("Prefix matched multiple jobs\n\n%s", createStatusListOutput(jobs)))
		return 0
	}
	// Prefix lookup matched a single job
	job, _, err := client.Jobs().Info(jobs[0].ID, nil)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error querying job: %s", err))
		return 1
	}

	// Check if it is periodic
	sJob, err := convertApiJob(job)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error converting job: %s", err))
		return 1
	}
	periodic := sJob.IsPeriodic()

	// Format the job info
	basic := []string{
		fmt.Sprintf("ID|%s", job.ID),
		fmt.Sprintf("Name|%s", job.Name),
		fmt.Sprintf("Type|%s", job.Type),
		fmt.Sprintf("Priority|%d", job.Priority),
		fmt.Sprintf("Datacenters|%s", strings.Join(job.Datacenters, ",")),
		fmt.Sprintf("Status|%s", job.Status),
		fmt.Sprintf("Periodic|%v", periodic),
	}

	if periodic {
		now := time.Now().UTC()
		next := sJob.Periodic.Next(now)
		basic = append(basic, fmt.Sprintf("Next Periodic Launch|%s",
			fmt.Sprintf("%s (%s from now)",
				formatTime(next), formatTimeDifference(now, next, time.Second))))
	}

	c.Ui.Output(formatKV(basic))

	// Exit early
	if short {
		return 0
	}

	// Print periodic job information
	if periodic {
		if err := c.outputPeriodicInfo(client, job); err != nil {
			c.Ui.Error(err.Error())
			return 1
		}

		return 0
	}

	if err := c.outputJobInfo(client, job); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	return 0
}

// outputPeriodicInfo prints information about the passed periodic job. If a
// request fails, an error is returned.
func (c *StatusCommand) outputPeriodicInfo(client *api.Client, job *api.Job) error {
	// Generate the prefix that matches launched jobs from the periodic job.
	prefix := fmt.Sprintf("%s%s", job.ID, structs.PeriodicLaunchSuffix)
	children, _, err := client.Jobs().PrefixList(prefix)
	if err != nil {
		return fmt.Errorf("Error querying job: %s", err)
	}

	if len(children) == 0 {
		c.Ui.Output("\nNo instances of periodic job found")
		return nil
	}

	out := make([]string, 1)
	out[0] = "ID|Status"
	for _, child := range children {
		// Ensure that we are only showing jobs whose parent is the requested
		// job.
		if child.ParentID != job.ID {
			continue
		}

		out = append(out, fmt.Sprintf("%s|%s",
			child.ID,
			child.Status))
	}

	c.Ui.Output(fmt.Sprintf("\nPreviously launched jobs:\n%s", formatList(out)))
	return nil
}

// outputJobInfo prints information about the passed non-periodic job. If a
// request fails, an error is returned.
func (c *StatusCommand) outputJobInfo(client *api.Client, job *api.Job) error {
	var evals, allocs []string

	// Query the allocations
	jobAllocs, _, err := client.Jobs().Allocations(job.ID, nil)
	if err != nil {
		return fmt.Errorf("Error querying job allocations: %s", err)
	}

	// Query the evaluations
	jobEvals, _, err := client.Jobs().Evaluations(job.ID, nil)
	if err != nil {
		return fmt.Errorf("Error querying job evaluations: %s", err)
	}

	// Query the summary
	summary, _, err := client.Jobs().Summary(job.ID, nil)
	if err != nil {
		return fmt.Errorf("Error querying job summary: %s", err)
	}

	// Format the summary
	c.Ui.Output(c.Colorize().Color("\n[bold]Summary[reset]"))
	if summary != nil {
		summaries := make([]string, len(summary.Summary)+1)
		summaries[0] = "Task Group|Queued|Starting|Running|Failed|Complete|Lost"
		taskGroups := make([]string, 0, len(summary.Summary))
		for taskGroup := range summary.Summary {
			taskGroups = append(taskGroups, taskGroup)
		}
		sort.Strings(taskGroups)
		for idx, taskGroup := range taskGroups {
			tgs := summary.Summary[taskGroup]
			summaries[idx+1] = fmt.Sprintf("%s|%d|%d|%d|%d|%d|%d",
				taskGroup, tgs.Queued, tgs.Starting,
				tgs.Running, tgs.Failed,
				tgs.Complete, tgs.Lost,
			)
		}
		c.Ui.Output(formatList(summaries))
	}

	// Determine latest evaluation with failures whose follow up hasn't
	// completed, this is done while formatting
	var latestFailedPlacement *api.Evaluation
	blockedEval := false

	// Format the evals
	evals = make([]string, len(jobEvals)+1)
	evals[0] = "ID|Priority|Triggered By|Status|Placement Failures"
	for i, eval := range jobEvals {
		failures, _ := evalFailureStatus(eval)
		evals[i+1] = fmt.Sprintf("%s|%d|%s|%s|%s",
			limit(eval.ID, c.length),
			eval.Priority,
			eval.TriggeredBy,
			eval.Status,
			failures,
		)

		if eval.Status == "blocked" {
			blockedEval = true
		}

		if len(eval.FailedTGAllocs) == 0 {
			// Skip evals without failures
			continue
		}

		if latestFailedPlacement == nil || latestFailedPlacement.CreateIndex < eval.CreateIndex {
			latestFailedPlacement = eval
		}
	}

	if c.verbose || c.evals {
		c.Ui.Output(c.Colorize().Color("\n[bold]Evaluations[reset]"))
		c.Ui.Output(formatList(evals))
	}

	if blockedEval && latestFailedPlacement != nil {
		c.outputFailedPlacements(latestFailedPlacement)
	}

	// Format the allocs
	c.Ui.Output(c.Colorize().Color("\n[bold]Allocations[reset]"))
	if len(jobAllocs) > 0 {
		allocs = make([]string, len(jobAllocs)+1)
		allocs[0] = "ID|Eval ID|Node ID|Task Group|Desired|Status|Created At"
		for i, alloc := range jobAllocs {
			allocs[i+1] = fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s",
				limit(alloc.ID, c.length),
				limit(alloc.EvalID, c.length),
				limit(alloc.NodeID, c.length),
				alloc.TaskGroup,
				alloc.DesiredStatus,
				alloc.ClientStatus,
				formatUnixNanoTime(alloc.CreateTime))
		}

		c.Ui.Output(formatList(allocs))
	} else {
		c.Ui.Output("No allocations placed")
	}
	return nil
}

func (c *StatusCommand) outputFailedPlacements(failedEval *api.Evaluation) {
	if failedEval == nil || len(failedEval.FailedTGAllocs) == 0 {
		return
	}

	c.Ui.Output(c.Colorize().Color("\n[bold]Placement Failure[reset]"))

	sorted := sortedTaskGroupFromMetrics(failedEval.FailedTGAllocs)
	for i, tg := range sorted {
		if i >= maxFailedTGs {
			break
		}

		c.Ui.Output(fmt.Sprintf("Task Group %q:", tg))
		metrics := failedEval.FailedTGAllocs[tg]
		c.Ui.Output(formatAllocMetrics(metrics, false, "  "))
		if i != len(sorted)-1 {
			c.Ui.Output("")
		}
	}

	if len(sorted) > maxFailedTGs {
		trunc := fmt.Sprintf("\nPlacement failures truncated. To see remainder run:\nnomad eval-status %s", failedEval.ID)
		c.Ui.Output(trunc)
	}
}

// convertApiJob is used to take a *api.Job and convert it to an *struct.Job.
// This function is just a hammer and probably needs to be revisited.
func convertApiJob(in *api.Job) (*structs.Job, error) {
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
	var structJob *structs.Job
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(in); err != nil {
		return nil, err
	}
	if err := gob.NewDecoder(buf).Decode(&structJob); err != nil {
		return nil, err
	}
	return structJob, nil
}

// list general information about a list of jobs
func createStatusListOutput(jobs []*api.JobListStub) string {
	out := make([]string, len(jobs)+1)
	out[0] = "ID|Type|Priority|Status"
	for i, job := range jobs {
		out[i+1] = fmt.Sprintf("%s|%s|%d|%s",
			job.ID,
			job.Type,
			job.Priority,
			job.Status)
	}
	return formatList(out)
}
