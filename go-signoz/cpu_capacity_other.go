//go:build !linux

package signoz

import "runtime"

// availableCPUCapacity uses runtime limits where cgroup quotas do not apply.
func availableCPUCapacity() float64 {
	return normalizedCPUCapacity(
		float64(runtime.NumCPU()),
		float64(runtime.GOMAXPROCS(0)),
		0,
		false,
	)
}
