// +build windows

package net

import (
	"errors"
	"net"
	"os"
	"syscall"
	"unsafe"

	"github.com/shirou/gopsutil/internal/common"
)

var (
	modiphlpapi             = syscall.NewLazyDLL("iphlpapi.dll")
	procGetExtendedTCPTable = modiphlpapi.NewProc("GetExtendedTcpTable")
	procGetExtendedUDPTable = modiphlpapi.NewProc("GetExtendedUdpTable")
)

const (
	TCPTableBasicListener = iota
	TCPTableBasicConnections
	TCPTableBasicAll
	TCPTableOwnerPIDListener
	TCPTableOwnerPIDConnections
	TCPTableOwnerPIDAll
	TCPTableOwnerModuleListener
	TCPTableOwnerModuleConnections
	TCPTableOwnerModuleAll
)

func IOCounters(pernic bool) ([]IOCountersStat, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	ai, err := getAdapterList()
	if err != nil {
		return nil, err
	}
	var ret []IOCountersStat

	for _, ifi := range ifs {
		name := ifi.Name
		for ; ai != nil; ai = ai.Next {
			name = common.BytePtrToString(&ai.Description[0])
			c := IOCountersStat{
				Name: name,
			}

			row := syscall.MibIfRow{Index: ai.Index}
			e := syscall.GetIfEntry(&row)
			if e != nil {
				return nil, os.NewSyscallError("GetIfEntry", e)
			}
			c.BytesSent = uint64(row.OutOctets)
			c.BytesRecv = uint64(row.InOctets)
			c.PacketsSent = uint64(row.OutUcastPkts)
			c.PacketsRecv = uint64(row.InUcastPkts)
			c.Errin = uint64(row.InErrors)
			c.Errout = uint64(row.OutErrors)
			c.Dropin = uint64(row.InDiscards)
			c.Dropout = uint64(row.OutDiscards)

			ret = append(ret, c)
		}
	}

	if pernic == false {
		return getIOCountersAll(ret)
	}
	return ret, nil
}

// NetIOCountersByFile is an method which is added just a compatibility for linux.
func IOCountersByFile(pernic bool, filename string) ([]IOCountersStat, error) {
	return IOCounters(pernic)
}

// Return a list of network connections opened by a process
func Connections(kind string) ([]ConnectionStat, error) {
	var ret []ConnectionStat

	return ret, common.ErrNotImplementedError
}

// borrowed from src/pkg/net/interface_windows.go
func getAdapterList() (*syscall.IpAdapterInfo, error) {
	b := make([]byte, 1000)
	l := uint32(len(b))
	a := (*syscall.IpAdapterInfo)(unsafe.Pointer(&b[0]))
	err := syscall.GetAdaptersInfo(a, &l)
	if err == syscall.ERROR_BUFFER_OVERFLOW {
		b = make([]byte, l)
		a = (*syscall.IpAdapterInfo)(unsafe.Pointer(&b[0]))
		err = syscall.GetAdaptersInfo(a, &l)
	}
	if err != nil {
		return nil, os.NewSyscallError("GetAdaptersInfo", err)
	}
	return a, nil
}

func FilterCounters() ([]FilterStat, error) {
	return nil, errors.New("NetFilterCounters not implemented for windows")
}

// NetProtoCounters returns network statistics for the entire system
// If protocols is empty then all protocols are returned, otherwise
// just the protocols in the list are returned.
// Not Implemented for Windows
func ProtoCounters(protocols []string) ([]ProtoCountersStat, error) {
	return nil, errors.New("NetProtoCounters not implemented for windows")
}
