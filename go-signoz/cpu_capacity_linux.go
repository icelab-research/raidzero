//go:build linux

package signoz

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// availableCPUCapacity includes process limits and Linux cgroup quotas.
func availableCPUCapacity() float64 {
	quota, hasQuota := linuxCPUQuota()
	return normalizedCPUCapacity(
		float64(runtime.NumCPU()),
		float64(runtime.GOMAXPROCS(0)),
		quota,
		hasQuota,
	)
}

// linuxCPUQuota returns the smallest applicable cgroup v2 or v1 quota.
func linuxCPUQuota() (float64, bool) {
	var effectiveQuota float64
	hasQuota := false
	for _, quotaPath := range cgroupV2CPUPaths() {
		rawValue, err := os.ReadFile(quotaPath)
		if err == nil {
			if quota, ok := parseCPUMax(string(rawValue)); ok {
				if !hasQuota || quota < effectiveQuota {
					effectiveQuota = quota
					hasQuota = true
				}
			}
		}
	}

	for _, cpuDirectory := range cgroupV1CPUDirectories() {
		rawQuota, quotaErr := os.ReadFile(filepath.Join(cpuDirectory, "cpu.cfs_quota_us"))
		rawPeriod, periodErr := os.ReadFile(filepath.Join(cpuDirectory, "cpu.cfs_period_us"))
		if quotaErr == nil && periodErr == nil {
			if quota, ok := parseCPUQuota(string(rawQuota), string(rawPeriod)); ok {
				if !hasQuota || quota < effectiveQuota {
					effectiveQuota = quota
					hasQuota = true
				}
			}
		}
	}
	return effectiveQuota, hasQuota
}

// cgroupV2CPUPaths returns process-specific and root quota paths.
func cgroupV2CPUPaths() []string {
	paths := []string{"/sys/fs/cgroup/cpu.max"}
	for _, entry := range processCgroupEntries() {
		if entry.controllers == "" {
			paths = append([]string{filepath.Join("/sys/fs/cgroup", entry.path, "cpu.max")}, paths...)
		}
	}
	return slices.Compact(paths)
}

// cgroupV1CPUDirectories returns common process-specific CPU controller paths.
func cgroupV1CPUDirectories() []string {
	directories := []string{
		"/sys/fs/cgroup/cpu",
		"/sys/fs/cgroup/cpu,cpuacct",
	}
	for _, entry := range processCgroupEntries() {
		controllers := strings.Split(entry.controllers, ",")
		if !slices.Contains(controllers, "cpu") {
			continue
		}
		relativePath := strings.TrimPrefix(entry.path, "/")
		directories = append([]string{
			filepath.Join("/sys/fs/cgroup/cpu", relativePath),
			filepath.Join("/sys/fs/cgroup/cpu,cpuacct", relativePath),
			filepath.Join("/sys/fs/cgroup", relativePath),
		}, directories...)
	}
	return slices.Compact(directories)
}

type cgroupEntry struct {
	controllers string
	path        string
}

// processCgroupEntries parses the current process's cgroup memberships.
func processCgroupEntries() []cgroupEntry {
	rawValue, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return nil
	}

	var entries []cgroupEntry
	for _, line := range strings.Split(string(rawValue), "\n") {
		fields := strings.SplitN(line, ":", 3)
		if len(fields) != 3 {
			continue
		}
		entries = append(entries, cgroupEntry{
			controllers: fields[1],
			path:        strings.TrimPrefix(fields[2], "/"),
		})
	}
	return entries
}
