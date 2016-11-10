package command

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/scheduler"
	"github.com/mitchellh/colorstring"
)

const (
	jobModifyIndexHelp = `To submit the job with version verification run:

nomad run -check-index %d %s

When running the job with the check-index flag, the job will only be run if the
server side version matches the the job modify index returned. If the index has
changed, another user has modified the job and the plan's results are
potentially invalid.`
)

type PlanCommand struct {
	Meta
	JobGetter
	color *colorstring.Colorize
}

func (c *PlanCommand) Help() string {
	helpText := `
Usage: nomad plan [options] <file>

  Plan invokes a dry-run of the scheduler to determine the effects of submitting
  either a new or updated version of a job. The plan will not result in any
  changes to the cluster but gives insight into whether the job could be run
  successfully and how it would affect existing allocations.

  If the supplied path is "-", the jobfile is read from stdin. Otherwise
  it is read from the file at the supplied path or downloaded and
  read from URL specified.

  A job modify index is returned with the plan. This value can be used when
  submitting the job using "nomad run -check-index", which will check that the job
  was not modified between the plan and run command before invoking the
  scheduler. This ensures the job has not been modified since the plan.

  A structured diff between the local and remote job is displayed to
  give insight into what the scheduler will attempt to do and why.

  If the job has specified the region, the -region flag and NOMAD_REGION
  environment variable are overridden and the the job's region is used.

  Plan will return one of the following exit codes:
    * 0: No allocations created or destroyed.
    * 1: Allocations created or destroyed.
    * 255: Error determining plan results.

General Options:

  ` + generalOptionsUsage() + `

Plan Options:

  -diff
    Determines whether the diff between the remote job and planned job is shown.
    Defaults to true.

  -verbose
    Increase diff verbosity.
`
	return strings.TrimSpace(helpText)
}

func (c *PlanCommand) Synopsis() string {
	return "Dry-run a job update to determine its effects"
}

func (c *PlanCommand) Run(args []string) int {
	var diff, verbose bool

	flags := c.Meta.FlagSet("plan", FlagSetClient)
	flags.Usage = func() { c.Ui.Output(c.Help()) }
	flags.BoolVar(&diff, "diff", true, "")
	flags.BoolVar(&verbose, "verbose", false, "")

	if err := flags.Parse(args); err != nil {
		return 255
	}

	// Check that we got exactly one job
	args = flags.Args()
	if len(args) != 1 {
		c.Ui.Error(c.Help())
		return 255
	}

	path := args[0]
	// Get Job struct from Jobfile
	job, err := c.JobGetter.StructJob(args[0])
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error getting job struct: %s", err))
		return 255
	}

	// Initialize any fields that need to be.
	job.Canonicalize()

	// Check that the job is valid
	if err := job.Validate(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error validating job: %s", err))
		return 255
	}

	// Convert it to something we can use
	apiJob, err := convertStructJob(job)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error converting job: %s", err))
		return 255
	}

	// Get the HTTP client
	client, err := c.Meta.Client()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error initializing client: %s", err))
		return 255
	}

	// Force the region to be that of the job.
	if r := job.Region; r != "" {
		client.SetRegion(r)
	}

	// Submit the job
	resp, _, err := client.Jobs().Plan(apiJob, diff, nil)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error during plan: %s", err))
		return 255
	}

	// Print the diff if not disabled
	if diff {
		c.Ui.Output(fmt.Sprintf("%s\n",
			c.Colorize().Color(strings.TrimSpace(formatJobDiff(resp.Diff, verbose)))))
	}

	// Print the scheduler dry-run output
	c.Ui.Output(c.Colorize().Color("[bold]Scheduler dry-run:[reset]"))
	c.Ui.Output(c.Colorize().Color(formatDryRun(resp, job)))
	c.Ui.Output("")

	// Print the job index info
	c.Ui.Output(c.Colorize().Color(formatJobModifyIndex(resp.JobModifyIndex, path)))
	return getExitCode(resp)
}

// getExitCode returns 0:
// * 0: No allocations created or destroyed.
// * 1: Allocations created or destroyed.
func getExitCode(resp *api.JobPlanResponse) int {
	// Check for changes
	for _, d := range resp.Annotations.DesiredTGUpdates {
		if d.Stop+d.Place+d.Migrate+d.DestructiveUpdate > 0 {
			return 1
		}
	}

	return 0
}

// formatJobModifyIndex produces a help string that displays the job modify
// index and how to submit a job with it.
func formatJobModifyIndex(jobModifyIndex uint64, jobName string) string {
	help := fmt.Sprintf(jobModifyIndexHelp, jobModifyIndex, jobName)
	out := fmt.Sprintf("[reset][bold]Job Modify Index: %d[reset]\n%s", jobModifyIndex, help)
	return out
}

// formatDryRun produces a string explaining the results of the dry run.
func formatDryRun(resp *api.JobPlanResponse, job *structs.Job) string {
	var rolling *api.Evaluation
	for _, eval := range resp.CreatedEvals {
		if eval.TriggeredBy == "rolling-update" {
			rolling = eval
		}
	}

	var out string
	if len(resp.FailedTGAllocs) == 0 {
		out = "[bold][green]- All tasks successfully allocated.[reset]\n"
	} else {
		// Change the output depending on if we are a system job or not
		if job.Type == "system" {
			out = "[bold][yellow]- WARNING: Failed to place allocations on all nodes.[reset]\n"
		} else {
			out = "[bold][yellow]- WARNING: Failed to place all allocations.[reset]\n"
		}
		sorted := sortedTaskGroupFromMetrics(resp.FailedTGAllocs)
		for _, tg := range sorted {
			metrics := resp.FailedTGAllocs[tg]

			noun := "allocation"
			if metrics.CoalescedFailures > 0 {
				noun += "s"
			}
			out += fmt.Sprintf("%s[yellow]Task Group %q (failed to place %d %s):\n[reset]", strings.Repeat(" ", 2), tg, metrics.CoalescedFailures+1, noun)
			out += fmt.Sprintf("[yellow]%s[reset]\n\n", formatAllocMetrics(metrics, false, strings.Repeat(" ", 4)))
		}
		if rolling == nil {
			out = strings.TrimSuffix(out, "\n")
		}
	}

	if rolling != nil {
		out += fmt.Sprintf("[green]- Rolling update, next evaluation will be in %s.\n", rolling.Wait)
	}

	if next := resp.NextPeriodicLaunch; !next.IsZero() {
		out += fmt.Sprintf("[green]- If submitted now, next periodic launch would be at %s (%s from now).\n",
			formatTime(next), formatTimeDifference(time.Now().UTC(), next, time.Second))
	}

	out = strings.TrimSuffix(out, "\n")
	return out
}

// formatJobDiff produces an annoted diff of the the job. If verbose mode is
// set, added or deleted task groups and tasks are expanded.
func formatJobDiff(job *api.JobDiff, verbose bool) string {
	marker, _ := getDiffString(job.Type)
	out := fmt.Sprintf("%s[bold]Job: %q\n", marker, job.ID)

	// Determine the longest markers and fields so that the output can be
	// properly aligned.
	longestField, longestMarker := getLongestPrefixes(job.Fields, job.Objects)
	for _, tg := range job.TaskGroups {
		if _, l := getDiffString(tg.Type); l > longestMarker {
			longestMarker = l
		}
	}

	// Only show the job's field and object diffs if the job is edited or
	// verbose mode is set.
	if job.Type == "Edited" || verbose {
		fo := alignedFieldAndObjects(job.Fields, job.Objects, 0, longestField, longestMarker)
		out += fo
		if len(fo) > 0 {
			out += "\n"
		}
	}

	// Print the task groups
	for _, tg := range job.TaskGroups {
		_, mLength := getDiffString(tg.Type)
		kPrefix := longestMarker - mLength
		out += fmt.Sprintf("%s\n", formatTaskGroupDiff(tg, kPrefix, verbose))
	}

	return out
}

// formatTaskGroupDiff produces an annotated diff of a task group. If the
// verbose field is set, the task groups fields and objects are expanded even if
// the full object is an addition or removal. tgPrefix is the number of spaces to prefix
// the output of the task group.
func formatTaskGroupDiff(tg *api.TaskGroupDiff, tgPrefix int, verbose bool) string {
	marker, _ := getDiffString(tg.Type)
	out := fmt.Sprintf("%s%s[bold]Task Group: %q[reset]", marker, strings.Repeat(" ", tgPrefix), tg.Name)

	// Append the updates and colorize them
	if l := len(tg.Updates); l > 0 {
		order := make([]string, 0, l)
		for updateType := range tg.Updates {
			order = append(order, updateType)
		}

		sort.Strings(order)
		updates := make([]string, 0, l)
		for _, updateType := range order {
			count := tg.Updates[updateType]
			var color string
			switch updateType {
			case scheduler.UpdateTypeIgnore:
			case scheduler.UpdateTypeCreate:
				color = "[green]"
			case scheduler.UpdateTypeDestroy:
				color = "[red]"
			case scheduler.UpdateTypeMigrate:
				color = "[blue]"
			case scheduler.UpdateTypeInplaceUpdate:
				color = "[cyan]"
			case scheduler.UpdateTypeDestructiveUpdate:
				color = "[yellow]"
			}
			updates = append(updates, fmt.Sprintf("[reset]%s%d %s", color, count, updateType))
		}
		out += fmt.Sprintf(" (%s[reset])\n", strings.Join(updates, ", "))
	} else {
		out += "[reset]\n"
	}

	// Determine the longest field and markers so the output is properly
	// aligned
	longestField, longestMarker := getLongestPrefixes(tg.Fields, tg.Objects)
	for _, task := range tg.Tasks {
		if _, l := getDiffString(task.Type); l > longestMarker {
			longestMarker = l
		}
	}

	// Only show the task groups's field and object diffs if the group is edited or
	// verbose mode is set.
	subStartPrefix := tgPrefix + 2
	if tg.Type == "Edited" || verbose {
		fo := alignedFieldAndObjects(tg.Fields, tg.Objects, subStartPrefix, longestField, longestMarker)
		out += fo
		if len(fo) > 0 {
			out += "\n"
		}
	}

	// Output the tasks
	for _, task := range tg.Tasks {
		_, mLength := getDiffString(task.Type)
		prefix := longestMarker - mLength
		out += fmt.Sprintf("%s\n", formatTaskDiff(task, subStartPrefix, prefix, verbose))
	}

	return out
}

// formatTaskDiff produces an annotated diff of a task. If the verbose field is
// set, the tasks fields and objects are expanded even if the full object is an
// addition or removal. startPrefix is the number of spaces to prefix the output of
// the task and taskPrefix is the number of spaces to put between the marker and
// task name output.
func formatTaskDiff(task *api.TaskDiff, startPrefix, taskPrefix int, verbose bool) string {
	marker, _ := getDiffString(task.Type)
	out := fmt.Sprintf("%s%s%s[bold]Task: %q",
		strings.Repeat(" ", startPrefix), marker, strings.Repeat(" ", taskPrefix), task.Name)
	if len(task.Annotations) != 0 {
		out += fmt.Sprintf(" [reset](%s)", colorAnnotations(task.Annotations))
	}

	if task.Type == "None" {
		return out
	} else if (task.Type == "Deleted" || task.Type == "Added") && !verbose {
		// Exit early if the job was not edited and it isn't verbose output
		return out
	} else {
		out += "\n"
	}

	subStartPrefix := startPrefix + 2
	longestField, longestMarker := getLongestPrefixes(task.Fields, task.Objects)
	out += alignedFieldAndObjects(task.Fields, task.Objects, subStartPrefix, longestField, longestMarker)
	return out
}

// formatObjectDiff produces an annotated diff of an object. startPrefix is the
// number of spaces to prefix the output of the object and keyPrefix is the number
// of spaces to put between the marker and object name output.
func formatObjectDiff(diff *api.ObjectDiff, startPrefix, keyPrefix int) string {
	start := strings.Repeat(" ", startPrefix)
	marker, _ := getDiffString(diff.Type)
	out := fmt.Sprintf("%s%s%s%s {\n", start, marker, strings.Repeat(" ", keyPrefix), diff.Name)

	// Determine the length of the longest name and longest diff marker to
	// properly align names and values
	longestField, longestMarker := getLongestPrefixes(diff.Fields, diff.Objects)
	subStartPrefix := startPrefix + 2
	out += alignedFieldAndObjects(diff.Fields, diff.Objects, subStartPrefix, longestField, longestMarker)
	return fmt.Sprintf("%s\n%s}", out, start)
}

// formatFieldDiff produces an annotated diff of a field. startPrefix is the
// number of spaces to prefix the output of the field, keyPrefix is the number
// of spaces to put between the marker and field name output and valuePrefix is
// the number of spaces to put infront of the value for aligning values.
func formatFieldDiff(diff *api.FieldDiff, startPrefix, keyPrefix, valuePrefix int) string {
	marker, _ := getDiffString(diff.Type)
	out := fmt.Sprintf("%s%s%s%s: %s",
		strings.Repeat(" ", startPrefix),
		marker, strings.Repeat(" ", keyPrefix),
		diff.Name,
		strings.Repeat(" ", valuePrefix))

	switch diff.Type {
	case "Added":
		out += fmt.Sprintf("%q", diff.New)
	case "Deleted":
		out += fmt.Sprintf("%q", diff.Old)
	case "Edited":
		out += fmt.Sprintf("%q => %q", diff.Old, diff.New)
	default:
		out += fmt.Sprintf("%q", diff.New)
	}

	// Color the annotations where possible
	if l := len(diff.Annotations); l != 0 {
		out += fmt.Sprintf(" (%s)", colorAnnotations(diff.Annotations))
	}

	return out
}

// alignedFieldAndObjects is a helper method that prints fields and objects
// properly aligned.
func alignedFieldAndObjects(fields []*api.FieldDiff, objects []*api.ObjectDiff,
	startPrefix, longestField, longestMarker int) string {

	var out string
	numFields := len(fields)
	numObjects := len(objects)
	haveObjects := numObjects != 0
	for i, field := range fields {
		_, mLength := getDiffString(field.Type)
		kPrefix := longestMarker - mLength
		vPrefix := longestField - len(field.Name)
		out += formatFieldDiff(field, startPrefix, kPrefix, vPrefix)

		// Avoid a dangling new line
		if i+1 != numFields || haveObjects {
			out += "\n"
		}
	}

	for i, object := range objects {
		_, mLength := getDiffString(object.Type)
		kPrefix := longestMarker - mLength
		out += formatObjectDiff(object, startPrefix, kPrefix)

		// Avoid a dangling new line
		if i+1 != numObjects {
			out += "\n"
		}
	}

	return out
}

// getLongestPrefixes takes a list  of fields and objects and determines the
// longest field name and the longest marker.
func getLongestPrefixes(fields []*api.FieldDiff, objects []*api.ObjectDiff) (longestField, longestMarker int) {
	for _, field := range fields {
		if l := len(field.Name); l > longestField {
			longestField = l
		}
		if _, l := getDiffString(field.Type); l > longestMarker {
			longestMarker = l
		}
	}
	for _, obj := range objects {
		if _, l := getDiffString(obj.Type); l > longestMarker {
			longestMarker = l
		}
	}
	return longestField, longestMarker
}

// getDiffString returns a colored diff marker and the length of the string
// without color annotations.
func getDiffString(diffType string) (string, int) {
	switch diffType {
	case "Added":
		return "[green]+[reset] ", 2
	case "Deleted":
		return "[red]-[reset] ", 2
	case "Edited":
		return "[light_yellow]+/-[reset] ", 4
	default:
		return "", 0
	}
}

// colorAnnotations returns a comma concatonated list of the annotations where
// the annotations are colored where possible.
func colorAnnotations(annotations []string) string {
	l := len(annotations)
	if l == 0 {
		return ""
	}

	colored := make([]string, l)
	for i, annotation := range annotations {
		switch annotation {
		case "forces create":
			colored[i] = fmt.Sprintf("[green]%s[reset]", annotation)
		case "forces destroy":
			colored[i] = fmt.Sprintf("[red]%s[reset]", annotation)
		case "forces in-place update":
			colored[i] = fmt.Sprintf("[cyan]%s[reset]", annotation)
		case "forces create/destroy update":
			colored[i] = fmt.Sprintf("[yellow]%s[reset]", annotation)
		default:
			colored[i] = annotation
		}
	}

	return strings.Join(colored, ", ")
}
