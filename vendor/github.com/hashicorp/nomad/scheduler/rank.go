package scheduler

import (
	"fmt"

	"github.com/hashicorp/nomad/nomad/structs"
)

// Rank is used to provide a score and various ranking metadata
// along with a node when iterating. This state can be modified as
// various rank methods are applied.
type RankedNode struct {
	Node          *structs.Node
	Score         float64
	TaskResources map[string]*structs.Resources

	// Allocs is used to cache the proposed allocations on the
	// node. This can be shared between iterators that require it.
	Proposed []*structs.Allocation
}

func (r *RankedNode) GoString() string {
	return fmt.Sprintf("<Node: %s Score: %0.3f>", r.Node.ID, r.Score)
}

func (r *RankedNode) ProposedAllocs(ctx Context) ([]*structs.Allocation, error) {
	if r.Proposed != nil {
		return r.Proposed, nil
	}

	p, err := ctx.ProposedAllocs(r.Node.ID)
	if err != nil {
		return nil, err
	}
	r.Proposed = p
	return p, nil
}

func (r *RankedNode) SetTaskResources(task *structs.Task,
	resource *structs.Resources) {
	if r.TaskResources == nil {
		r.TaskResources = make(map[string]*structs.Resources)
	}
	r.TaskResources[task.Name] = resource
}

// RankFeasibleIterator is used to iteratively yield nodes along
// with ranking metadata. The iterators may manage some state for
// performance optimizations.
type RankIterator interface {
	// Next yields a ranked option or nil if exhausted
	Next() *RankedNode

	// Reset is invoked when an allocation has been placed
	// to reset any stale state.
	Reset()
}

// FeasibleRankIterator is used to consume from a FeasibleIterator
// and return an unranked node with base ranking.
type FeasibleRankIterator struct {
	ctx    Context
	source FeasibleIterator
}

// NewFeasibleRankIterator is used to return a new FeasibleRankIterator
// from a FeasibleIterator source.
func NewFeasibleRankIterator(ctx Context, source FeasibleIterator) *FeasibleRankIterator {
	iter := &FeasibleRankIterator{
		ctx:    ctx,
		source: source,
	}
	return iter
}

func (iter *FeasibleRankIterator) Next() *RankedNode {
	option := iter.source.Next()
	if option == nil {
		return nil
	}
	ranked := &RankedNode{
		Node: option,
	}
	return ranked
}

func (iter *FeasibleRankIterator) Reset() {
	iter.source.Reset()
}

// StaticRankIterator is a RankIterator that returns a static set of results.
// This is largely only useful for testing.
type StaticRankIterator struct {
	ctx    Context
	nodes  []*RankedNode
	offset int
	seen   int
}

// NewStaticRankIterator returns a new static rank iterator over the given nodes
func NewStaticRankIterator(ctx Context, nodes []*RankedNode) *StaticRankIterator {
	iter := &StaticRankIterator{
		ctx:   ctx,
		nodes: nodes,
	}
	return iter
}

func (iter *StaticRankIterator) Next() *RankedNode {
	// Check if exhausted
	n := len(iter.nodes)
	if iter.offset == n || iter.seen == n {
		if iter.seen != n {
			iter.offset = 0
		} else {
			return nil
		}
	}

	// Return the next offset
	offset := iter.offset
	iter.offset += 1
	iter.seen += 1
	return iter.nodes[offset]
}

func (iter *StaticRankIterator) Reset() {
	iter.seen = 0
}

// BinPackIterator is a RankIterator that scores potential options
// based on a bin-packing algorithm.
type BinPackIterator struct {
	ctx      Context
	source   RankIterator
	evict    bool
	priority int
	tasks    []*structs.Task
}

// NewBinPackIterator returns a BinPackIterator which tries to fit tasks
// potentially evicting other tasks based on a given priority.
func NewBinPackIterator(ctx Context, source RankIterator, evict bool, priority int) *BinPackIterator {
	iter := &BinPackIterator{
		ctx:      ctx,
		source:   source,
		evict:    evict,
		priority: priority,
	}
	return iter
}

func (iter *BinPackIterator) SetPriority(p int) {
	iter.priority = p
}

func (iter *BinPackIterator) SetTasks(tasks []*structs.Task) {
	iter.tasks = tasks
}

func (iter *BinPackIterator) Next() *RankedNode {
OUTER:
	for {
		// Get the next potential option
		option := iter.source.Next()
		if option == nil {
			return nil
		}

		// Get the proposed allocations
		proposed, err := option.ProposedAllocs(iter.ctx)
		if err != nil {
			iter.ctx.Logger().Printf(
				"[ERR] sched.binpack: failed to get proposed allocations: %v",
				err)
			continue
		}

		// Index the existing network usage
		netIdx := structs.NewNetworkIndex()
		netIdx.SetNode(option.Node)
		netIdx.AddAllocs(proposed)

		// Assign the resources for each task
		total := new(structs.Resources)
		for _, task := range iter.tasks {
			taskResources := task.Resources.Copy()

			// Check if we need a network resource
			if len(taskResources.Networks) > 0 {
				ask := taskResources.Networks[0]
				offer, err := netIdx.AssignNetwork(ask)
				if offer == nil {
					iter.ctx.Metrics().ExhaustedNode(option.Node,
						fmt.Sprintf("network: %s", err))
					netIdx.Release()
					continue OUTER
				}

				// Reserve this to prevent another task from colliding
				netIdx.AddReserved(offer)

				// Update the network ask to the offer
				taskResources.Networks = []*structs.NetworkResource{offer}
			}

			// Store the task resource
			option.SetTaskResources(task, taskResources)

			// Accumulate the total resource requirement
			total.Add(taskResources)
		}

		// Add the resources we are trying to fit
		proposed = append(proposed, &structs.Allocation{Resources: total})

		// Check if these allocations fit, if they do not, simply skip this node
		fit, dim, util, _ := structs.AllocsFit(option.Node, proposed, netIdx)
		netIdx.Release()
		if !fit {
			iter.ctx.Metrics().ExhaustedNode(option.Node, dim)
			continue
		}

		// XXX: For now we completely ignore evictions. We should use that flag
		// to determine if its possible to evict other lower priority allocations
		// to make room. This explodes the search space, so it must be done
		// carefully.

		// Score the fit normally otherwise
		fitness := structs.ScoreFit(option.Node, util)
		option.Score += fitness
		iter.ctx.Metrics().ScoreNode(option.Node, "binpack", fitness)
		return option
	}
}

func (iter *BinPackIterator) Reset() {
	iter.source.Reset()
}

// JobAntiAffinityIterator is used to apply an anti-affinity to allocating
// along side other allocations from this job. This is used to help distribute
// load across the cluster.
type JobAntiAffinityIterator struct {
	ctx     Context
	source  RankIterator
	penalty float64
	jobID   string
}

// NewJobAntiAffinityIterator is used to create a JobAntiAffinityIterator that
// applies the given penalty for co-placement with allocs from this job.
func NewJobAntiAffinityIterator(ctx Context, source RankIterator, penalty float64, jobID string) *JobAntiAffinityIterator {
	iter := &JobAntiAffinityIterator{
		ctx:     ctx,
		source:  source,
		penalty: penalty,
		jobID:   jobID,
	}
	return iter
}

func (iter *JobAntiAffinityIterator) SetJob(jobID string) {
	iter.jobID = jobID
}

func (iter *JobAntiAffinityIterator) Next() *RankedNode {
	for {
		option := iter.source.Next()
		if option == nil {
			return nil
		}

		// Get the proposed allocations
		proposed, err := option.ProposedAllocs(iter.ctx)
		if err != nil {
			iter.ctx.Logger().Printf(
				"[ERR] sched.job-anti-aff: failed to get proposed allocations: %v",
				err)
			continue
		}

		// Determine the number of collisions
		collisions := 0
		for _, alloc := range proposed {
			if alloc.JobID == iter.jobID {
				collisions += 1
			}
		}

		// Apply a penalty if there are collisions
		if collisions > 0 {
			scorePenalty := -1 * float64(collisions) * iter.penalty
			option.Score += scorePenalty
			iter.ctx.Metrics().ScoreNode(option.Node, "job-anti-affinity", scorePenalty)
		}
		return option
	}
}

func (iter *JobAntiAffinityIterator) Reset() {
	iter.source.Reset()
}
