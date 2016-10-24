package scheduler

import (
	"fmt"
	"log"
	"math/rand"
	"reflect"

	"github.com/hashicorp/nomad/nomad/structs"
)

// allocTuple is a tuple of the allocation name and potential alloc ID
type allocTuple struct {
	Name      string
	TaskGroup *structs.TaskGroup
	Alloc     *structs.Allocation
}

// materializeTaskGroups is used to materialize all the task groups
// a job requires. This is used to do the count expansion.
func materializeTaskGroups(job *structs.Job) map[string]*structs.TaskGroup {
	out := make(map[string]*structs.TaskGroup)
	if job == nil {
		return out
	}

	for _, tg := range job.TaskGroups {
		for i := 0; i < tg.Count; i++ {
			name := fmt.Sprintf("%s.%s[%d]", job.Name, tg.Name, i)
			out[name] = tg
		}
	}
	return out
}

// diffResult is used to return the sets that result from the diff
type diffResult struct {
	place, update, migrate, stop, ignore, lost []allocTuple
}

func (d *diffResult) GoString() string {
	return fmt.Sprintf("allocs: (place %d) (update %d) (migrate %d) (stop %d) (ignore %d) (lost %d)",
		len(d.place), len(d.update), len(d.migrate), len(d.stop), len(d.ignore), len(d.lost))
}

func (d *diffResult) Append(other *diffResult) {
	d.place = append(d.place, other.place...)
	d.update = append(d.update, other.update...)
	d.migrate = append(d.migrate, other.migrate...)
	d.stop = append(d.stop, other.stop...)
	d.ignore = append(d.ignore, other.ignore...)
	d.lost = append(d.lost, other.lost...)
}

// diffAllocs is used to do a set difference between the target allocations
// and the existing allocations. This returns 6 sets of results, the list of
// named task groups that need to be placed (no existing allocation), the
// allocations that need to be updated (job definition is newer), allocs that
// need to be migrated (node is draining), the allocs that need to be evicted
// (no longer required), those that should be ignored and those that are lost
// that need to be replaced (running on a lost node).
func diffAllocs(job *structs.Job, taintedNodes map[string]*structs.Node,
	required map[string]*structs.TaskGroup, allocs []*structs.Allocation) *diffResult {
	result := &diffResult{}

	// Scan the existing updates
	existing := make(map[string]struct{})
	for _, exist := range allocs {
		// Index the existing node
		name := exist.Name
		existing[name] = struct{}{}

		// Check for the definition in the required set
		tg, ok := required[name]

		// If not required, we stop the alloc
		if !ok {
			result.stop = append(result.stop, allocTuple{
				Name:      name,
				TaskGroup: tg,
				Alloc:     exist,
			})
			continue
		}

		// If we are on a tainted node, we must migrate if we are a service or
		// if the batch allocation did not finish
		if node, ok := taintedNodes[exist.NodeID]; ok {
			// If the job is batch and finished successfully, the fact that the
			// node is tainted does not mean it should be migrated or marked as
			// lost as the work was already successfully finished. However for
			// service/system jobs, tasks should never complete. The check of
			// batch type, defends against client bugs.
			if exist.Job.Type == structs.JobTypeBatch && exist.RanSuccessfully() {
				goto IGNORE
			}

			if node == nil || node.TerminalStatus() {
				result.lost = append(result.lost, allocTuple{
					Name:      name,
					TaskGroup: tg,
					Alloc:     exist,
				})
			} else {
				// This is the drain case
				result.migrate = append(result.migrate, allocTuple{
					Name:      name,
					TaskGroup: tg,
					Alloc:     exist,
				})
			}
			continue
		}

		// If the definition is updated we need to update
		if job.JobModifyIndex != exist.Job.JobModifyIndex {
			result.update = append(result.update, allocTuple{
				Name:      name,
				TaskGroup: tg,
				Alloc:     exist,
			})
			continue
		}

		// Everything is up-to-date
	IGNORE:
		result.ignore = append(result.ignore, allocTuple{
			Name:      name,
			TaskGroup: tg,
			Alloc:     exist,
		})
	}

	// Scan the required groups
	for name, tg := range required {
		// Check for an existing allocation
		_, ok := existing[name]

		// Require a placement if no existing allocation. If there
		// is an existing allocation, we would have checked for a potential
		// update or ignore above.
		if !ok {
			result.place = append(result.place, allocTuple{
				Name:      name,
				TaskGroup: tg,
			})
		}
	}
	return result
}

// diffSystemAllocs is like diffAllocs however, the allocations in the
// diffResult contain the specific nodeID they should be allocated on.
func diffSystemAllocs(job *structs.Job, nodes []*structs.Node, taintedNodes map[string]*structs.Node,
	allocs []*structs.Allocation) *diffResult {

	// Build a mapping of nodes to all their allocs.
	nodeAllocs := make(map[string][]*structs.Allocation, len(allocs))
	for _, alloc := range allocs {
		nallocs := append(nodeAllocs[alloc.NodeID], alloc)
		nodeAllocs[alloc.NodeID] = nallocs
	}

	for _, node := range nodes {
		if _, ok := nodeAllocs[node.ID]; !ok {
			nodeAllocs[node.ID] = nil
		}
	}

	// Create the required task groups.
	required := materializeTaskGroups(job)

	result := &diffResult{}
	for nodeID, allocs := range nodeAllocs {
		diff := diffAllocs(job, taintedNodes, required, allocs)

		// Mark the alloc as being for a specific node.
		for i := range diff.place {
			alloc := &diff.place[i]
			alloc.Alloc = &structs.Allocation{NodeID: nodeID}
		}

		// Migrate does not apply to system jobs and instead should be marked as
		// stop because if a node is tainted, the job is invalid on that node.
		diff.stop = append(diff.stop, diff.migrate...)
		diff.migrate = nil

		result.Append(diff)
	}

	return result
}

// readyNodesInDCs returns all the ready nodes in the given datacenters and a
// mapping of each data center to the count of ready nodes.
func readyNodesInDCs(state State, dcs []string) ([]*structs.Node, map[string]int, error) {
	// Index the DCs
	dcMap := make(map[string]int, len(dcs))
	for _, dc := range dcs {
		dcMap[dc] = 0
	}

	// Scan the nodes
	var out []*structs.Node
	iter, err := state.Nodes()
	if err != nil {
		return nil, nil, err
	}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}

		// Filter on datacenter and status
		node := raw.(*structs.Node)
		if node.Status != structs.NodeStatusReady {
			continue
		}
		if node.Drain {
			continue
		}
		if _, ok := dcMap[node.Datacenter]; !ok {
			continue
		}
		out = append(out, node)
		dcMap[node.Datacenter] += 1
	}
	return out, dcMap, nil
}

// retryMax is used to retry a callback until it returns success or
// a maximum number of attempts is reached. An optional reset function may be
// passed which is called after each failed iteration. If the reset function is
// set and returns true, the number of attempts is reset back to max.
func retryMax(max int, cb func() (bool, error), reset func() bool) error {
	attempts := 0
	for attempts < max {
		done, err := cb()
		if err != nil {
			return err
		}
		if done {
			return nil
		}

		// Check if we should reset the number attempts
		if reset != nil && reset() {
			attempts = 0
		} else {
			attempts += 1
		}
	}
	return &SetStatusError{
		Err:        fmt.Errorf("maximum attempts reached (%d)", max),
		EvalStatus: structs.EvalStatusFailed,
	}
}

// progressMade checks to see if the plan result made allocations or updates.
// If the result is nil, false is returned.
func progressMade(result *structs.PlanResult) bool {
	return result != nil && (len(result.NodeUpdate) != 0 ||
		len(result.NodeAllocation) != 0)
}

// taintedNodes is used to scan the allocations and then check if the
// underlying nodes are tainted, and should force a migration of the allocation.
// All the nodes returned in the map are tainted.
func taintedNodes(state State, allocs []*structs.Allocation) (map[string]*structs.Node, error) {
	out := make(map[string]*structs.Node)
	for _, alloc := range allocs {
		if _, ok := out[alloc.NodeID]; ok {
			continue
		}

		node, err := state.NodeByID(alloc.NodeID)
		if err != nil {
			return nil, err
		}

		// If the node does not exist, we should migrate
		if node == nil {
			out[alloc.NodeID] = nil
			continue
		}
		if structs.ShouldDrainNode(node.Status) || node.Drain {
			out[alloc.NodeID] = node
		}
	}
	return out, nil
}

// shuffleNodes randomizes the slice order with the Fisher-Yates algorithm
func shuffleNodes(nodes []*structs.Node) {
	n := len(nodes)
	for i := n - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		nodes[i], nodes[j] = nodes[j], nodes[i]
	}
}

// tasksUpdated does a diff between task groups to see if the
// tasks, their drivers, environment variables or config have updated.
func tasksUpdated(a, b *structs.TaskGroup) bool {
	// If the number of tasks do not match, clearly there is an update
	if len(a.Tasks) != len(b.Tasks) {
		return true
	}

	// Check each task
	for _, at := range a.Tasks {
		bt := b.LookupTask(at.Name)
		if bt == nil {
			return true
		}
		if at.Driver != bt.Driver {
			return true
		}
		if at.User != bt.User {
			return true
		}
		if !reflect.DeepEqual(at.Config, bt.Config) {
			return true
		}
		if !reflect.DeepEqual(at.Env, bt.Env) {
			return true
		}
		if !reflect.DeepEqual(at.Meta, bt.Meta) {
			return true
		}
		if !reflect.DeepEqual(at.Artifacts, bt.Artifacts) {
			return true
		}

		// Inspect the network to see if the dynamic ports are different
		if len(at.Resources.Networks) != len(bt.Resources.Networks) {
			return true
		}
		for idx := range at.Resources.Networks {
			an := at.Resources.Networks[idx]
			bn := bt.Resources.Networks[idx]

			if an.MBits != bn.MBits {
				return true
			}

			aPorts, bPorts := networkPortMap(an), networkPortMap(bn)
			if !reflect.DeepEqual(aPorts, bPorts) {
				return true
			}
		}

		// Inspect the non-network resources
		if ar, br := at.Resources, bt.Resources; ar.CPU != br.CPU {
			return true
		} else if ar.MemoryMB != br.MemoryMB {
			return true
		} else if ar.DiskMB != br.DiskMB {
			return true
		} else if ar.IOPS != br.IOPS {
			return true
		}
	}
	return false
}

// networkPortMap takes a network resource and returns a map of port labels to
// values. The value for dynamic ports is disregarded even if it is set. This
// makes this function suitable for comparing two network resources for changes.
func networkPortMap(n *structs.NetworkResource) map[string]int {
	m := make(map[string]int, len(n.DynamicPorts)+len(n.ReservedPorts))
	for _, p := range n.ReservedPorts {
		m[p.Label] = p.Value
	}
	for _, p := range n.DynamicPorts {
		m[p.Label] = -1
	}
	return m
}

// setStatus is used to update the status of the evaluation
func setStatus(logger *log.Logger, planner Planner,
	eval, nextEval, spawnedBlocked *structs.Evaluation,
	tgMetrics map[string]*structs.AllocMetric, status, desc string,
	queuedAllocs map[string]int) error {

	logger.Printf("[DEBUG] sched: %#v: setting status to %s", eval, status)
	newEval := eval.Copy()
	newEval.Status = status
	newEval.StatusDescription = desc
	newEval.FailedTGAllocs = tgMetrics
	if nextEval != nil {
		newEval.NextEval = nextEval.ID
	}
	if spawnedBlocked != nil {
		newEval.BlockedEval = spawnedBlocked.ID
	}
	if queuedAllocs != nil {
		newEval.QueuedAllocations = queuedAllocs
	}

	return planner.UpdateEval(newEval)
}

// inplaceUpdate attempts to update allocations in-place where possible. It
// returns the allocs that couldn't be done inplace and then those that could.
func inplaceUpdate(ctx Context, eval *structs.Evaluation, job *structs.Job,
	stack Stack, updates []allocTuple) (destructive, inplace []allocTuple) {

	n := len(updates)
	inplaceCount := 0
	for i := 0; i < n; i++ {
		// Get the update
		update := updates[i]

		// Check if the task drivers or config has changed, requires
		// a rolling upgrade since that cannot be done in-place.
		existing := update.Alloc.Job.LookupTaskGroup(update.TaskGroup.Name)
		if tasksUpdated(update.TaskGroup, existing) {
			continue
		}

		// Get the existing node
		node, err := ctx.State().NodeByID(update.Alloc.NodeID)
		if err != nil {
			ctx.Logger().Printf("[ERR] sched: %#v failed to get node '%s': %v",
				eval, update.Alloc.NodeID, err)
			continue
		}
		if node == nil {
			continue
		}

		// Set the existing node as the base set
		stack.SetNodes([]*structs.Node{node})

		// Stage an eviction of the current allocation. This is done so that
		// the current allocation is discounted when checking for feasability.
		// Otherwise we would be trying to fit the tasks current resources and
		// updated resources. After select is called we can remove the evict.
		ctx.Plan().AppendUpdate(update.Alloc, structs.AllocDesiredStatusStop,
			allocInPlace, "")

		// Attempt to match the task group
		option, _ := stack.Select(update.TaskGroup)

		// Pop the allocation
		ctx.Plan().PopUpdate(update.Alloc)

		// Skip if we could not do an in-place update
		if option == nil {
			continue
		}

		// Restore the network offers from the existing allocation.
		// We do not allow network resources (reserved/dynamic ports)
		// to be updated. This is guarded in taskUpdated, so we can
		// safely restore those here.
		for task, resources := range option.TaskResources {
			existing := update.Alloc.TaskResources[task]
			resources.Networks = existing.Networks
		}

		// Create a shallow copy
		newAlloc := new(structs.Allocation)
		*newAlloc = *update.Alloc

		// Update the allocation
		newAlloc.EvalID = eval.ID
		newAlloc.Job = nil       // Use the Job in the Plan
		newAlloc.Resources = nil // Computed in Plan Apply
		newAlloc.TaskResources = option.TaskResources
		newAlloc.Metrics = ctx.Metrics()
		ctx.Plan().AppendAlloc(newAlloc)

		// Remove this allocation from the slice
		updates[i], updates[n-1] = updates[n-1], updates[i]
		i--
		n--
		inplaceCount++
	}
	if len(updates) > 0 {
		ctx.Logger().Printf("[DEBUG] sched: %#v: %d in-place updates of %d", eval, inplaceCount, len(updates))
	}
	return updates[:n], updates[n:]
}

// evictAndPlace is used to mark allocations for evicts and add them to the
// placement queue. evictAndPlace modifies both the the diffResult and the
// limit. It returns true if the limit has been reached.
func evictAndPlace(ctx Context, diff *diffResult, allocs []allocTuple, desc string, limit *int) bool {
	n := len(allocs)
	for i := 0; i < n && i < *limit; i++ {
		a := allocs[i]
		ctx.Plan().AppendUpdate(a.Alloc, structs.AllocDesiredStatusStop, desc, "")
		diff.place = append(diff.place, a)
	}
	if n <= *limit {
		*limit -= n
		return false
	}
	*limit = 0
	return true
}

// markLostAndPlace is used to mark allocations as lost and add them to the
// placement queue. evictAndPlace modifies both the the diffResult and the
// limit. It returns true if the limit has been reached.
func markLostAndPlace(ctx Context, diff *diffResult, allocs []allocTuple, desc string, limit *int) bool {
	n := len(allocs)
	for i := 0; i < n && i < *limit; i++ {
		a := allocs[i]
		ctx.Plan().AppendUpdate(a.Alloc, structs.AllocDesiredStatusStop, desc, structs.AllocClientStatusLost)
		diff.place = append(diff.place, a)
	}
	if n <= *limit {
		*limit -= n
		return false
	}
	*limit = 0
	return true
}

// tgConstrainTuple is used to store the total constraints of a task group.
type tgConstrainTuple struct {
	// Holds the combined constraints of the task group and all it's sub-tasks.
	constraints []*structs.Constraint

	// The set of required drivers within the task group.
	drivers map[string]struct{}

	// The combined resources of all tasks within the task group.
	size *structs.Resources
}

// taskGroupConstraints collects the constraints, drivers and resources required by each
// sub-task to aggregate the TaskGroup totals
func taskGroupConstraints(tg *structs.TaskGroup) tgConstrainTuple {
	c := tgConstrainTuple{
		constraints: make([]*structs.Constraint, 0, len(tg.Constraints)),
		drivers:     make(map[string]struct{}),
		size:        new(structs.Resources),
	}

	c.constraints = append(c.constraints, tg.Constraints...)
	for _, task := range tg.Tasks {
		c.drivers[task.Driver] = struct{}{}
		c.constraints = append(c.constraints, task.Constraints...)
		c.size.Add(task.Resources)
	}

	return c
}

// desiredUpdates takes the diffResult as well as the set of inplace and
// destructive updates and returns a map of task groups to their set of desired
// updates.
func desiredUpdates(diff *diffResult, inplaceUpdates,
	destructiveUpdates []allocTuple) map[string]*structs.DesiredUpdates {
	desiredTgs := make(map[string]*structs.DesiredUpdates)

	for _, tuple := range diff.place {
		name := tuple.TaskGroup.Name
		des, ok := desiredTgs[name]
		if !ok {
			des = &structs.DesiredUpdates{}
			desiredTgs[name] = des
		}

		des.Place++
	}

	for _, tuple := range diff.stop {
		name := tuple.Alloc.TaskGroup
		des, ok := desiredTgs[name]
		if !ok {
			des = &structs.DesiredUpdates{}
			desiredTgs[name] = des
		}

		des.Stop++
	}

	for _, tuple := range diff.ignore {
		name := tuple.TaskGroup.Name
		des, ok := desiredTgs[name]
		if !ok {
			des = &structs.DesiredUpdates{}
			desiredTgs[name] = des
		}

		des.Ignore++
	}

	for _, tuple := range diff.migrate {
		name := tuple.TaskGroup.Name
		des, ok := desiredTgs[name]
		if !ok {
			des = &structs.DesiredUpdates{}
			desiredTgs[name] = des
		}

		des.Migrate++
	}

	for _, tuple := range inplaceUpdates {
		name := tuple.TaskGroup.Name
		des, ok := desiredTgs[name]
		if !ok {
			des = &structs.DesiredUpdates{}
			desiredTgs[name] = des
		}

		des.InPlaceUpdate++
	}

	for _, tuple := range destructiveUpdates {
		name := tuple.TaskGroup.Name
		des, ok := desiredTgs[name]
		if !ok {
			des = &structs.DesiredUpdates{}
			desiredTgs[name] = des
		}

		des.DestructiveUpdate++
	}

	return desiredTgs
}

// adjustQueuedAllocations decrements the number of allocations pending per task
// group based on the number of allocations successfully placed
func adjustQueuedAllocations(logger *log.Logger, result *structs.PlanResult, queuedAllocs map[string]int) {
	if result != nil {
		for _, allocations := range result.NodeAllocation {
			for _, allocation := range allocations {
				// Ensure that the allocation is newly created
				if allocation.CreateIndex != result.AllocIndex {
					continue
				}

				if _, ok := queuedAllocs[allocation.TaskGroup]; ok {
					queuedAllocs[allocation.TaskGroup] -= 1
				} else {
					logger.Printf("[ERR] sched: allocation %q placed but not in list of unplaced allocations", allocation.TaskGroup)
				}
			}
		}
	}
}

// updateNonTerminalAllocsToLost updates the allocations which are in pending/running state on tainted node
// to lost
func updateNonTerminalAllocsToLost(plan *structs.Plan, tainted map[string]*structs.Node, allocs []*structs.Allocation) {
	for _, alloc := range allocs {
		if _, ok := tainted[alloc.NodeID]; ok &&
			alloc.DesiredStatus == structs.AllocDesiredStatusStop &&
			(alloc.ClientStatus == structs.AllocClientStatusRunning ||
				alloc.ClientStatus == structs.AllocClientStatusPending) {
			plan.AppendUpdate(alloc, structs.AllocDesiredStatusStop, allocLost, structs.AllocClientStatusLost)
		}
	}
}
