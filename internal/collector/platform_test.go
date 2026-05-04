package collector

import (
	"strings"
	"testing"
)

func TestValidatePlatformAcceptsSupportedOSes(t *testing.T) {
	for _, goos := range []string{"linux", "darwin"} {
		if err := ValidatePlatform(goos); err != nil {
			t.Fatalf("ValidatePlatform(%q) returned error: %v", goos, err)
		}
	}
}

func TestValidatePlatformRejectsUnsupportedOS(t *testing.T) {
	err := ValidatePlatform("windows")
	if err == nil {
		t.Fatal("ValidatePlatform returned nil, want unsupported platform error")
	}
	message := err.Error()
	for _, want := range []string{"windows", "linux", "darwin", "unsupported"} {
		if !strings.Contains(message, want) {
			t.Fatalf("error %q does not include %q", message, want)
		}
	}
}

func TestNewPlanRejectsUnsupportedOS(t *testing.T) {
	_, err := NewPlan("windows")
	if err == nil {
		t.Fatal("NewPlan returned nil, want unsupported platform error")
	}
	if !strings.Contains(err.Error(), "linux") {
		t.Fatalf("error %q does not explain supported platform", err)
	}
}

func TestNewPlanReturnsLinuxMetricPlan(t *testing.T) {
	plan, err := NewPlan("linux")
	if err != nil {
		t.Fatalf("NewPlan returned error: %v", err)
	}
	if plan.GOOS != "linux" {
		t.Fatalf("GOOS = %q, want linux", plan.GOOS)
	}
	if len(plan.Metrics) != len(MVPMetricDefinitions()) {
		t.Fatalf("plan metric count = %d, want %d", len(plan.Metrics), len(MVPMetricDefinitions()))
	}
}

func TestNewPlanReturnsDarwinDevelopmentMetricPlan(t *testing.T) {
	plan, err := NewPlan("darwin")
	if err != nil {
		t.Fatalf("NewPlan returned error: %v", err)
	}
	if plan.GOOS != "darwin" {
		t.Fatalf("GOOS = %q, want darwin", plan.GOOS)
	}
	if len(plan.Metrics) != 2 {
		t.Fatalf("plan metric count = %d, want filesystem-only development metrics", len(plan.Metrics))
	}
	for _, metric := range []string{MetricDiskUsedPercent, MetricDiskInodesUsedPercent} {
		if !planHasMetric(plan, metric) {
			t.Fatalf("darwin plan missing %s", metric)
		}
	}
	for _, metric := range []string{MetricMemoryPressurePercent, MetricLoad1} {
		if planHasMetric(plan, metric) {
			t.Fatalf("darwin plan unexpectedly includes Linux proc metric %s", metric)
		}
	}
}

func TestMVPMetricDefinitions(t *testing.T) {
	definitions := MVPMetricDefinitions()
	byName := make(map[string]MetricDefinition, len(definitions))
	for _, definition := range definitions {
		if _, exists := byName[definition.Name]; exists {
			t.Fatalf("duplicate metric definition %q", definition.Name)
		}
		byName[definition.Name] = definition
	}

	tests := []struct {
		name            string
		source          string
		scope           string
		formula         string
		unavailableWhen string
	}{
		{name: MetricDiskUsedPercent, source: "statfs", scope: "mounted filesystem byte capacity", formula: "100 * (Blocks - Bfree) / Blocks", unavailableWhen: "Blocks == 0"},
		{name: MetricDiskInodesUsedPercent, source: "statfs", scope: "mounted filesystem inode capacity", formula: "100 * (Files - Ffree) / Files", unavailableWhen: "Files == 0"},
		{name: MetricMemoryPressurePercent, source: "/proc/meminfo", scope: "memory pressure", formula: "100 * (MemTotal - MemAvailable) / MemTotal", unavailableWhen: "MemTotal == 0 or MemAvailable missing"},
		{name: MetricLoad1, source: "/proc/loadavg", scope: "1-minute load average", formula: "first field of /proc/loadavg", unavailableWhen: "loadavg missing or unparsable"},
	}
	if len(definitions) != len(tests) {
		t.Fatalf("metric definition count = %d, want %d", len(definitions), len(tests))
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			definition, ok := byName[tt.name]
			if !ok {
				t.Fatalf("missing metric definition %q", tt.name)
			}
			if definition.Source != tt.source {
				t.Fatalf("Source = %q, want %q", definition.Source, tt.source)
			}
			if definition.Scope != tt.scope {
				t.Fatalf("Scope = %q, want %q", definition.Scope, tt.scope)
			}
			if definition.Formula != tt.formula {
				t.Fatalf("Formula = %q, want %q", definition.Formula, tt.formula)
			}
			if definition.UnavailableWhen != tt.unavailableWhen {
				t.Fatalf("UnavailableWhen = %q, want %q", definition.UnavailableWhen, tt.unavailableWhen)
			}
			if !definition.DefaultEnabled {
				t.Fatal("metric is not enabled by default")
			}
		})
	}
}

func planHasMetric(plan Plan, metric string) bool {
	for _, definition := range plan.Metrics {
		if definition.Name == metric {
			return true
		}
	}
	return false
}
