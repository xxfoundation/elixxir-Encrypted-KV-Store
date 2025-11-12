// +build linux

package ekv

import (
	"os"
)

// getFDCount returns the number of open file descriptors for the current process
// by counting entries in /proc/self/fd.
func getFDCount() (int, error) {
	files, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		return 0, err
	}
	return len(files), nil
}
