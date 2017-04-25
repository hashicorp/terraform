package sysinfo

import (
	"syscall"
	"unsafe"
)

// PhysicalMemoryBytes returns the total amount of host memory.
func PhysicalMemoryBytes() (uint64, error) {
	mib := []int32{6 /* CTL_HW */, 5 /* HW_PHYSMEM */}

	buf := make([]byte, 8)
	bufLen := uintptr(8)

	_, _, e1 := syscall.Syscall6(syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])), uintptr(len(mib)),
		uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&bufLen)),
		uintptr(0), uintptr(0))

	if e1 != 0 {
		return 0, e1
	}

	switch bufLen {
	case 4:
		return uint64(*(*uint32)(unsafe.Pointer(&buf[0]))), nil
	case 8:
		return *(*uint64)(unsafe.Pointer(&buf[0])), nil
	default:
		return 0, syscall.EIO
	}
}
