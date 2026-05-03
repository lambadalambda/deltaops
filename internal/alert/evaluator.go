package alert

import (
	"fmt"
	"sync"
	"time"

	"deltaops/internal/collector"
)

const DefaultCooldown = 30 * time.Minute

type Severity string

const (
	SeverityNormal   Severity = "normal"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

type Kind string

const (
	KindNoop     Kind = "noop"
	KindAlert    Kind = "alert"
	KindRecovery Kind = "recovery"
)

type Threshold struct {
	Warning  float64
	Critical float64
}

type Config struct {
	Host       string
	Cooldown   time.Duration
	Thresholds map[string]Threshold
}

type Clock interface {
	Now() time.Time
}

type Evaluator struct {
	mu     sync.Mutex
	config Config
	clock  Clock
	active map[string]alertState
}

type Decision struct {
	Kind             Kind
	Host             string
	Metric           string
	Target           string
	Value            float64
	Threshold        float64
	Severity         Severity
	PreviousSeverity Severity
}

type alertState struct {
	CurrentSeverity  Severity
	NotifiedSeverity Severity
	LastSent         time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now()
}

func DefaultConfig(host string) Config {
	return Config{
		Host:     host,
		Cooldown: DefaultCooldown,
		Thresholds: map[string]Threshold{
			collector.MetricDiskUsedPercent:       {Warning: 85, Critical: 95},
			collector.MetricDiskInodesUsedPercent: {Warning: 85, Critical: 95},
			collector.MetricMemoryPressurePercent: {Warning: 80, Critical: 90},
			collector.MetricLoad1:                 {Warning: 1, Critical: 2},
		},
	}
}

func NewEvaluator(config Config, clock Clock) *Evaluator {
	if config.Cooldown == 0 {
		config.Cooldown = DefaultCooldown
	}
	config.Thresholds = mergeThresholds(DefaultConfig(config.Host).Thresholds, config.Thresholds)
	if clock == nil {
		clock = systemClock{}
	}
	return &Evaluator{config: config, clock: clock, active: make(map[string]alertState)}
}

func mergeThresholds(defaults, overrides map[string]Threshold) map[string]Threshold {
	merged := make(map[string]Threshold, len(defaults)+len(overrides))
	for metric, threshold := range defaults {
		merged[metric] = threshold
	}
	for metric, threshold := range overrides {
		base := merged[metric]
		if threshold.Warning != 0 {
			base.Warning = threshold.Warning
		}
		if threshold.Critical != 0 {
			base.Critical = threshold.Critical
		}
		merged[metric] = base
	}
	return merged
}

func (e *Evaluator) Evaluate(samples []collector.Sample) []Decision {
	e.mu.Lock()
	defer e.mu.Unlock()

	decisions := make([]Decision, 0, len(samples))
	for _, sample := range samples {
		decisions = append(decisions, e.evaluate(sample))
	}
	return decisions
}

func (e *Evaluator) evaluate(sample collector.Sample) Decision {
	threshold, ok := e.config.Thresholds[sample.Metric]
	if !ok {
		return e.noop(sample, SeverityNormal, 0)
	}
	severity, crossed := classify(sample.Value, threshold)
	key := sample.Metric + "\x00" + sample.Target
	state, active := e.active[key]

	if severity == SeverityNormal {
		if !active {
			return e.noop(sample, severity, threshold.Warning)
		}
		delete(e.active, key)
		return Decision{
			Kind:             KindRecovery,
			Host:             e.config.Host,
			Metric:           sample.Metric,
			Target:           sample.Target,
			Value:            sample.Value,
			Threshold:        threshold.Warning,
			Severity:         SeverityNormal,
			PreviousSeverity: state.CurrentSeverity,
		}
	}

	now := e.clock.Now()
	if !active || severityRank(severity) > severityRank(state.NotifiedSeverity) || now.Sub(state.LastSent) >= e.config.Cooldown {
		e.active[key] = alertState{CurrentSeverity: severity, NotifiedSeverity: severity, LastSent: now}
		return Decision{Kind: KindAlert, Host: e.config.Host, Metric: sample.Metric, Target: sample.Target, Value: sample.Value, Threshold: crossed, Severity: severity}
	}
	state.CurrentSeverity = severity
	e.active[key] = state
	return e.noop(sample, severity, crossed)
}

func (e *Evaluator) noop(sample collector.Sample, severity Severity, threshold float64) Decision {
	return Decision{Kind: KindNoop, Host: e.config.Host, Metric: sample.Metric, Target: sample.Target, Value: sample.Value, Threshold: threshold, Severity: severity}
}

func (d Decision) Message() string {
	switch d.Kind {
	case KindAlert:
		return fmt.Sprintf("host=%s check=%s target=%s severity=%s observed=%.2f threshold=%.2f state=alert", d.Host, d.Metric, d.Target, d.Severity, d.Value, d.Threshold)
	case KindRecovery:
		return fmt.Sprintf("host=%s check=%s target=%s severity=%s observed=%.2f threshold=%.2f state=recovered previous=%s", d.Host, d.Metric, d.Target, d.Severity, d.Value, d.Threshold, d.PreviousSeverity)
	default:
		return ""
	}
}

func classify(value float64, threshold Threshold) (Severity, float64) {
	if value >= threshold.Critical {
		return SeverityCritical, threshold.Critical
	}
	if value >= threshold.Warning {
		return SeverityWarning, threshold.Warning
	}
	return SeverityNormal, threshold.Warning
}

func severityRank(severity Severity) int {
	switch severity {
	case SeverityCritical:
		return 2
	case SeverityWarning:
		return 1
	default:
		return 0
	}
}
