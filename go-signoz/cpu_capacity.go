package signoz

import (
	"strconv"
	"strings"
)

// normalizedCPUCapacity returns the smallest positive process CPU limit.
func normalizedCPUCapacity(runtimeCPUs, maxProcs, quota float64, hasQuota bool) float64 {
	capacity := runtimeCPUs
	if capacity <= 0 {
		capacity = 1
	}
	if maxProcs > 0 && maxProcs < capacity {
		capacity = maxProcs
	}
	if hasQuota && quota > 0 && quota < capacity {
		capacity = quota
	}
	return capacity
}

// parseCPUMax reads the quota and period fields from a cgroup v2 cpu.max file.
func parseCPUMax(rawValue string) (float64, bool) {
	fields := strings.Fields(rawValue)
	if len(fields) != 2 || fields[0] == "max" {
		return 0, false
	}
	return parseCPUQuota(fields[0], fields[1])
}

// parseCPUQuota converts cgroup quota and period values into CPU units.
func parseCPUQuota(rawQuota, rawPeriod string) (float64, bool) {
	quota, quotaErr := strconv.ParseFloat(strings.TrimSpace(rawQuota), 64)
	period, periodErr := strconv.ParseFloat(strings.TrimSpace(rawPeriod), 64)
	if quotaErr != nil || periodErr != nil || quota <= 0 || period <= 0 {
		return 0, false
	}
	return quota / period, true
}
