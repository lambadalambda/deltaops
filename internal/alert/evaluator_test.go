package alert

import (
	"strings"
	"testing"
	"time"

	"deltaops/internal/collector"
)

func TestEvaluatorNormalSampleNoops(t *testing.T) {
	evaluator := NewEvaluator(DefaultConfig("host-a"), newFakeClock(time.Unix(0, 0)))

	decisions := evaluator.Evaluate([]collector.Sample{{Metric: collector.MetricDiskUsedPercent, Target: "/", Value: 10}})
	if len(decisions) != 1 {
		t.Fatalf("decision count = %d, want 1", len(decisions))
	}
	if decisions[0].Kind != KindNoop {
		t.Fatalf("Kind = %q, want %q", decisions[0].Kind, KindNoop)
	}
}

func TestEvaluatorAlertsAndFormatsMessage(t *testing.T) {
	evaluator := NewEvaluator(DefaultConfig("host-a"), newFakeClock(time.Unix(0, 0)))

	decision := onlyDecision(t, evaluator.Evaluate([]collector.Sample{{Metric: collector.MetricDiskUsedPercent, Target: "/", Value: 96}}))
	if decision.Kind != KindAlert {
		t.Fatalf("Kind = %q, want %q", decision.Kind, KindAlert)
	}
	if decision.Severity != SeverityCritical {
		t.Fatalf("Severity = %q, want %q", decision.Severity, SeverityCritical)
	}
	if decision.Threshold != 95 {
		t.Fatalf("Threshold = %v, want 95", decision.Threshold)
	}

	message := decision.Message()
	for _, want := range []string{"host=host-a", "check=disk.used_percent", "target=/", "severity=critical", "observed=96.00", "threshold=95.00", "state=alert"} {
		if !strings.Contains(message, want) {
			t.Fatalf("message %q does not include %q", message, want)
		}
	}
}

func TestEvaluatorSuppressesRepeatedAlertBeforeCooldown(t *testing.T) {
	clock := newFakeClock(time.Unix(0, 0))
	evaluator := NewEvaluator(DefaultConfig("host-a"), clock)

	first := onlyDecision(t, evaluator.Evaluate([]collector.Sample{{Metric: collector.MetricDiskUsedPercent, Target: "/", Value: 90}}))
	if first.Kind != KindAlert {
		t.Fatalf("first Kind = %q, want %q", first.Kind, KindAlert)
	}
	clock.Advance(5 * time.Minute)

	repeated := onlyDecision(t, evaluator.Evaluate([]collector.Sample{{Metric: collector.MetricDiskUsedPercent, Target: "/", Value: 91}}))
	if repeated.Kind != KindNoop {
		t.Fatalf("repeated Kind = %q, want %q", repeated.Kind, KindNoop)
	}
}

func TestEvaluatorRepeatsAfterCooldown(t *testing.T) {
	clock := newFakeClock(time.Unix(0, 0))
	evaluator := NewEvaluator(DefaultConfig("host-a"), clock)

	_ = onlyDecision(t, evaluator.Evaluate([]collector.Sample{{Metric: collector.MetricDiskUsedPercent, Target: "/", Value: 90}}))
	clock.Advance(DefaultCooldown)

	repeated := onlyDecision(t, evaluator.Evaluate([]collector.Sample{{Metric: collector.MetricDiskUsedPercent, Target: "/", Value: 91}}))
	if repeated.Kind != KindAlert {
		t.Fatalf("repeated Kind = %q, want %q", repeated.Kind, KindAlert)
	}
}

func TestEvaluatorEscalatesImmediately(t *testing.T) {
	clock := newFakeClock(time.Unix(0, 0))
	evaluator := NewEvaluator(DefaultConfig("host-a"), clock)

	warning := onlyDecision(t, evaluator.Evaluate([]collector.Sample{{Metric: collector.MetricMemoryPressurePercent, Target: "memory", Value: 85}}))
	if warning.Severity != SeverityWarning {
		t.Fatalf("warning Severity = %q, want %q", warning.Severity, SeverityWarning)
	}
	clock.Advance(time.Minute)

	critical := onlyDecision(t, evaluator.Evaluate([]collector.Sample{{Metric: collector.MetricMemoryPressurePercent, Target: "memory", Value: 95}}))
	if critical.Kind != KindAlert {
		t.Fatalf("critical Kind = %q, want %q", critical.Kind, KindAlert)
	}
	if critical.Severity != SeverityCritical {
		t.Fatalf("critical Severity = %q, want %q", critical.Severity, SeverityCritical)
	}
}

func TestEvaluatorSuppressesCriticalWarningCriticalFlapBeforeCooldown(t *testing.T) {
	clock := newFakeClock(time.Unix(0, 0))
	evaluator := NewEvaluator(DefaultConfig("host-a"), clock)

	critical := onlyDecision(t, evaluator.Evaluate([]collector.Sample{{Metric: collector.MetricDiskUsedPercent, Target: "/", Value: 96}}))
	if critical.Severity != SeverityCritical {
		t.Fatalf("critical Severity = %q, want %q", critical.Severity, SeverityCritical)
	}
	clock.Advance(time.Minute)

	warning := onlyDecision(t, evaluator.Evaluate([]collector.Sample{{Metric: collector.MetricDiskUsedPercent, Target: "/", Value: 90}}))
	if warning.Kind != KindNoop {
		t.Fatalf("warning Kind = %q, want %q", warning.Kind, KindNoop)
	}
	clock.Advance(time.Minute)

	repeatedCritical := onlyDecision(t, evaluator.Evaluate([]collector.Sample{{Metric: collector.MetricDiskUsedPercent, Target: "/", Value: 96}}))
	if repeatedCritical.Kind != KindNoop {
		t.Fatalf("repeated critical Kind = %q, want %q", repeatedCritical.Kind, KindNoop)
	}
}

func TestEvaluatorRecovers(t *testing.T) {
	evaluator := NewEvaluator(DefaultConfig("host-a"), newFakeClock(time.Unix(0, 0)))

	_ = onlyDecision(t, evaluator.Evaluate([]collector.Sample{{Metric: collector.MetricDiskUsedPercent, Target: "/", Value: 96}}))
	recovery := onlyDecision(t, evaluator.Evaluate([]collector.Sample{{Metric: collector.MetricDiskUsedPercent, Target: "/", Value: 50}}))
	if recovery.Kind != KindRecovery {
		t.Fatalf("Kind = %q, want %q", recovery.Kind, KindRecovery)
	}
	if recovery.Severity != SeverityNormal {
		t.Fatalf("Severity = %q, want %q", recovery.Severity, SeverityNormal)
	}
	for _, want := range []string{"host=host-a", "check=disk.used_percent", "state=recovered", "previous=critical", "observed=50.00", "threshold=85.00"} {
		if !strings.Contains(recovery.Message(), want) {
			t.Fatalf("message %q does not include %q", recovery.Message(), want)
		}
	}
}

func TestEvaluatorUsesCustomThreshold(t *testing.T) {
	config := Config{Host: "host-a", Thresholds: map[string]Threshold{collector.MetricLoad1: {Warning: 4, Critical: 8}}}
	evaluator := NewEvaluator(config, newFakeClock(time.Unix(0, 0)))

	decision := onlyDecision(t, evaluator.Evaluate([]collector.Sample{{Metric: collector.MetricLoad1, Target: "system", Value: 5}}))
	if decision.Kind != KindAlert {
		t.Fatalf("Kind = %q, want %q", decision.Kind, KindAlert)
	}
	if decision.Severity != SeverityWarning {
		t.Fatalf("Severity = %q, want %q", decision.Severity, SeverityWarning)
	}
	if decision.Threshold != 4 {
		t.Fatalf("Threshold = %v, want 4", decision.Threshold)
	}
}

func TestEvaluatorMergesPartialCustomThresholdsWithDefaults(t *testing.T) {
	config := Config{Host: "host-a", Thresholds: map[string]Threshold{collector.MetricLoad1: {Warning: 4, Critical: 8}}}
	evaluator := NewEvaluator(config, newFakeClock(time.Unix(0, 0)))

	decision := onlyDecision(t, evaluator.Evaluate([]collector.Sample{{Metric: collector.MetricDiskUsedPercent, Target: "/", Value: 96}}))
	if decision.Kind != KindAlert {
		t.Fatalf("Kind = %q, want %q", decision.Kind, KindAlert)
	}
	if decision.Threshold != 95 {
		t.Fatalf("Threshold = %v, want default critical 95", decision.Threshold)
	}
}

func TestEvaluatorMergesPartialThresholdFieldsWithDefaults(t *testing.T) {
	config := Config{Host: "host-a", Thresholds: map[string]Threshold{collector.MetricLoad1: {Warning: 0.5}}}
	evaluator := NewEvaluator(config, newFakeClock(time.Unix(0, 0)))

	decision := onlyDecision(t, evaluator.Evaluate([]collector.Sample{{Metric: collector.MetricLoad1, Target: "system", Value: 1.5}}))
	if decision.Kind != KindAlert {
		t.Fatalf("Kind = %q, want %q", decision.Kind, KindAlert)
	}
	if decision.Severity != SeverityWarning {
		t.Fatalf("Severity = %q, want %q", decision.Severity, SeverityWarning)
	}
	if decision.Threshold != 0.5 {
		t.Fatalf("Threshold = %v, want warning 0.5", decision.Threshold)
	}
}

func onlyDecision(t *testing.T, decisions []Decision) Decision {
	t.Helper()
	if len(decisions) != 1 {
		t.Fatalf("decision count = %d, want 1", len(decisions))
	}
	return decisions[0]
}

type fakeClock struct {
	now time.Time
}

func newFakeClock(now time.Time) *fakeClock {
	return &fakeClock{now: now}
}

func (c *fakeClock) Now() time.Time {
	return c.now
}

func (c *fakeClock) Advance(duration time.Duration) {
	c.now = c.now.Add(duration)
}
