package sysinfo

/*
#include <unistd.h>
*/
import "C"

// PhysicalMemoryBytes returns the total amount of host memory.
func PhysicalMemoryBytes() (uint64, error) {
	// The function we're calling on Solaris is
	// long sysconf(int name);
	var pages C.long
	var pagesizeBytes C.long
	var err error

	pagesizeBytes, err = C.sysconf(C._SC_PAGE_SIZE)
	if pagesizeBytes < 1 {
		return 0, err
	}
	pages, err = C.sysconf(C._SC_PHYS_PAGES)
	if pages < 1 {
		return 0, err
	}

	return uint64(pages) * uint64(pagesizeBytes), nil
}
