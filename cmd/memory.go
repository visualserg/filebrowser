package cmd

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// imageMemoryBudget turns the imageMemoryBudget flag (MiB) into bytes. Zero
// picks a quarter of the memory limit: a few huge images decoded at once
// otherwise push a small container into swap thrashing.
func imageMemoryBudget(flagMiB int) int64 {
	switch {
	case flagMiB < 0:
		return 0
	case flagMiB > 0:
		return int64(flagMiB) << 20
	}
	return memoryLimit() / 4
}

// memoryLimit returns the cgroup memory limit, or the total RAM when there is
// none, or 0 when neither is known.
func memoryLimit() int64 {
	for _, path := range []string{
		"/sys/fs/cgroup/memory.max",                   // cgroup v2
		"/sys/fs/cgroup/memory/memory.limit_in_bytes", // cgroup v1
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		limit, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		// "max" on v2 and a huge number on v1 mean no limit.
		if err == nil && limit > 0 && limit < 1<<50 {
			return limit
		}
	}

	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[0] == "MemTotal:" {
			kib, err := strconv.ParseInt(fields[1], 10, 64)
			if err == nil {
				return kib << 10
			}
		}
	}
	return 0
}
