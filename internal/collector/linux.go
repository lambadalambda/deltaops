package collector

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type FileSystemStats struct {
	Blocks uint64
	Bfree  uint64
	Files  uint64
	Ffree  uint64
}

type FileSystems interface {
	StatFS(path string) (FileSystemStats, error)
}

type ProcReader interface {
	ReadProcFile(name string) ([]byte, error)
}

type Dependencies struct {
	FileSystems FileSystems
	Proc        ProcReader
}

type Sample struct {
	Metric string
	Target string
	Value  float64
}

type Collector struct {
	plan        Plan
	fs          FileSystems
	proc        ProcReader
	mountPoints []string
}

type UnavailableError struct {
	Metric string
	Target string
	Reason string
}

func (e *UnavailableError) Error() string {
	if e.Target == "" {
		return fmt.Sprintf("%s unavailable: %s", e.Metric, e.Reason)
	}
	return fmt.Sprintf("%s unavailable for %s: %s", e.Metric, e.Target, e.Reason)
}

func NewCollector(goos string, deps Dependencies, mountPoints []string) (*Collector, error) {
	plan, err := NewPlan(goos)
	if err != nil {
		return nil, err
	}
	if deps.FileSystems == nil {
		deps.FileSystems = osFileSystems{}
	}
	if deps.Proc == nil {
		deps.Proc = osProc{root: "/proc"}
	}
	if len(mountPoints) == 0 {
		mountPoints = []string{"/"}
	}
	return &Collector{plan: plan, fs: deps.FileSystems, proc: deps.Proc, mountPoints: mountPoints}, nil
}

func (c *Collector) Collect(ctx context.Context) ([]Sample, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	samples := make([]Sample, 0, len(c.mountPoints)*2+2)
	for _, mountPoint := range c.mountPoints {
		stats, err := c.fs.StatFS(mountPoint)
		if err != nil {
			return nil, metricUnavailable(err, MetricDiskUsedPercent, mountPoint)
		}
		diskUsed, err := DiskUsedPercent(stats)
		if err != nil {
			return nil, withTarget(err, mountPoint)
		}
		inodesUsed, err := DiskInodesUsedPercent(stats)
		if err != nil {
			return nil, withTarget(err, mountPoint)
		}
		samples = append(samples,
			Sample{Metric: MetricDiskUsedPercent, Target: mountPoint, Value: diskUsed},
			Sample{Metric: MetricDiskInodesUsedPercent, Target: mountPoint, Value: inodesUsed},
		)
	}

	if metricEnabled(c.plan, MetricMemoryPressurePercent) {
		meminfo, err := c.readProcMetric("meminfo", MetricMemoryPressurePercent)
		if err != nil {
			return nil, err
		}
		memoryPressure, err := MemoryPressureFromMeminfo(meminfo)
		if err != nil {
			return nil, err
		}
		samples = append(samples, Sample{Metric: MetricMemoryPressurePercent, Target: "memory", Value: memoryPressure})
	}

	if metricEnabled(c.plan, MetricLoad1) {
		loadavg, err := c.readProcMetric("loadavg", MetricLoad1)
		if err != nil {
			return nil, err
		}
		load1, err := Load1FromLoadavg(loadavg)
		if err != nil {
			return nil, err
		}
		samples = append(samples, Sample{Metric: MetricLoad1, Target: "system", Value: load1})
	}

	return samples, nil
}

func metricEnabled(plan Plan, metric string) bool {
	for _, definition := range plan.Metrics {
		if definition.Name == metric && definition.DefaultEnabled {
			return true
		}
	}
	return false
}

func (c *Collector) readProcMetric(name, metric string) ([]byte, error) {
	data, err := c.proc.ReadProcFile(name)
	if err == nil {
		return data, nil
	}
	return nil, metricUnavailable(err, metric, name)
}

func DiskUsedPercent(stats FileSystemStats) (float64, error) {
	return usedPercent(MetricDiskUsedPercent, stats.Blocks, stats.Bfree, "Blocks == 0", "Bfree > Blocks")
}

func DiskInodesUsedPercent(stats FileSystemStats) (float64, error) {
	return usedPercent(MetricDiskInodesUsedPercent, stats.Files, stats.Ffree, "Files == 0", "Ffree > Files")
}

func MemoryPressureFromMeminfo(data []byte) (float64, error) {
	values := make(map[string]uint64)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		if key != "MemTotal" && key != "MemAvailable" {
			continue
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, &UnavailableError{Metric: MetricMemoryPressurePercent, Reason: "parse " + key}
		}
		values[key] = value
	}

	total := values["MemTotal"]
	if total == 0 {
		return 0, &UnavailableError{Metric: MetricMemoryPressurePercent, Reason: "MemTotal == 0"}
	}
	available, ok := values["MemAvailable"]
	if !ok {
		return 0, &UnavailableError{Metric: MetricMemoryPressurePercent, Reason: "MemAvailable missing"}
	}
	if available > total {
		return 0, &UnavailableError{Metric: MetricMemoryPressurePercent, Reason: "MemAvailable > MemTotal"}
	}
	return float64(total-available) / float64(total) * 100, nil
}

func Load1FromLoadavg(data []byte) (float64, error) {
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0, &UnavailableError{Metric: MetricLoad1, Reason: "loadavg missing or unparsable"}
	}
	load1, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, &UnavailableError{Metric: MetricLoad1, Reason: "loadavg missing or unparsable"}
	}
	if math.IsNaN(load1) || math.IsInf(load1, 0) || load1 < 0 {
		return 0, &UnavailableError{Metric: MetricLoad1, Reason: "loadavg missing or unparsable"}
	}
	return load1, nil
}

func usedPercent(metric string, total, free uint64, zeroReason, invalidReason string) (float64, error) {
	if total == 0 {
		return 0, &UnavailableError{Metric: metric, Reason: zeroReason}
	}
	if free > total {
		return 0, &UnavailableError{Metric: metric, Reason: invalidReason}
	}
	return float64(total-free) / float64(total) * 100, nil
}

func withTarget(err error, target string) error {
	var unavailable *UnavailableError
	if !errors.As(err, &unavailable) {
		return err
	}
	copy := *unavailable
	copy.Target = target
	return &copy
}

func metricUnavailable(err error, metric, target string) error {
	var unavailable *UnavailableError
	if errors.As(err, &unavailable) {
		copy := *unavailable
		copy.Metric = metric
		if copy.Target == "" {
			copy.Target = target
		}
		return &copy
	}
	return &UnavailableError{Metric: metric, Target: target, Reason: err.Error()}
}

type osProc struct {
	root string
}

func (p osProc) ReadProcFile(name string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(p.root, name))
	if err != nil {
		return nil, &UnavailableError{Metric: "proc", Target: name, Reason: err.Error()}
	}
	return data, nil
}
