// +build darwin

#ifndef _GO_PROCESSDARWIN_H_INCLUDED
#define _GO_PROCESSDARWIN_H_INCLUDED

#include <errno.h>
#include <stdlib.h>
#include <sys/sysctl.h>

// This is declared in process_darwin.go
extern void go_darwin_append_proc(pid_t, pid_t, char *);

// Loads the process table and calls the exported Go function to insert
// the data back into the Go space.
//
// This function is implemented in C because while it would technically
// be possible to do this all in Go, I didn't want to go spelunking through
// header files to get all the structures properly. It is much easier to just
// call it in C and be done with it.
static inline int darwinProcesses() {
    int err = 0;
    int i = 0;
    static const int name[] = { CTL_KERN, KERN_PROC, KERN_PROC_ALL, 0 };
    size_t length = 0;
    struct kinfo_proc *result = NULL;
    size_t resultCount = 0;

    // Get the length first
    err = sysctl((int*)name, (sizeof(name) / sizeof(*name)) - 1,
            NULL, &length, NULL, 0);
    if (err != 0) {
        goto ERREXIT;
    }

    // Allocate the appropriate sized buffer to read the process list
    result = malloc(length);

    // Call sysctl again with our buffer to fill it with the process list
    err = sysctl((int*)name, (sizeof(name) / sizeof(*name)) - 1,
            result, &length,
            NULL, 0);
    if (err != 0) {
        goto ERREXIT;
    }

    resultCount = length / sizeof(struct kinfo_proc);
    for (i = 0; i < resultCount; i++) {
        struct kinfo_proc *single = &result[i];
        go_darwin_append_proc(
                single->kp_proc.p_pid,
                single->kp_eproc.e_ppid,
                single->kp_proc.p_comm);
    }

ERREXIT:
    if (result != NULL) {
        free(result);
    }

    if (err != 0) {
        return errno;
    }
    return 0;
}

#endif
