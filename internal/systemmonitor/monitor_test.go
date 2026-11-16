package systemmonitor

import (
	"sync"
	"testing"
	"time"
)

func TestCPUAccounting(t *testing.T) {
	// Guest time must not be counted twice; iowait counts with idle.
	before := parseCPU("cpu 100 20 30 400 50 6 7 8 90 10\ncpu0 1 2 3 4\n")
	after := parseCPU("cpu 120 20 40 450 70 6 7 8 100 10\n")
	if before == nil || before.total != 621 || before.idle != 450 {
		t.Fatalf("baseline = %+v", before)
	}
	if got := cpuPercent(*before, *after); got == nil || *got != 30 {
		t.Fatalf("CPU = %v, want 30", got)
	}
	if cpuPercent(*after, *before) != nil || cpuPercent(*before, *before) != nil {
		t.Fatal("reset or unchanged counters must be unavailable")
	}
	for _, raw := range []string{"", "cpu 1 2", "cpu 1 bad 3 4", "cpu0 1 2 3 4"} {
		if parseCPU(raw) != nil {
			t.Errorf("accepted invalid CPU reading %q", raw)
		}
	}
}

func TestMemoryUsesAvailableNotFree(t *testing.T) {
	got := parseMemory("MemTotal: 1000 kB\nMemFree: 100 kB\nMemAvailable: 600 kB\nCached: 500 kB\n")
	if got == nil || got.Total != 1024000 || got.Used != 409600 || got.Percent != 40 {
		t.Fatalf("memory = %+v", got)
	}
	for _, raw := range []string{"", "MemTotal: 1000 kB\n", "MemTotal: 100 kB\nMemAvailable: 200 kB", "MemTotal: bad kB\nMemAvailable: 1 kB"} {
		if parseMemory(raw) != nil {
			t.Errorf("accepted invalid memory %q", raw)
		}
	}
}

func TestOnDemandCacheAndIdleBaseline(t *testing.T) {
	now := time.Unix(1000, 0)
	reads := 0
	c := &Collector{now: func() time.Time { return now }, read: func(string) reading {
		reads++
		return reading{snapshot: Snapshot{Supported: true}, cpu: &cpuTimes{total: uint64(reads * 100), idle: uint64(reads * 60)}}
	}}
	if reads != 0 {
		t.Fatal("collector read before first request")
	}
	first := c.Sample("/data", RefreshInterval)
	if reads != 1 || first.CPUPercent != nil {
		t.Fatal("first sample must only establish a CPU baseline")
	}
	now = now.Add(time.Second)
	var group sync.WaitGroup
	for i := 0; i < 20; i++ {
		group.Add(1)
		go func() { defer group.Done(); c.Sample("/data", RefreshInterval) }()
	}
	group.Wait()
	if reads != 1 {
		t.Fatalf("concurrent viewers caused %d reads", reads)
	}
	now = now.Add(RefreshInterval)
	next := c.Sample("/data", RefreshInterval)
	if reads != 2 || next.CPUPercent == nil || *next.CPUPercent != 40 {
		t.Fatalf("second sample = %+v, reads = %d", next, reads)
	}
	now = now.Add(10 * time.Minute)
	if c.Sample("/data", RefreshInterval).CPUPercent != nil {
		t.Fatal("CPU averaged across an inactive viewing gap")
	}
	before := reads
	c.Sample("/other", RefreshInterval)
	if reads != before+1 {
		t.Fatal("disk path change reused another filesystem's cache")
	}
}

func TestMissingReadingsStayUnavailable(t *testing.T) {
	c := &Collector{now: time.Now, read: func(string) reading { return reading{snapshot: Snapshot{Supported: true}} }}
	s := c.Sample("/missing", RefreshInterval)
	if !s.Supported || s.SampledAt.IsZero() || s.CPUPercent != nil || s.Memory != nil || s.Disk != nil || s.TemperatureC != nil {
		t.Fatalf("missing data fabricated readings: %+v", s)
	}
}

func TestRefreshIntervals(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want time.Duration
	}{
		{"", RefreshInterval}, {"15", RefreshInterval}, {"5", 5 * time.Second},
		{"30", 30 * time.Second}, {"60", time.Minute}, {"0", RefreshInterval},
		{"-1", RefreshInterval}, {"999", RefreshInterval}, {"bad", RefreshInterval},
	} {
		if got := Interval(tc.raw); got != tc.want {
			t.Errorf("Interval(%q) = %v", tc.raw, got)
		}
	}
	for _, interval := range []time.Duration{5 * time.Second, RefreshInterval, 30 * time.Second, time.Minute} {
		now := time.Unix(1000, 0)
		reads := 0
		c := &Collector{now: func() time.Time { return now }, read: func(string) reading {
			reads++
			return reading{cpu: &cpuTimes{total: uint64(reads * 100), idle: uint64(reads * 60)}}
		}}
		c.Sample("/data", interval)
		now = now.Add(interval - time.Millisecond)
		c.Sample("/data", interval)
		if reads != 1 {
			t.Fatalf("cache expired before %v", interval)
		}
		now = now.Add(time.Millisecond)
		if s := c.Sample("/data", interval); reads != 2 || s.CPUPercent == nil || *s.CPUPercent != 40 {
			t.Fatalf("refresh at %v: %+v", interval, s)
		}
	}
}
