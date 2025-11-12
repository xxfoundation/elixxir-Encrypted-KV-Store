// +build darwin

package ekv

/*
#include <libproc.h>
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"os"
)

// getFDCount returns the number of open file descriptors for the current process
// using the proc_pidinfo system call, which is much faster than lsof.
func getFDCount() (int, error) {
	pid := C.int(os.Getpid())

	// First call to get buffer size needed
	bufferSize := C.proc_pidinfo(pid, C.PROC_PIDLISTFDS, 0, nil, 0)
	if bufferSize <= 0 {
		return 0, fmt.Errorf("proc_pidinfo failed to get buffer size")
	}

	// Allocate buffer
	buffer := C.malloc(C.size_t(bufferSize))
	defer C.free(buffer)

	// Second call to get actual FD info
	retSize := C.proc_pidinfo(pid, C.PROC_PIDLISTFDS, 0, buffer, bufferSize)
	if retSize <= 0 {
		return 0, fmt.Errorf("proc_pidinfo failed to retrieve FDs")
	}

	// Calculate count (each entry is proc_fdinfo size)
	fdCount := int(retSize) / C.sizeof_struct_proc_fdinfo
	return fdCount, nil
}
