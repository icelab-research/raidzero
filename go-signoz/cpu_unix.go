//go:build unix

package signoz

import (
	"syscall"
	"time"
)

// processCPUTime returns the cumulative user+system CPU time consumed by this
// process so far, via getrusage(RUSAGE_SELF). Available on Linux and macOS.
func processCPUTime() (time.Duration, error) {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return 0, err
	}
	return timevalToDuration(ru.Utime) + timevalToDuration(ru.Stime), nil
}

// timevalToDuration converts a syscall.Timeval to a time.Duration. The Usec
// field is int64 on Linux and int32 on macOS; time.Duration accepts both.
func timevalToDuration(tv syscall.Timeval) time.Duration {
	return time.Duration(tv.Sec)*time.Second + time.Duration(tv.Usec)*time.Microsecond
}
