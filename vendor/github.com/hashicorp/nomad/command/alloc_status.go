package command

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/mitchellh/colorstring"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/client"
)

type AllocStatusCommand struct {
	Meta
	color *colorstring.Colorize
}

func (c *AllocStatusCommand) Help() string {
	helpText := `
Usage: nomad alloc-status [options] <allocation>

  Display information about existing allocations and its tasks. This command can
  be used to inspect the current status of all allocation, including its running
  status, metadata, and verbose failure messages reported by internal
  subsystems.

General Options:

  ` + generalOptionsUsage() + `

Alloc Status Options:

  -short
    Display short output. Shows only the most recent task event.

  -stats
    Display detailed resource usage statistics.

  -verbose
    Show full information.

  -json
    Output the allocation in its JSON format.

  -t
    Format and display allocation using a Go template.
`

	return strings.TrimSpace(helpText)
}

func (c *AllocStatusCommand) Synopsis() string {
	return "Display allocation status information and metadata"
}

func (c *AllocStatusCommand) Run(args []string) int {
	var short, displayStats, verbose, json bool
	var tmpl string

	flags := c.Meta.FlagSet("alloc-status", FlagSetClient)
	flags.Usage = func() { c.Ui.Output(c.Help()) }
	flags.BoolVar(&short, "short", false, "")
	flags.BoolVar(&verbose, "verbose", false, "")
	flags.BoolVar(&displayStats, "stats", false, "")
	flags.BoolVar(&json, "json", false, "")
	flags.StringVar(&tmpl, "t", "", "")

	if err := flags.Parse(args); err != nil {
		return 1
	}

	// Check that we got exactly one allocation ID
	args = flags.Args()

	// Get the HTTP client
	client, err := c.Meta.Client()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error initializing client: %s", err))
		return 1
	}

	// If args not specified but output format is specified, format and output the allocations data list
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
			allocs, _, err := client.Allocations().List(nil)
			if err != nil {
				c.Ui.Error(fmt.Sprintf("Error querying allocations: %v", err))
				return 1
			}
			// Return nothing if no allocations found
			if len(allocs) == 0 {
				return 0
			}

			f, err := DataFormat(format, tmpl)
			if err != nil {
				c.Ui.Error(fmt.Sprintf("Error getting formatter: %s", err))
				return 1
			}

			out, err := f.TransformData(allocs)
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
	allocID := args[0]

	// Truncate the id unless full length is requested
	length := shortId
	if verbose {
		length = fullId
	}

	// Query the allocation info
	if len(allocID) == 1 {
		c.Ui.Error(fmt.Sprintf("Identifier must contain at least two characters."))
		return 1
	}
	if len(allocID)%2 == 1 {
		// Identifiers must be of even length, so we strip off the last byte
		// to provide a consistent user experience.
		allocID = allocID[:len(allocID)-1]
	}

	allocs, _, err := client.Allocations().PrefixList(allocID)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error querying allocation: %v", err))
		return 1
	}
	if len(allocs) == 0 {
		c.Ui.Error(fmt.Sprintf("No allocation(s) with prefix or id %q found", allocID))
		return 1
	}
	if len(allocs) > 1 {
		// Format the allocs
		out := make([]string, len(allocs)+1)
		out[0] = "ID|Eval ID|Job ID|Task Group|Desired Status|Client Status"
		for i, alloc := range allocs {
			out[i+1] = fmt.Sprintf("%s|%s|%s|%s|%s|%s",
				limit(alloc.ID, length),
				limit(alloc.EvalID, length),
				alloc.JobID,
				alloc.TaskGroup,
				alloc.DesiredStatus,
				alloc.ClientStatus,
			)
		}
		c.Ui.Output(fmt.Sprintf("Prefix matched multiple allocations\n\n%s", formatList(out)))
		return 0
	}
	// Prefix lookup matched a single allocation
	alloc, _, err := client.Allocations().Info(allocs[0].ID, nil)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error querying allocation: %s", err))
		return 1
	}

	// If output format is specified, format and output the data
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
		f, err := DataFormat(format, tmpl)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error getting formatter: %s", err))
			return 1
		}

		out, err := f.TransformData(alloc)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error formatting the data: %s", err))
			return 1
		}
		c.Ui.Output(out)
		return 0
	}

	var statsErr error
	var stats *api.AllocResourceUsage
	stats, statsErr = client.Allocations().Stats(alloc, nil)
	if statsErr != nil {
		c.Ui.Output("")
		c.Ui.Error(fmt.Sprintf("couldn't retrieve stats (HINT: ensure Client.Advertise.HTTP is set): %v", statsErr))
	}

	// Format the allocation data
	basic := []string{
		fmt.Sprintf("ID|%s", limit(alloc.ID, length)),
		fmt.Sprintf("Eval ID|%s", limit(alloc.EvalID, length)),
		fmt.Sprintf("Name|%s", alloc.Name),
		fmt.Sprintf("Node ID|%s", limit(alloc.NodeID, length)),
		fmt.Sprintf("Job ID|%s", alloc.JobID),
		fmt.Sprintf("Client Status|%s", alloc.ClientStatus),
	}

	if verbose {
		basic = append(basic,
			fmt.Sprintf("Evaluated Nodes|%d", alloc.Metrics.NodesEvaluated),
			fmt.Sprintf("Filtered Nodes|%d", alloc.Metrics.NodesFiltered),
			fmt.Sprintf("Exhausted Nodes|%d", alloc.Metrics.NodesExhausted),
			fmt.Sprintf("Allocation Time|%s", alloc.Metrics.AllocationTime),
			fmt.Sprintf("Failures|%d", alloc.Metrics.CoalescedFailures))
	}
	c.Ui.Output(formatKV(basic))

	if short {
		c.shortTaskStatus(alloc)
	} else {
		c.outputTaskDetails(alloc, stats, displayStats)
	}

	// Format the detailed status
	if verbose {
		c.Ui.Output(c.Colorize().Color("\n[bold]Placement Metrics[reset]"))
		c.Ui.Output(formatAllocMetrics(alloc.Metrics, true, "  "))
	}

	return 0
}

// outputTaskDetails prints task details for each task in the allocation,
// optionally printing verbose statistics if displayStats is set
func (c *AllocStatusCommand) outputTaskDetails(alloc *api.Allocation, stats *api.AllocResourceUsage, displayStats bool) {
	for task := range c.sortedTaskStateIterator(alloc.TaskStates) {
		state := alloc.TaskStates[task]
		c.Ui.Output(c.Colorize().Color(fmt.Sprintf("\n[bold]Task %q is %q[reset]", task, state.State)))
		c.outputTaskResources(alloc, task, stats, displayStats)
		c.Ui.Output("")
		c.outputTaskStatus(state)
	}
}

// outputTaskStatus prints out a list of the most recent events for the given
// task state.
func (c *AllocStatusCommand) outputTaskStatus(state *api.TaskState) {
	c.Ui.Output("Recent Events:")
	events := make([]string, len(state.Events)+1)
	events[0] = "Time|Type|Description"

	size := len(state.Events)
	for i, event := range state.Events {
		formatedTime := formatUnixNanoTime(event.Time)

		// Build up the description based on the event type.
		var desc string
		switch event.Type {
		case api.TaskStarted:
			desc = "Task started by client"
		case api.TaskReceived:
			desc = "Task received by client"
		case api.TaskFailedValidation:
			if event.ValidationError != "" {
				desc = event.ValidationError
			} else {
				desc = "Validation of task failed"
			}
		case api.TaskDriverFailure:
			if event.DriverError != "" {
				desc = event.DriverError
			} else {
				desc = "Failed to start task"
			}
		case api.TaskDownloadingArtifacts:
			desc = "Client is downloading artifacts"
		case api.TaskArtifactDownloadFailed:
			if event.DownloadError != "" {
				desc = event.DownloadError
			} else {
				desc = "Failed to download artifacts"
			}
		case api.TaskKilling:
			if event.KillTimeout != 0 {
				desc = fmt.Sprintf("Sent interupt. Waiting %v before force killing", event.KillTimeout)
			} else {
				desc = "Sent interupt"
			}
		case api.TaskKilled:
			if event.KillError != "" {
				desc = event.KillError
			} else {
				desc = "Task successfully killed"
			}
		case api.TaskTerminated:
			var parts []string
			parts = append(parts, fmt.Sprintf("Exit Code: %d", event.ExitCode))

			if event.Signal != 0 {
				parts = append(parts, fmt.Sprintf("Signal: %d", event.Signal))
			}

			if event.Message != "" {
				parts = append(parts, fmt.Sprintf("Exit Message: %q", event.Message))
			}
			desc = strings.Join(parts, ", ")
		case api.TaskRestarting:
			in := fmt.Sprintf("Task restarting in %v", time.Duration(event.StartDelay))
			if event.RestartReason != "" && event.RestartReason != client.ReasonWithinPolicy {
				desc = fmt.Sprintf("%s - %s", event.RestartReason, in)
			} else {
				desc = in
			}
		case api.TaskNotRestarting:
			if event.RestartReason != "" {
				desc = event.RestartReason
			} else {
				desc = "Task exceeded restart policy"
			}
		}

		// Reverse order so we are sorted by time
		events[size-i] = fmt.Sprintf("%s|%s|%s", formatedTime, event.Type, desc)
	}
	c.Ui.Output(formatList(events))
}

// outputTaskResources prints the task resources for the passed task and if
// displayStats is set, verbose resource usage statistics
func (c *AllocStatusCommand) outputTaskResources(alloc *api.Allocation, task string, stats *api.AllocResourceUsage, displayStats bool) {
	resource, ok := alloc.TaskResources[task]
	if !ok {
		return
	}

	c.Ui.Output("Task Resources")
	var addr []string
	for _, nw := range resource.Networks {
		ports := append(nw.DynamicPorts, nw.ReservedPorts...)
		for _, port := range ports {
			addr = append(addr, fmt.Sprintf("%v: %v:%v\n", port.Label, nw.IP, port.Value))
		}
	}
	var resourcesOutput []string
	resourcesOutput = append(resourcesOutput, "CPU|Memory|Disk|IOPS|Addresses")
	firstAddr := ""
	if len(addr) > 0 {
		firstAddr = addr[0]
	}

	// Display the rolled up stats. If possible prefer the live stastics
	cpuUsage := strconv.Itoa(resource.CPU)
	memUsage := humanize.IBytes(uint64(resource.MemoryMB * bytesPerMegabyte))
	if ru, ok := stats.Tasks[task]; ok && ru != nil && ru.ResourceUsage != nil {
		if cs := ru.ResourceUsage.CpuStats; cs != nil {
			cpuUsage = fmt.Sprintf("%v/%v", math.Floor(cs.TotalTicks), resource.CPU)
		}
		if ms := ru.ResourceUsage.MemoryStats; ms != nil {
			memUsage = fmt.Sprintf("%v/%v", humanize.IBytes(ms.RSS), memUsage)
		}
	}
	resourcesOutput = append(resourcesOutput, fmt.Sprintf("%v MHz|%v|%v|%v|%v",
		cpuUsage,
		memUsage,
		humanize.IBytes(uint64(resource.DiskMB*bytesPerMegabyte)),
		resource.IOPS,
		firstAddr))
	for i := 1; i < len(addr); i++ {
		resourcesOutput = append(resourcesOutput, fmt.Sprintf("||||%v", addr[i]))
	}
	c.Ui.Output(formatListWithSpaces(resourcesOutput))

	if ru, ok := stats.Tasks[task]; ok && ru != nil && displayStats && ru.ResourceUsage != nil {
		c.Ui.Output("")
		c.outputVerboseResourceUsage(task, ru.ResourceUsage)
	}
}

// outputVerboseResourceUsage outputs the verbose resource usage for the passed
// task
func (c *AllocStatusCommand) outputVerboseResourceUsage(task string, resourceUsage *api.ResourceUsage) {
	memoryStats := resourceUsage.MemoryStats
	cpuStats := resourceUsage.CpuStats
	if memoryStats != nil && len(memoryStats.Measured) > 0 {
		c.Ui.Output("Memory Stats")

		// Sort the measured stats
		sort.Strings(memoryStats.Measured)

		var measuredStats []string
		for _, measured := range memoryStats.Measured {
			switch measured {
			case "RSS":
				measuredStats = append(measuredStats, humanize.IBytes(memoryStats.RSS))
			case "Cache":
				measuredStats = append(measuredStats, humanize.IBytes(memoryStats.Cache))
			case "Swap":
				measuredStats = append(measuredStats, humanize.IBytes(memoryStats.Swap))
			case "Max Usage":
				measuredStats = append(measuredStats, humanize.IBytes(memoryStats.MaxUsage))
			case "Kernel Usage":
				measuredStats = append(measuredStats, humanize.IBytes(memoryStats.KernelUsage))
			case "Kernel Max Usage":
				measuredStats = append(measuredStats, humanize.IBytes(memoryStats.KernelMaxUsage))
			}
		}

		out := make([]string, 2)
		out[0] = strings.Join(memoryStats.Measured, "|")
		out[1] = strings.Join(measuredStats, "|")
		c.Ui.Output(formatList(out))
		c.Ui.Output("")
	}

	if cpuStats != nil && len(cpuStats.Measured) > 0 {
		c.Ui.Output("CPU Stats")

		// Sort the measured stats
		sort.Strings(cpuStats.Measured)

		var measuredStats []string
		for _, measured := range cpuStats.Measured {
			switch measured {
			case "Percent":
				percent := strconv.FormatFloat(cpuStats.Percent, 'f', 2, 64)
				measuredStats = append(measuredStats, fmt.Sprintf("%v%%", percent))
			case "Throttled Periods":
				measuredStats = append(measuredStats, fmt.Sprintf("%v", cpuStats.ThrottledPeriods))
			case "Throttled Time":
				measuredStats = append(measuredStats, fmt.Sprintf("%v", cpuStats.ThrottledTime))
			case "User Mode":
				percent := strconv.FormatFloat(cpuStats.UserMode, 'f', 2, 64)
				measuredStats = append(measuredStats, fmt.Sprintf("%v%%", percent))
			case "System Mode":
				percent := strconv.FormatFloat(cpuStats.SystemMode, 'f', 2, 64)
				measuredStats = append(measuredStats, fmt.Sprintf("%v%%", percent))
			}
		}

		out := make([]string, 2)
		out[0] = strings.Join(cpuStats.Measured, "|")
		out[1] = strings.Join(measuredStats, "|")
		c.Ui.Output(formatList(out))
	}
}

// shortTaskStatus prints out the current state of each task.
func (c *AllocStatusCommand) shortTaskStatus(alloc *api.Allocation) {
	tasks := make([]string, 0, len(alloc.TaskStates)+1)
	tasks = append(tasks, "Name|State|Last Event|Time")
	for task := range c.sortedTaskStateIterator(alloc.TaskStates) {
		state := alloc.TaskStates[task]
		lastState := state.State
		var lastEvent, lastTime string

		l := len(state.Events)
		if l != 0 {
			last := state.Events[l-1]
			lastEvent = last.Type
			lastTime = formatUnixNanoTime(last.Time)
		}

		tasks = append(tasks, fmt.Sprintf("%s|%s|%s|%s",
			task, lastState, lastEvent, lastTime))
	}

	c.Ui.Output(c.Colorize().Color("\n[bold]Tasks[reset]"))
	c.Ui.Output(formatList(tasks))
}

// sortedTaskStateIterator is a helper that takes the task state map and returns a
// channel that returns the keys in a sorted order.
func (c *AllocStatusCommand) sortedTaskStateIterator(m map[string]*api.TaskState) <-chan string {
	output := make(chan string, len(m))
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)

	for _, key := range keys {
		output <- key
	}

	close(output)
	return output
}
