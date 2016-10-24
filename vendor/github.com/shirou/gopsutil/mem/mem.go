package mem

import (
	"encoding/json"

	"github.com/shirou/gopsutil/internal/common"
)

var invoke common.Invoker

func init() {
	invoke = common.Invoke{}
}

// Memory usage statistics. Total, Available and Used contain numbers of bytes
// for human consumption.
//
// The other fields in this struct contain kernel specific values.
type VirtualMemoryStat struct {
	// Total amount of RAM on this system
	Total uint64 `json:"total"`

	// RAM available for programs to allocate
	//
	// This value is computed from the kernel specific values.
	Available uint64 `json:"available"`

	// RAM used by programs
	//
	// This value is computed from the kernel specific values.
	Used uint64 `json:"used"`

	// Percentage of RAM used by programs
	//
	// This value is computed from the kernel specific values.
	UsedPercent float64 `json:"usedPercent"`

	// This is the kernel's notion of free memory; RAM chips whose bits nobody
	// cares about the value of right now. For a human consumable number,
	// Available is what you really want.
	Free uint64 `json:"free"`

	// OS X / BSD specific numbers:
	// http://www.macyourself.com/2010/02/17/what-is-free-wired-active-and-inactive-system-memory-ram/
	Active   uint64 `json:"active"`
	Inactive uint64 `json:"inactive"`
	Wired    uint64 `json:"wired"`

	// Linux specific numbers
	// https://www.centos.org/docs/5/html/5.1/Deployment_Guide/s2-proc-meminfo.html
	Buffers uint64 `json:"buffers"`
	Cached  uint64 `json:"cached"`
}

type SwapMemoryStat struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"usedPercent"`
	Sin         uint64  `json:"sin"`
	Sout        uint64  `json:"sout"`
}

func (m VirtualMemoryStat) String() string {
	s, _ := json.Marshal(m)
	return string(s)
}

func (m SwapMemoryStat) String() string {
	s, _ := json.Marshal(m)
	return string(s)
}
