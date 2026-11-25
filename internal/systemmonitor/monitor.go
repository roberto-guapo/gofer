// Package systemmonitor provides on-demand, read-only host statistics. It starts
// no workers and writes no history; callers share a short-lived snapshot.
package systemmonitor

import (
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const RefreshInterval = 15 * time.Second
const SettingKey = "system_monitor_enabled"
const IntervalSettingKey = "system_monitor_interval"

// Interval accepts only the refresh choices offered by the settings UI.
func Interval(raw string) time.Duration {
	switch raw {
	case "5":
		return 5 * time.Second
	case "30":
		return 30 * time.Second
	case "60":
		return time.Minute
	default:
		return RefreshInterval
	}
}

func Supported() bool { return runtime.GOOS == "linux" }

type Usage struct {
	Used    uint64  `json:"used"`
	Total   uint64  `json:"total"`
	Percent float64 `json:"percent"`
}

type Snapshot struct {
	Supported          bool      `json:"supported"`
	SampledAt          time.Time `json:"sampled_at"`
	CPUPercent         *float64  `json:"cpu_percent"`
	Memory             *Usage    `json:"memory"`
	TemperatureC       *float64  `json:"temperature_c"`
	Disk               *Usage    `json:"disk"`
	UptimeSeconds      *float64  `json:"uptime_seconds"`
	ProcessMemoryBytes *uint64   `json:"process_memory_bytes"`
}

type cpuTimes struct{ total, idle uint64 }
type reading struct {
	snapshot Snapshot
	cpu      *cpuTimes
}

type Collector struct {
	mu   sync.Mutex
	now  func() time.Time
	read func(string) reading
	last Snapshot
	cpu  *cpuTimes
	path string
}

func New() *Collector { return &Collector{now: time.Now, read: readSystem} }

// Sample never waits to measure CPU. The first reading establishes a baseline;
// the next request measures usage since that baseline. After a viewing gap it
// starts again instead of presenting a long idle average as current usage.
func (c *Collector) Sample(diskPath string, interval time.Duration) Snapshot {
	if interval != 5*time.Second && interval != RefreshInterval && interval != 30*time.Second && interval != time.Minute {
		interval = RefreshInterval
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	age := now.Sub(c.last.SampledAt)
	if !c.last.SampledAt.IsZero() && c.path == diskPath && age >= 0 && age < interval {
		return c.last
	}
	r := c.read(diskPath)
	r.snapshot.SampledAt = now.UTC()
	if age >= 0 && age <= 3*interval && c.cpu != nil && r.cpu != nil {
		r.snapshot.CPUPercent = cpuPercent(*c.cpu, *r.cpu)
	}
	c.last, c.cpu, c.path = r.snapshot, r.cpu, diskPath
	return c.last
}

func cpuPercent(before, after cpuTimes) *float64 {
	if after.total <= before.total || after.idle < before.idle {
		return nil
	}
	total, idle := after.total-before.total, after.idle-before.idle
	if idle > total {
		return nil
	}
	v := 100 * float64(total-idle) / float64(total)
	return &v
}

func parseCPU(raw string) *cpuTimes {
	fields := strings.Fields(strings.SplitN(raw, "\n", 2)[0])
	if len(fields) < 5 || fields[0] != "cpu" {
		return nil
	}
	var cpu cpuTimes
	// guest and guest_nice are already counted in user and nice.
	for i := 1; i < len(fields) && i <= 8; i++ {
		n, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			return nil
		}
		cpu.total += n
		if i == 4 || i == 5 {
			cpu.idle += n
		}
	}
	return &cpu
}

func usage(total, available uint64) *Usage {
	if total == 0 || available > total {
		return nil
	}
	used := total - available
	return &Usage{Used: used, Total: total, Percent: 100 * float64(used) / float64(total)}
}

func parseMemory(raw string) *Usage {
	values := map[string]uint64{}
	for _, line := range strings.Split(raw, "\n") {
		f := strings.Fields(line)
		if len(f) != 3 || f[2] != "kB" {
			continue
		}
		n, err := strconv.ParseUint(f[1], 10, 64)
		if err == nil && n <= ^uint64(0)/1024 {
			values[f[0]] = n * 1024
		}
	}
	available, ok := values["MemAvailable:"]
	if !ok {
		return nil
	}
	return usage(values["MemTotal:"], available)
}
