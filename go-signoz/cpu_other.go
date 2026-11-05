//go:build !unix

package signoz

import (
	"errors"
	"time"
)

var errCPUUnsupported = errors.New("process CPU sampling is not supported on this platform")

// processCPUTime reports that process sampling is unavailable on this platform.
func processCPUTime() (time.Duration, error) {
	return 0, errCPUUnsupported
}
