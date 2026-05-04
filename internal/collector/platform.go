package collector

import (
	"fmt"
	"runtime"
)

const (
	MVPPlatform               = "linux"
	DarwinDevelopmentPlatform = "darwin"
)

const (
	MetricDiskUsedPercent       = "disk.used_percent"
	MetricDiskInodesUsedPercent = "disk.inodes_used_percent"
	MetricMemoryPressurePercent = "memory.pressure_percent"
	MetricLoad1                 = "load.1m"
)

type MetricDefinition struct {
	Name            string
	Source          string
	Scope           string
	Formula         string
	UnavailableWhen string
	DefaultEnabled  bool
}

type Plan struct {
	GOOS    string
	Metrics []MetricDefinition
}

func ValidateRuntimePlatform() error {
	return ValidatePlatform(runtime.GOOS)
}

func ValidatePlatform(goos string) error {
	if goos == MVPPlatform || goos == DarwinDevelopmentPlatform {
		return nil
	}
	return fmt.Errorf("unsupported operating system %q: DeltaOps collectors support linux and darwin development mode", goos)
}

func NewRuntimePlan() (Plan, error) {
	return NewPlan(runtime.GOOS)
}

func NewPlan(goos string) (Plan, error) {
	if err := ValidatePlatform(goos); err != nil {
		return Plan{}, err
	}
	return Plan{GOOS: goos, Metrics: metricDefinitionsFor(goos)}, nil
}

func metricDefinitionsFor(goos string) []MetricDefinition {
	if goos == DarwinDevelopmentPlatform {
		return DarwinDevelopmentMetricDefinitions()
	}
	return MVPMetricDefinitions()
}

func DarwinDevelopmentMetricDefinitions() []MetricDefinition {
	definitions := MVPMetricDefinitions()
	darwin := make([]MetricDefinition, 0, 2)
	for _, definition := range definitions {
		if definition.Name == MetricDiskUsedPercent || definition.Name == MetricDiskInodesUsedPercent {
			darwin = append(darwin, definition)
		}
	}
	return darwin
}

func MVPMetricDefinitions() []MetricDefinition {
	return []MetricDefinition{
		{
			Name:            MetricDiskUsedPercent,
			Source:          "statfs",
			Scope:           "mounted filesystem byte capacity",
			Formula:         "100 * (Blocks - Bfree) / Blocks",
			UnavailableWhen: "Blocks == 0",
			DefaultEnabled:  true,
		},
		{
			Name:            MetricDiskInodesUsedPercent,
			Source:          "statfs",
			Scope:           "mounted filesystem inode capacity",
			Formula:         "100 * (Files - Ffree) / Files",
			UnavailableWhen: "Files == 0",
			DefaultEnabled:  true,
		},
		{
			Name:            MetricMemoryPressurePercent,
			Source:          "/proc/meminfo",
			Scope:           "memory pressure",
			Formula:         "100 * (MemTotal - MemAvailable) / MemTotal",
			UnavailableWhen: "MemTotal == 0 or MemAvailable missing",
			DefaultEnabled:  true,
		},
		{
			Name:            MetricLoad1,
			Source:          "/proc/loadavg",
			Scope:           "1-minute load average",
			Formula:         "first field of /proc/loadavg",
			UnavailableWhen: "loadavg missing or unparsable",
			DefaultEnabled:  true,
		},
	}
}
