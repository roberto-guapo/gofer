package systemmonitor

import (
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

func readSystem(diskPath string) reading {
	s := Snapshot{Supported: true}
	cpu := parseCPU(readText("/proc/stat"))
	s.Memory = parseMemory(readText("/proc/meminfo"))
	if fields := strings.Fields(readText("/proc/uptime")); len(fields) > 0 {
		if n, err := strconv.ParseFloat(fields[0], 64); err == nil && n >= 0 && !math.IsInf(n, 0) && !math.IsNaN(n) {
			s.UptimeSeconds = &n
		}
	}
	if fields := strings.Fields(readText("/proc/self/statm")); len(fields) > 1 {
		if n, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
			n *= uint64(os.Getpagesize())
			s.ProcessMemoryBytes = &n
		}
	}
	var fs unix.Statfs_t
	if err := unix.Statfs(diskPath, &fs); err == nil && fs.Bsize > 0 {
		// Report space available to Gofer, including reserved blocks as used.
		s.Disk = usage(fs.Blocks*uint64(fs.Bsize), fs.Bavail*uint64(fs.Bsize))
	}
	s.TemperatureC = readTemperature("/sys/class/thermal")
	return reading{snapshot: s, cpu: cpu}
}

func readText(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func readTemperature(root string) *float64 {
	zones, _ := filepath.Glob(filepath.Join(root, "thermal_zone*"))
	for _, zone := range zones {
		switch strings.TrimSpace(readText(filepath.Join(zone, "type"))) {
		case "cpu-thermal", "cpu_thermal", "soc_thermal", "x86_pkg_temp":
			n, err := strconv.ParseFloat(strings.TrimSpace(readText(filepath.Join(zone, "temp"))), 64)
			if err == nil && !math.IsNaN(n) && n >= -40000 && n <= 150000 {
				n /= 1000
				return &n
			}
		}
	}
	return nil
}
