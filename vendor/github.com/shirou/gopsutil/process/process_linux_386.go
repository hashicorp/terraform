// +build linux
// +build 386

package process

const (
	ClockTicks = 100  // C.sysconf(C._SC_CLK_TCK)
	PageSize   = 4096 // C.sysconf(C._SC_PAGE_SIZE)
)
