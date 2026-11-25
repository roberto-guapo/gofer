package systemmonitor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTemperatureSelectsCPUSensor(t *testing.T) {
	root := t.TempDir()
	writeZone := func(name, kind, value string) {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "type"), []byte(kind), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "temp"), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	writeZone("thermal_zone0", "wifi", "99000")
	writeZone("thermal_zone1", "cpu-thermal\n", "48750\n")
	if got := readTemperature(root); got == nil || *got != 48.75 {
		t.Fatalf("CPU temperature = %v", got)
	}
	writeZone("thermal_zone1", "cpu-thermal", "NaN")
	if readTemperature(root) != nil {
		t.Fatal("invalid sensor must be unavailable")
	}
	if readTemperature(filepath.Join(root, "missing")) != nil {
		t.Fatal("missing sensors must be unavailable")
	}
}

func TestLinuxReadSystem(t *testing.T) {
	r := readSystem(t.TempDir())
	if !r.snapshot.Supported || r.cpu == nil || r.snapshot.Memory == nil || r.snapshot.Disk == nil || r.snapshot.ProcessMemoryBytes == nil || r.snapshot.UptimeSeconds == nil {
		t.Fatalf("incomplete Linux sample: %+v", r.snapshot)
	}
	if readSystem("/does-not-exist/gofer").snapshot.Disk != nil {
		t.Fatal("missing disk must not report zero usage")
	}
}
