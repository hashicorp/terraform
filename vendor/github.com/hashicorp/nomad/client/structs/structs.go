package structs

// MemoryStats holds memory usage related stats
type MemoryStats struct {
	RSS            uint64
	Cache          uint64
	Swap           uint64
	MaxUsage       uint64
	KernelUsage    uint64
	KernelMaxUsage uint64

	// A list of fields whose values were actually sampled
	Measured []string
}

func (ms *MemoryStats) Add(other *MemoryStats) {
	ms.RSS += other.RSS
	ms.Cache += other.Cache
	ms.Swap += other.Swap
	ms.MaxUsage += other.MaxUsage
	ms.KernelUsage += other.KernelUsage
	ms.KernelMaxUsage += other.KernelMaxUsage
	ms.Measured = joinStringSet(ms.Measured, other.Measured)
}

// CpuStats holds cpu usage related stats
type CpuStats struct {
	SystemMode       float64
	UserMode         float64
	TotalTicks       float64
	ThrottledPeriods uint64
	ThrottledTime    uint64
	Percent          float64

	// A list of fields whose values were actually sampled
	Measured []string
}

func (cs *CpuStats) Add(other *CpuStats) {
	cs.SystemMode += other.SystemMode
	cs.UserMode += other.UserMode
	cs.TotalTicks += other.TotalTicks
	cs.ThrottledPeriods += other.ThrottledPeriods
	cs.ThrottledTime += other.ThrottledTime
	cs.Percent += other.Percent
	cs.Measured = joinStringSet(cs.Measured, other.Measured)
}

// ResourceUsage holds information related to cpu and memory stats
type ResourceUsage struct {
	MemoryStats *MemoryStats
	CpuStats    *CpuStats
}

func (ru *ResourceUsage) Add(other *ResourceUsage) {
	ru.MemoryStats.Add(other.MemoryStats)
	ru.CpuStats.Add(other.CpuStats)
}

// TaskResourceUsage holds aggregated resource usage of all processes in a Task
// and the resource usage of the individual pids
type TaskResourceUsage struct {
	ResourceUsage *ResourceUsage
	Timestamp     int64
	Pids          map[string]*ResourceUsage
}

// AllocResourceUsage holds the aggregated task resource usage of the
// allocation.
type AllocResourceUsage struct {
	// ResourceUsage is the summation of the task resources
	ResourceUsage *ResourceUsage

	// Tasks contains the resource usage of each task
	Tasks map[string]*TaskResourceUsage

	// The max timestamp of all the Tasks
	Timestamp int64
}

// joinStringSet takes two slices of strings and joins them
func joinStringSet(s1, s2 []string) []string {
	lookup := make(map[string]struct{}, len(s1))
	j := make([]string, 0, len(s1))
	for _, s := range s1 {
		j = append(j, s)
		lookup[s] = struct{}{}
	}

	for _, s := range s2 {
		if _, ok := lookup[s]; !ok {
			j = append(j, s)
		}
	}

	return j
}
