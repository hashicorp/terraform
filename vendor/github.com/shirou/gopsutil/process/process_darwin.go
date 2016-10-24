// +build darwin

package process

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/internal/common"
	"github.com/shirou/gopsutil/net"
)

// copied from sys/sysctl.h
const (
	CTLKern          = 1  // "high kernel": proc, limits
	KernProc         = 14 // struct: process entries
	KernProcPID      = 1  // by process id
	KernProcProc     = 8  // only return procs
	KernProcAll      = 0  // everything
	KernProcPathname = 12 // path to executable
)

const (
	ClockTicks = 100 // C.sysconf(C._SC_CLK_TCK)
)

type _Ctype_struct___0 struct {
	Pad uint64
}

// MemoryInfoExStat is different between OSes
type MemoryInfoExStat struct {
}

type MemoryMapsStat struct {
}

func Pids() ([]int32, error) {
	var ret []int32

	pids, err := callPs("pid", 0, false)
	if err != nil {
		return ret, err
	}

	for _, pid := range pids {
		v, err := strconv.Atoi(pid[0])
		if err != nil {
			return ret, err
		}
		ret = append(ret, int32(v))
	}

	return ret, nil
}

func (p *Process) Ppid() (int32, error) {
	r, err := callPs("ppid", p.Pid, false)
	v, err := strconv.Atoi(r[0][0])
	if err != nil {
		return 0, err
	}

	return int32(v), err
}
func (p *Process) Name() (string, error) {
	k, err := p.getKProc()
	if err != nil {
		return "", err
	}

	return common.IntToString(k.Proc.P_comm[:]), nil
}
func (p *Process) Exe() (string, error) {
	return "", common.ErrNotImplementedError
}

// Cmdline returns the command line arguments of the process as a string with
// each argument separated by 0x20 ascii character.
func (p *Process) Cmdline() (string, error) {
	r, err := callPs("command", p.Pid, false)
	if err != nil {
		return "", err
	}
	return strings.Join(r[0], " "), err
}

// CmdlineSlice returns the command line arguments of the process as a slice with each
// element being an argument. Because of current deficiencies in the way that the command
// line arguments are found, single arguments that have spaces in the will actually be
// reported as two separate items. In order to do something better CGO would be needed
// to use the native darwin functions.
func (p *Process) CmdlineSlice() ([]string, error) {
	r, err := callPs("command", p.Pid, false)
	if err != nil {
		return nil, err
	}
	return r[0], err
}
func (p *Process) CreateTime() (int64, error) {
	return 0, common.ErrNotImplementedError
}
func (p *Process) Cwd() (string, error) {
	return "", common.ErrNotImplementedError
}
func (p *Process) Parent() (*Process, error) {
	rr, err := common.CallLsof(invoke, p.Pid, "-FR")
	if err != nil {
		return nil, err
	}
	for _, r := range rr {
		if strings.HasPrefix(r, "p") { // skip if process
			continue
		}
		l := string(r)
		v, err := strconv.Atoi(strings.Replace(l, "R", "", 1))
		if err != nil {
			return nil, err
		}
		return NewProcess(int32(v))
	}
	return nil, fmt.Errorf("could not find parent line")
}
func (p *Process) Status() (string, error) {
	r, err := callPs("state", p.Pid, false)
	if err != nil {
		return "", err
	}

	return r[0][0], err
}
func (p *Process) Uids() ([]int32, error) {
	k, err := p.getKProc()
	if err != nil {
		return nil, err
	}

	// See: http://unix.superglobalmegacorp.com/Net2/newsrc/sys/ucred.h.html
	userEffectiveUID := int32(k.Eproc.Ucred.UID)

	return []int32{userEffectiveUID}, nil
}
func (p *Process) Gids() ([]int32, error) {
	k, err := p.getKProc()
	if err != nil {
		return nil, err
	}

	gids := make([]int32, 0, 3)
	gids = append(gids, int32(k.Eproc.Pcred.P_rgid), int32(k.Eproc.Ucred.Ngroups), int32(k.Eproc.Pcred.P_svgid))

	return gids, nil
}
func (p *Process) Terminal() (string, error) {
	return "", common.ErrNotImplementedError
	/*
		k, err := p.getKProc()
		if err != nil {
			return "", err
		}

		ttyNr := uint64(k.Eproc.Tdev)
		termmap, err := getTerminalMap()
		if err != nil {
			return "", err
		}

		return termmap[ttyNr], nil
	*/
}
func (p *Process) Nice() (int32, error) {
	k, err := p.getKProc()
	if err != nil {
		return 0, err
	}
	return int32(k.Proc.P_nice), nil
}
func (p *Process) IOnice() (int32, error) {
	return 0, common.ErrNotImplementedError
}
func (p *Process) Rlimit() ([]RlimitStat, error) {
	var rlimit []RlimitStat
	return rlimit, common.ErrNotImplementedError
}
func (p *Process) IOCounters() (*IOCountersStat, error) {
	return nil, common.ErrNotImplementedError
}
func (p *Process) NumCtxSwitches() (*NumCtxSwitchesStat, error) {
	return nil, common.ErrNotImplementedError
}
func (p *Process) NumFDs() (int32, error) {
	return 0, common.ErrNotImplementedError
}
func (p *Process) NumThreads() (int32, error) {
	r, err := callPs("utime,stime", p.Pid, true)
	if err != nil {
		return 0, err
	}
	return int32(len(r)), nil
}
func (p *Process) Threads() (map[string]string, error) {
	ret := make(map[string]string, 0)
	return ret, common.ErrNotImplementedError
}

func convertCPUTimes(s string) (ret float64, err error) {
	var t int
	var _tmp string
	if strings.Contains(s, ":") {
		_t := strings.Split(s, ":")
		hour, err := strconv.Atoi(_t[0])
		if err != nil {
			return ret, err
		}
		t += hour * 60 * 100
		_tmp = _t[1]
	} else {
		_tmp = s
	}

	_t := strings.Split(_tmp, ".")
	if err != nil {
		return ret, err
	}
	h, err := strconv.Atoi(_t[0])
	t += h * 100
	h, err = strconv.Atoi(_t[1])
	t += h
	return float64(t) / ClockTicks, nil
}
func (p *Process) Times() (*cpu.TimesStat, error) {
	r, err := callPs("utime,stime", p.Pid, false)

	if err != nil {
		return nil, err
	}

	utime, err := convertCPUTimes(r[0][0])
	if err != nil {
		return nil, err
	}
	stime, err := convertCPUTimes(r[0][1])
	if err != nil {
		return nil, err
	}

	ret := &cpu.TimesStat{
		CPU:    "cpu",
		User:   utime,
		System: stime,
	}
	return ret, nil
}
func (p *Process) CPUAffinity() ([]int32, error) {
	return nil, common.ErrNotImplementedError
}
func (p *Process) MemoryInfo() (*MemoryInfoStat, error) {
	r, err := callPs("rss,vsize,pagein", p.Pid, false)
	if err != nil {
		return nil, err
	}
	rss, err := strconv.Atoi(r[0][0])
	if err != nil {
		return nil, err
	}
	vms, err := strconv.Atoi(r[0][1])
	if err != nil {
		return nil, err
	}
	pagein, err := strconv.Atoi(r[0][2])
	if err != nil {
		return nil, err
	}

	ret := &MemoryInfoStat{
		RSS:  uint64(rss) * 1024,
		VMS:  uint64(vms) * 1024,
		Swap: uint64(pagein),
	}

	return ret, nil
}
func (p *Process) MemoryInfoEx() (*MemoryInfoExStat, error) {
	return nil, common.ErrNotImplementedError
}

func (p *Process) Children() ([]*Process, error) {
	pids, err := common.CallPgrep(invoke, p.Pid)
	if err != nil {
		return nil, err
	}
	ret := make([]*Process, 0, len(pids))
	for _, pid := range pids {
		np, err := NewProcess(pid)
		if err != nil {
			return nil, err
		}
		ret = append(ret, np)
	}
	return ret, nil
}

func (p *Process) OpenFiles() ([]OpenFilesStat, error) {
	return nil, common.ErrNotImplementedError
}

func (p *Process) Connections() ([]net.ConnectionStat, error) {
	return net.ConnectionsPid("all", p.Pid)
}

func (p *Process) NetIOCounters(pernic bool) ([]net.IOCountersStat, error) {
	return nil, common.ErrNotImplementedError
}

func (p *Process) IsRunning() (bool, error) {
	return true, common.ErrNotImplementedError
}
func (p *Process) MemoryMaps(grouped bool) (*[]MemoryMapsStat, error) {
	var ret []MemoryMapsStat
	return &ret, common.ErrNotImplementedError
}

func processes() ([]Process, error) {
	results := make([]Process, 0, 50)

	mib := []int32{CTLKern, KernProc, KernProcAll, 0}
	buf, length, err := common.CallSyscall(mib)
	if err != nil {
		return results, err
	}

	// get kinfo_proc size
	k := KinfoProc{}
	procinfoLen := int(unsafe.Sizeof(k))
	count := int(length / uint64(procinfoLen))
	/*
		fmt.Println(length, procinfoLen, count)
		b := buf[0*procinfoLen : 0*procinfoLen+procinfoLen]
		fmt.Println(b)
		kk, err := parseKinfoProc(b)
		fmt.Printf("%#v", kk)
	*/

	// parse buf to procs
	for i := 0; i < count; i++ {
		b := buf[i*procinfoLen : i*procinfoLen+procinfoLen]
		k, err := parseKinfoProc(b)
		if err != nil {
			continue
		}
		p, err := NewProcess(int32(k.Proc.P_pid))
		if err != nil {
			continue
		}
		results = append(results, *p)
	}

	return results, nil
}

func parseKinfoProc(buf []byte) (KinfoProc, error) {
	var k KinfoProc
	br := bytes.NewReader(buf)

	err := common.Read(br, binary.LittleEndian, &k)
	if err != nil {
		return k, err
	}

	return k, nil
}

// Returns a proc as defined here:
// http://unix.superglobalmegacorp.com/Net2/newsrc/sys/kinfo_proc.h.html
func (p *Process) getKProc() (*KinfoProc, error) {
	mib := []int32{CTLKern, KernProc, KernProcPID, p.Pid}
	procK := KinfoProc{}
	length := uint64(unsafe.Sizeof(procK))
	buf := make([]byte, length)
	_, _, syserr := syscall.Syscall6(
		syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		uintptr(len(mib)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&length)),
		0,
		0)
	if syserr != 0 {
		return nil, syserr
	}
	k, err := parseKinfoProc(buf)
	if err != nil {
		return nil, err
	}

	return &k, nil
}

func NewProcess(pid int32) (*Process, error) {
	p := &Process{Pid: pid}

	return p, nil
}

// call ps command.
// Return value deletes Header line(you must not input wrong arg).
// And splited by Space. Caller have responsibility to manage.
// If passed arg pid is 0, get information from all process.
func callPs(arg string, pid int32, threadOption bool) ([][]string, error) {
	bin, err := exec.LookPath("ps")
	if err != nil {
		return [][]string{}, err
	}

	var cmd []string
	if pid == 0 { // will get from all processes.
		cmd = []string{"-ax", "-o", arg}
	} else if threadOption {
		cmd = []string{"-x", "-o", arg, "-M", "-p", strconv.Itoa(int(pid))}
	} else {
		cmd = []string{"-x", "-o", arg, "-p", strconv.Itoa(int(pid))}
	}
	out, err := invoke.Command(bin, cmd...)
	if err != nil {
		return [][]string{}, err
	}
	lines := strings.Split(string(out), "\n")

	var ret [][]string
	for _, l := range lines[1:] {
		var lr []string
		for _, r := range strings.Split(l, " ") {
			if r == "" {
				continue
			}
			lr = append(lr, strings.TrimSpace(r))
		}
		if len(lr) != 0 {
			ret = append(ret, lr)
		}
	}

	return ret, nil
}
