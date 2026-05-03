package collector

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestLinuxCollectorCollectsDefaultSamples(t *testing.T) {
	collector, err := NewCollector("linux", Dependencies{
		FileSystems: fakeFileSystems{
			"/": FileSystemStats{Blocks: 100, Bfree: 25, Files: 10, Ffree: 3},
		},
		Proc: fakeProc{
			"meminfo": []byte("MemTotal: 1000 kB\nMemAvailable: 250 kB\n"),
			"loadavg": []byte("1.23 0.50 0.25 1/100 12345\n"),
		},
	}, []string{"/"})
	if err != nil {
		t.Fatalf("NewCollector returned error: %v", err)
	}

	samples, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	assertSample(t, samples, MetricDiskUsedPercent, "/", 75)
	assertSample(t, samples, MetricDiskInodesUsedPercent, "/", 70)
	assertSample(t, samples, MetricMemoryPressurePercent, "memory", 75)
	assertSample(t, samples, MetricLoad1, "system", 1.23)
}

func TestLinuxCollectorUsesDefaultMountPoint(t *testing.T) {
	collector, err := NewCollector("linux", Dependencies{
		FileSystems: fakeFileSystems{
			"/": FileSystemStats{Blocks: 1, Bfree: 0, Files: 1, Ffree: 0},
		},
		Proc: fakeProc{
			"meminfo": []byte("MemTotal: 1 kB\nMemAvailable: 0 kB\n"),
			"loadavg": []byte("0.00 0.00 0.00 1/100 12345\n"),
		},
	}, nil)
	if err != nil {
		t.Fatalf("NewCollector returned error: %v", err)
	}

	samples, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	assertSample(t, samples, MetricDiskUsedPercent, "/", 100)
}

func TestLinuxCollectorRejectsUnsupportedPlatform(t *testing.T) {
	_, err := NewCollector("darwin", Dependencies{}, []string{"/"})
	if err == nil {
		t.Fatal("NewCollector returned nil, want unsupported platform error")
	}
	if !strings.Contains(err.Error(), "linux") {
		t.Fatalf("error %q does not explain supported platform", err)
	}
}

func TestDiskUnavailableWhenBlocksZero(t *testing.T) {
	_, err := DiskUsedPercent(FileSystemStats{})
	assertUnavailable(t, err, MetricDiskUsedPercent, "Blocks == 0")
}

func TestInodesUnavailableWhenFilesZero(t *testing.T) {
	_, err := DiskInodesUsedPercent(FileSystemStats{})
	assertUnavailable(t, err, MetricDiskInodesUsedPercent, "Files == 0")
}

func TestMemoryPressureFromMeminfo(t *testing.T) {
	value, err := MemoryPressureFromMeminfo([]byte("MemAvailable: 256 kB\nMemTotal: 1024 kB\n"))
	if err != nil {
		t.Fatalf("MemoryPressureFromMeminfo returned error: %v", err)
	}
	if value != 75 {
		t.Fatalf("memory pressure = %v, want 75", value)
	}
}

func TestMemoryPressureUnavailable(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{name: "missing MemAvailable", data: []byte("MemTotal: 1024 kB\n"), want: "MemAvailable missing"},
		{name: "zero MemTotal", data: []byte("MemTotal: 0 kB\nMemAvailable: 0 kB\n"), want: "MemTotal == 0"},
		{name: "bad value", data: []byte("MemTotal: nope kB\nMemAvailable: 0 kB\n"), want: "parse MemTotal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := MemoryPressureFromMeminfo(tt.data)
			assertUnavailable(t, err, MetricMemoryPressurePercent, tt.want)
		})
	}
}

func TestLoad1FromLoadavg(t *testing.T) {
	value, err := Load1FromLoadavg([]byte("2.50 1.00 0.25 1/100 12345\n"))
	if err != nil {
		t.Fatalf("Load1FromLoadavg returned error: %v", err)
	}
	if value != 2.5 {
		t.Fatalf("load1 = %v, want 2.5", value)
	}
}

func TestLoad1Unavailable(t *testing.T) {
	tests := []string{
		"not-a-number 1.00 0.25\n",
		"NaN 1.00 0.25\n",
		"+Inf 1.00 0.25\n",
		"-1.00 1.00 0.25\n",
	}

	for _, input := range tests {
		t.Run(strings.TrimSpace(input), func(t *testing.T) {
			_, err := Load1FromLoadavg([]byte(input))
			assertUnavailable(t, err, MetricLoad1, "unparsable")
		})
	}
}

func TestCollectorPropagatesUnavailableSources(t *testing.T) {
	collector, err := NewCollector("linux", Dependencies{
		FileSystems: fakeFileSystems{"/": FileSystemStats{}},
		Proc:        fakeProc{},
	}, []string{"/"})
	if err != nil {
		t.Fatalf("NewCollector returned error: %v", err)
	}

	_, err = collector.Collect(context.Background())
	assertUnavailable(t, err, MetricDiskUsedPercent, "Blocks == 0")
}

func TestCollectorNormalizesStatFSError(t *testing.T) {
	collector, err := NewCollector("linux", Dependencies{
		FileSystems: errorFileSystems{err: errors.New("statfs failed")},
		Proc: fakeProc{
			"meminfo": []byte("MemTotal: 1 kB\nMemAvailable: 0 kB\n"),
			"loadavg": []byte("0.00 0.00 0.00 1/100 12345\n"),
		},
	}, []string{"/"})
	if err != nil {
		t.Fatalf("NewCollector returned error: %v", err)
	}

	_, err = collector.Collect(context.Background())
	assertUnavailable(t, err, MetricDiskUsedPercent, "statfs failed")
}

func TestCollectorReportsMissingProcFileAsMetricUnavailable(t *testing.T) {
	collector, err := NewCollector("linux", Dependencies{
		FileSystems: fakeFileSystems{
			"/": FileSystemStats{Blocks: 1, Bfree: 0, Files: 1, Ffree: 0},
		},
		Proc: fakeProc{},
	}, []string{"/"})
	if err != nil {
		t.Fatalf("NewCollector returned error: %v", err)
	}

	_, err = collector.Collect(context.Background())
	assertUnavailable(t, err, MetricMemoryPressurePercent, "missing fake proc file")
}

func assertSample(t *testing.T, samples []Sample, metric, target string, value float64) {
	t.Helper()
	for _, sample := range samples {
		if sample.Metric == metric && sample.Target == target {
			if sample.Value != value {
				t.Fatalf("sample %s/%s value = %v, want %v", metric, target, sample.Value, value)
			}
			return
		}
	}
	t.Fatalf("missing sample %s/%s in %#v", metric, target, samples)
}

func assertUnavailable(t *testing.T, err error, metric, reason string) {
	t.Helper()
	if err == nil {
		t.Fatal("error = nil, want unavailable error")
	}
	var unavailable *UnavailableError
	if !errors.As(err, &unavailable) {
		t.Fatalf("error = %T %v, want UnavailableError", err, err)
	}
	if unavailable.Metric != metric {
		t.Fatalf("metric = %q, want %q", unavailable.Metric, metric)
	}
	if !strings.Contains(unavailable.Reason, reason) {
		t.Fatalf("reason = %q, want to contain %q", unavailable.Reason, reason)
	}
}

type fakeFileSystems map[string]FileSystemStats

func (f fakeFileSystems) StatFS(path string) (FileSystemStats, error) {
	stats, ok := f[path]
	if !ok {
		return FileSystemStats{}, &UnavailableError{Metric: "statfs", Target: path, Reason: "missing fake filesystem"}
	}
	return stats, nil
}

type errorFileSystems struct {
	err error
}

func (f errorFileSystems) StatFS(string) (FileSystemStats, error) {
	return FileSystemStats{}, f.err
}

type fakeProc map[string][]byte

func (f fakeProc) ReadProcFile(name string) ([]byte, error) {
	data, ok := f[name]
	if !ok {
		return nil, &UnavailableError{Metric: "proc", Target: name, Reason: "missing fake proc file"}
	}
	return data, nil
}
