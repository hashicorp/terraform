package scheduler

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/nomad/nomad/structs"
)

// FeasibleIterator is used to iteratively yield nodes that
// match feasibility constraints. The iterators may manage
// some state for performance optimizations.
type FeasibleIterator interface {
	// Next yields a feasible node or nil if exhausted
	Next() *structs.Node

	// Reset is invoked when an allocation has been placed
	// to reset any stale state.
	Reset()
}

// FeasibilityChecker is used to check if a single node meets feasibility
// constraints.
type FeasibilityChecker interface {
	Feasible(*structs.Node) bool
}

// StaticIterator is a FeasibleIterator which returns nodes
// in a static order. This is used at the base of the iterator
// chain only for testing due to deterministic behavior.
type StaticIterator struct {
	ctx    Context
	nodes  []*structs.Node
	offset int
	seen   int
}

// NewStaticIterator constructs a random iterator from a list of nodes
func NewStaticIterator(ctx Context, nodes []*structs.Node) *StaticIterator {
	iter := &StaticIterator{
		ctx:   ctx,
		nodes: nodes,
	}
	return iter
}

func (iter *StaticIterator) Next() *structs.Node {
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
	iter.ctx.Metrics().EvaluateNode()
	return iter.nodes[offset]
}

func (iter *StaticIterator) Reset() {
	iter.seen = 0
}

func (iter *StaticIterator) SetNodes(nodes []*structs.Node) {
	iter.nodes = nodes
	iter.offset = 0
	iter.seen = 0
}

// NewRandomIterator constructs a static iterator from a list of nodes
// after applying the Fisher-Yates algorithm for a random shuffle. This
// is applied in-place
func NewRandomIterator(ctx Context, nodes []*structs.Node) *StaticIterator {
	// shuffle with the Fisher-Yates algorithm
	shuffleNodes(nodes)

	// Create a static iterator
	return NewStaticIterator(ctx, nodes)
}

// DriverChecker is a FeasibilityChecker which returns whether a node has the
// drivers necessary to scheduler a task group.
type DriverChecker struct {
	ctx     Context
	drivers map[string]struct{}
}

// NewDriverChecker creates a DriverChecker from a set of drivers
func NewDriverChecker(ctx Context, drivers map[string]struct{}) *DriverChecker {
	return &DriverChecker{
		ctx:     ctx,
		drivers: drivers,
	}
}

func (c *DriverChecker) SetDrivers(d map[string]struct{}) {
	c.drivers = d
}

func (c *DriverChecker) Feasible(option *structs.Node) bool {
	// Use this node if possible
	if c.hasDrivers(option) {
		return true
	}
	c.ctx.Metrics().FilterNode(option, "missing drivers")
	return false
}

// hasDrivers is used to check if the node has all the appropriate
// drivers for this task group. Drivers are registered as node attribute
// like "driver.docker=1" with their corresponding version.
func (c *DriverChecker) hasDrivers(option *structs.Node) bool {
	for driver := range c.drivers {
		driverStr := fmt.Sprintf("driver.%s", driver)
		value, ok := option.Attributes[driverStr]
		if !ok {
			return false
		}

		enabled, err := strconv.ParseBool(value)
		if err != nil {
			c.ctx.Logger().
				Printf("[WARN] scheduler.DriverChecker: node %v has invalid driver setting %v: %v",
					option.ID, driverStr, value)
			return false
		}

		if !enabled {
			return false
		}
	}
	return true
}

// ProposedAllocConstraintIterator is a FeasibleIterator which returns nodes that
// match constraints that are not static such as Node attributes but are
// effected by proposed alloc placements. Examples are distinct_hosts and
// tenancy constraints. This is used to filter on job and task group
// constraints.
type ProposedAllocConstraintIterator struct {
	ctx    Context
	source FeasibleIterator
	tg     *structs.TaskGroup
	job    *structs.Job

	// Store whether the Job or TaskGroup has a distinct_hosts constraints so
	// they don't have to be calculated every time Next() is called.
	tgDistinctHosts  bool
	jobDistinctHosts bool
}

// NewProposedAllocConstraintIterator creates a ProposedAllocConstraintIterator
// from a source.
func NewProposedAllocConstraintIterator(ctx Context, source FeasibleIterator) *ProposedAllocConstraintIterator {
	return &ProposedAllocConstraintIterator{
		ctx:    ctx,
		source: source,
	}
}

func (iter *ProposedAllocConstraintIterator) SetTaskGroup(tg *structs.TaskGroup) {
	iter.tg = tg
	iter.tgDistinctHosts = iter.hasDistinctHostsConstraint(tg.Constraints)
}

func (iter *ProposedAllocConstraintIterator) SetJob(job *structs.Job) {
	iter.job = job
	iter.jobDistinctHosts = iter.hasDistinctHostsConstraint(job.Constraints)
}

func (iter *ProposedAllocConstraintIterator) hasDistinctHostsConstraint(constraints []*structs.Constraint) bool {
	for _, con := range constraints {
		if con.Operand == structs.ConstraintDistinctHosts {
			return true
		}
	}
	return false
}

func (iter *ProposedAllocConstraintIterator) Next() *structs.Node {
	for {
		// Get the next option from the source
		option := iter.source.Next()

		// Hot-path if the option is nil or there are no distinct_hosts constraints.
		if option == nil || !(iter.jobDistinctHosts || iter.tgDistinctHosts) {
			return option
		}

		if !iter.satisfiesDistinctHosts(option) {
			iter.ctx.Metrics().FilterNode(option, structs.ConstraintDistinctHosts)
			continue
		}

		return option
	}
}

// satisfiesDistinctHosts checks if the node satisfies a distinct_hosts
// constraint either specified at the job level or the TaskGroup level.
func (iter *ProposedAllocConstraintIterator) satisfiesDistinctHosts(option *structs.Node) bool {
	// Check if there is no constraint set.
	if !(iter.jobDistinctHosts || iter.tgDistinctHosts) {
		return true
	}

	// Get the proposed allocations
	proposed, err := iter.ctx.ProposedAllocs(option.ID)
	if err != nil {
		iter.ctx.Logger().Printf(
			"[ERR] scheduler.dynamic-constraint: failed to get proposed allocations: %v", err)
		return false
	}

	// Skip the node if the task group has already been allocated on it.
	for _, alloc := range proposed {
		// If the job has a distinct_hosts constraint we only need an alloc
		// collision on the JobID but if the constraint is on the TaskGroup then
		// we need both a job and TaskGroup collision.
		jobCollision := alloc.JobID == iter.job.ID
		taskCollision := alloc.TaskGroup == iter.tg.Name
		if iter.jobDistinctHosts && jobCollision || jobCollision && taskCollision {
			return false
		}
	}

	return true
}

func (iter *ProposedAllocConstraintIterator) Reset() {
	iter.source.Reset()
}

// ConstraintChecker is a FeasibilityChecker which returns nodes that match a
// given set of constraints. This is used to filter on job, task group, and task
// constraints.
type ConstraintChecker struct {
	ctx         Context
	constraints []*structs.Constraint
}

// NewConstraintChecker creates a ConstraintChecker for a set of constraints
func NewConstraintChecker(ctx Context, constraints []*structs.Constraint) *ConstraintChecker {
	return &ConstraintChecker{
		ctx:         ctx,
		constraints: constraints,
	}
}

func (c *ConstraintChecker) SetConstraints(constraints []*structs.Constraint) {
	c.constraints = constraints
}

func (c *ConstraintChecker) Feasible(option *structs.Node) bool {
	// Use this node if possible
	for _, constraint := range c.constraints {
		if !c.meetsConstraint(constraint, option) {
			c.ctx.Metrics().FilterNode(option, constraint.String())
			return false
		}
	}
	return true
}

func (c *ConstraintChecker) meetsConstraint(constraint *structs.Constraint, option *structs.Node) bool {
	// Resolve the targets
	lVal, ok := resolveConstraintTarget(constraint.LTarget, option)
	if !ok {
		return false
	}
	rVal, ok := resolveConstraintTarget(constraint.RTarget, option)
	if !ok {
		return false
	}

	// Check if satisfied
	return checkConstraint(c.ctx, constraint.Operand, lVal, rVal)
}

// resolveConstraintTarget is used to resolve the LTarget and RTarget of a Constraint
func resolveConstraintTarget(target string, node *structs.Node) (interface{}, bool) {
	// If no prefix, this must be a literal value
	if !strings.HasPrefix(target, "${") {
		return target, true
	}

	// Handle the interpolations
	switch {
	case "${node.unique.id}" == target:
		return node.ID, true

	case "${node.datacenter}" == target:
		return node.Datacenter, true

	case "${node.unique.name}" == target:
		return node.Name, true

	case "${node.class}" == target:
		return node.NodeClass, true

	case strings.HasPrefix(target, "${attr."):
		attr := strings.TrimSuffix(strings.TrimPrefix(target, "${attr."), "}")
		val, ok := node.Attributes[attr]
		return val, ok

	case strings.HasPrefix(target, "${meta."):
		meta := strings.TrimSuffix(strings.TrimPrefix(target, "${meta."), "}")
		val, ok := node.Meta[meta]
		return val, ok

	default:
		return nil, false
	}
}

// checkConstraint checks if a constraint is satisfied
func checkConstraint(ctx Context, operand string, lVal, rVal interface{}) bool {
	// Check for constraints not handled by this checker.
	switch operand {
	case structs.ConstraintDistinctHosts:
		return true
	default:
		break
	}

	switch operand {
	case "=", "==", "is":
		return reflect.DeepEqual(lVal, rVal)
	case "!=", "not":
		return !reflect.DeepEqual(lVal, rVal)
	case "<", "<=", ">", ">=":
		return checkLexicalOrder(operand, lVal, rVal)
	case structs.ConstraintVersion:
		return checkVersionConstraint(ctx, lVal, rVal)
	case structs.ConstraintRegex:
		return checkRegexpConstraint(ctx, lVal, rVal)
	default:
		return false
	}
}

// checkLexicalOrder is used to check for lexical ordering
func checkLexicalOrder(op string, lVal, rVal interface{}) bool {
	// Ensure the values are strings
	lStr, ok := lVal.(string)
	if !ok {
		return false
	}
	rStr, ok := rVal.(string)
	if !ok {
		return false
	}

	switch op {
	case "<":
		return lStr < rStr
	case "<=":
		return lStr <= rStr
	case ">":
		return lStr > rStr
	case ">=":
		return lStr >= rStr
	default:
		return false
	}
}

// checkVersionConstraint is used to compare a version on the
// left hand side with a set of constraints on the right hand side
func checkVersionConstraint(ctx Context, lVal, rVal interface{}) bool {
	// Parse the version
	var versionStr string
	switch v := lVal.(type) {
	case string:
		versionStr = v
	case int:
		versionStr = fmt.Sprintf("%d", v)
	default:
		return false
	}

	// Parse the version
	vers, err := version.NewVersion(versionStr)
	if err != nil {
		return false
	}

	// Constraint must be a string
	constraintStr, ok := rVal.(string)
	if !ok {
		return false
	}

	// Check the cache for a match
	cache := ctx.ConstraintCache()
	constraints := cache[constraintStr]

	// Parse the constraints
	if constraints == nil {
		constraints, err = version.NewConstraint(constraintStr)
		if err != nil {
			return false
		}
		cache[constraintStr] = constraints
	}

	// Check the constraints against the version
	return constraints.Check(vers)
}

// checkRegexpConstraint is used to compare a value on the
// left hand side with a regexp on the right hand side
func checkRegexpConstraint(ctx Context, lVal, rVal interface{}) bool {
	// Ensure left-hand is string
	lStr, ok := lVal.(string)
	if !ok {
		return false
	}

	// Regexp must be a string
	regexpStr, ok := rVal.(string)
	if !ok {
		return false
	}

	// Check the cache
	cache := ctx.RegexpCache()
	re := cache[regexpStr]

	// Parse the regexp
	if re == nil {
		var err error
		re, err = regexp.Compile(regexpStr)
		if err != nil {
			return false
		}
		cache[regexpStr] = re
	}

	// Look for a match
	return re.MatchString(lStr)
}

// FeasibilityWrapper is a FeasibleIterator which wraps both job and task group
// FeasibilityCheckers in which feasibility checking can be skipped if the
// computed node class has previously been marked as eligible or ineligible.
type FeasibilityWrapper struct {
	ctx         Context
	source      FeasibleIterator
	jobCheckers []FeasibilityChecker
	tgCheckers  []FeasibilityChecker
	tg          string
}

// NewFeasibilityWrapper returns a FeasibleIterator based on the passed source
// and FeasibilityCheckers.
func NewFeasibilityWrapper(ctx Context, source FeasibleIterator,
	jobCheckers, tgCheckers []FeasibilityChecker) *FeasibilityWrapper {
	return &FeasibilityWrapper{
		ctx:         ctx,
		source:      source,
		jobCheckers: jobCheckers,
		tgCheckers:  tgCheckers,
	}
}

func (w *FeasibilityWrapper) SetTaskGroup(tg string) {
	w.tg = tg
}

func (w *FeasibilityWrapper) Reset() {
	w.source.Reset()
}

// Next returns an eligible node, only running the FeasibilityCheckers as needed
// based on the sources computed node class.
func (w *FeasibilityWrapper) Next() *structs.Node {
	evalElig := w.ctx.Eligibility()
	metrics := w.ctx.Metrics()

OUTER:
	for {
		// Get the next option from the source
		option := w.source.Next()
		if option == nil {
			return nil
		}

		// Check if the job has been marked as eligible or ineligible.
		jobEscaped, jobUnknown := false, false
		switch evalElig.JobStatus(option.ComputedClass) {
		case EvalComputedClassIneligible:
			// Fast path the ineligible case
			metrics.FilterNode(option, "computed class ineligible")
			continue
		case EvalComputedClassEscaped:
			jobEscaped = true
		case EvalComputedClassUnknown:
			jobUnknown = true
		}

		// Run the job feasibility checks.
		for _, check := range w.jobCheckers {
			feasible := check.Feasible(option)
			if !feasible {
				// If the job hasn't escaped, set it to be ineligible since it
				// failed a job check.
				if !jobEscaped {
					evalElig.SetJobEligibility(false, option.ComputedClass)
				}
				continue OUTER
			}
		}

		// Set the job eligibility if the constraints weren't escaped and it
		// hasn't been set before.
		if !jobEscaped && jobUnknown {
			evalElig.SetJobEligibility(true, option.ComputedClass)
		}

		// Check if the task group has been marked as eligible or ineligible.
		tgEscaped, tgUnknown := false, false
		switch evalElig.TaskGroupStatus(w.tg, option.ComputedClass) {
		case EvalComputedClassIneligible:
			// Fast path the ineligible case
			metrics.FilterNode(option, "computed class ineligible")
			continue
		case EvalComputedClassEligible:
			// Fast path the eligible case
			return option
		case EvalComputedClassEscaped:
			tgEscaped = true
		case EvalComputedClassUnknown:
			tgUnknown = true
		}

		// Run the task group feasibility checks.
		for _, check := range w.tgCheckers {
			feasible := check.Feasible(option)
			if !feasible {
				// If the task group hasn't escaped, set it to be ineligible
				// since it failed a check.
				if !tgEscaped {
					evalElig.SetTaskGroupEligibility(false, w.tg, option.ComputedClass)
				}
				continue OUTER
			}
		}

		// Set the task group eligibility if the constraints weren't escaped and
		// it hasn't been set before.
		if !tgEscaped && tgUnknown {
			evalElig.SetTaskGroupEligibility(true, w.tg, option.ComputedClass)
		}

		return option
	}
}
