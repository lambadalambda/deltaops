package runtime

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"deltaops/internal/alert"
	"deltaops/internal/binding"
	"deltaops/internal/collector"
)

func TestRunnerStartsWithExistingBinding(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	contact := binding.Contact{ID: "contact-1"}
	sleeper := &fakeSleeper{onSleep: func(int, time.Duration) { cancel() }}
	notifier := &fakeNotifier{}
	runner := newTestRunner(t, Dependencies{
		Account:   &fakeAccount{},
		Pairer:    &fakePairer{bound: &contact},
		Collector: &fakeCollector{samples: [][]collector.Sample{{{Metric: collector.MetricDiskUsedPercent, Target: "/", Value: 42}}, {{Metric: collector.MetricDiskUsedPercent, Target: "/", Value: 96}}}},
		Evaluator: &fakeEvaluator{decisions: [][]alert.Decision{{{Kind: alert.KindAlert, Metric: collector.MetricDiskUsedPercent}}}},
		Notifier:  notifier,
		Sleeper:   sleeper,
	})
	runner.config.Host = "host1"

	if err := runner.Run(ctx); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(notifier.reports) != 1 {
		t.Fatalf("sent reports = %d, want 1", len(notifier.reports))
	}
	if notifier.reports[0].contact != contact {
		t.Fatalf("report contact = %#v, want %#v", notifier.reports[0].contact, contact)
	}
	if notifier.reports[0].report.Reason != ReportReasonStartup || notifier.reports[0].report.Host != "host1" {
		t.Fatalf("report = %#v, want startup report for host1", notifier.reports[0].report)
	}
	if got := notifier.reports[0].report.Message(); !strings.Contains(got, "reason=startup") || !strings.Contains(got, "host=host1") || !strings.Contains(got, "disk.used_percent") || !strings.Contains(got, "observed=42.00") {
		t.Fatalf("report message %q does not include expected status fields", got)
	}
	if len(notifier.sent) != 1 {
		t.Fatalf("sent notifications = %d, want 1", len(notifier.sent))
	}
	if notifier.sent[0].contact != contact {
		t.Fatalf("notification contact = %#v, want %#v", notifier.sent[0].contact, contact)
	}
	if got, want := strings.Join(notifier.sequence, ","), "report,notify"; got != want {
		t.Fatalf("notification sequence = %s, want %s", got, want)
	}
}

func TestRunnerWaitsForPairingWhenUnbound(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	contact := binding.Contact{ID: "contact-1"}
	pairer := &fakePairer{paired: contact}
	sleeper := &fakeSleeper{onSleep: func(int, time.Duration) { cancel() }}
	notifier := &fakeNotifier{}
	runner := newTestRunner(t, Dependencies{
		Account:   &fakeAccount{},
		Pairer:    pairer,
		Collector: &fakeCollector{samples: [][]collector.Sample{{{Metric: collector.MetricMemoryPressurePercent, Target: "memory", Value: 25}}, {}}},
		Evaluator: &fakeEvaluator{decisions: [][]alert.Decision{{}}},
		Notifier:  notifier,
		Sleeper:   sleeper,
	})
	runner.config.Host = "host1"

	if err := runner.Run(ctx); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if pairer.waits != 1 {
		t.Fatalf("pairing waits = %d, want 1", pairer.waits)
	}
	if len(notifier.reports) != 1 {
		t.Fatalf("sent reports = %d, want 1", len(notifier.reports))
	}
	if notifier.reports[0].report.Reason != ReportReasonPaired {
		t.Fatalf("report reason = %q, want paired", notifier.reports[0].report.Reason)
	}
	if got := notifier.reports[0].report.Message(); !strings.Contains(got, "reason=paired") || !strings.Contains(got, "memory.pressure_percent") || !strings.Contains(got, "observed=25.00") {
		t.Fatalf("report message %q does not include pairing status", got)
	}
}

func TestReportMessageIncludesCurrentSamples(t *testing.T) {
	report := Report{Reason: ReportReasonStartup, Host: "host1", Samples: []collector.Sample{
		{Metric: collector.MetricDiskUsedPercent, Target: "/", Value: 42.125},
		{Metric: collector.MetricLoad1, Target: "system", Value: 0.5},
	}}

	message := report.Message()
	for _, want := range []string{"DeltaOps status report", "reason=startup", "host=host1", "disk.used_percent", "target=/", "observed=42.12", "load.1m", "target=system", "observed=0.50"} {
		if !strings.Contains(message, want) {
			t.Fatalf("message %q does not include %q", message, want)
		}
	}
}

func TestRunnerLogsLifecycleEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	contact := binding.Contact{ID: "contact-1"}
	logger := &fakeLogger{}
	sleeper := &fakeSleeper{onSleep: func(int, time.Duration) { cancel() }}
	runner := newTestRunner(t, Dependencies{
		Account:   &fakeAccount{},
		Pairer:    &fakePairer{paired: contact},
		Collector: &fakeCollector{samples: [][]collector.Sample{{}}},
		Evaluator: &fakeEvaluator{decisions: [][]alert.Decision{{{Kind: alert.KindAlert, Metric: collector.MetricDiskUsedPercent, Target: "/", Severity: alert.SeverityCritical}}}},
		Notifier:  &fakeNotifier{},
		Sleeper:   sleeper,
		Logger:    logger,
	})

	if err := runner.Run(ctx); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for _, name := range []string{"runtime_starting", "account_ready", "pairing_waiting", "pairing_bound", "alert_decision", "notification_sent", "runtime_shutdown"} {
		if !logger.has(name) {
			t.Fatalf("missing log event %q in %#v", name, logger.events)
		}
	}
}

func TestRunnerPollsMultipleIterationsWithoutSleeping(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	contact := binding.Contact{ID: "contact-1"}
	collector := &fakeCollector{samples: [][]collector.Sample{{}, {}}}
	sleeper := &fakeSleeper{onSleep: func(call int, _ time.Duration) {
		if call == 2 {
			cancel()
		}
	}}
	runner := newTestRunner(t, Dependencies{
		Account:   &fakeAccount{},
		Pairer:    &fakePairer{bound: &contact},
		Collector: collector,
		Evaluator: &fakeEvaluator{decisions: [][]alert.Decision{{}, {}}},
		Notifier:  &fakeNotifier{},
		Sleeper:   sleeper,
	})

	if err := runner.Run(ctx); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if collector.calls != 3 {
		t.Fatalf("collector calls = %d, want startup report plus two polls", collector.calls)
	}
}

func TestRunnerStopsOnSignal(t *testing.T) {
	contact := binding.Contact{ID: "contact-1"}
	signals := newFakeSignals()
	sleeper := &fakeSleeper{onSleep: func(int, time.Duration) { signals.Stop() }}
	runner := newTestRunner(t, Dependencies{
		Account:   &fakeAccount{},
		Pairer:    &fakePairer{bound: &contact},
		Collector: &fakeCollector{samples: [][]collector.Sample{{}}},
		Evaluator: &fakeEvaluator{decisions: [][]alert.Decision{{}}},
		Notifier:  &fakeNotifier{},
		Sleeper:   sleeper,
		Signals:   signals,
	})

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func TestRunnerRetriesAccountReadinessWithBoundedBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	contact := binding.Contact{ID: "contact-1"}
	account := &fakeAccount{errors: []error{errors.New("offline"), errors.New("still offline")}}
	sleeper := &fakeSleeper{onSleep: func(call int, _ time.Duration) {
		if call == 3 {
			cancel()
		}
	}}
	runner := newTestRunner(t, Dependencies{
		Account:   account,
		Pairer:    &fakePairer{bound: &contact},
		Collector: &fakeCollector{samples: [][]collector.Sample{{}}},
		Evaluator: &fakeEvaluator{decisions: [][]alert.Decision{{}}},
		Notifier:  &fakeNotifier{},
		Sleeper:   sleeper,
	})
	runner.config.InitialBackoff = time.Second
	runner.config.MaxBackoff = 2 * time.Second

	if err := runner.Run(ctx); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if account.calls != 3 {
		t.Fatalf("account readiness calls = %d, want 3", account.calls)
	}
	if got, want := sleeper.durations[:2], []time.Duration{time.Second, 2 * time.Second}; got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("backoff durations = %v, want %v", got, want)
	}
}

func TestRunnerRetriesNotificationFailureWithBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	contact := binding.Contact{ID: "contact-1"}
	account := &fakeAccount{}
	notifier := &fakeNotifier{errors: []error{errors.New("send failed")}}
	sleeper := &fakeSleeper{onSleep: func(call int, _ time.Duration) {
		if call == 2 {
			cancel()
		}
	}}
	runner := newTestRunner(t, Dependencies{
		Account:   account,
		Pairer:    &fakePairer{bound: &contact},
		Collector: &fakeCollector{samples: [][]collector.Sample{{}}},
		Evaluator: &fakeEvaluator{decisions: [][]alert.Decision{{{Kind: alert.KindAlert, Metric: collector.MetricDiskUsedPercent}}}},
		Notifier:  notifier,
		Sleeper:   sleeper,
	})

	if err := runner.Run(ctx); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if notifier.notifyCalls != 2 {
		t.Fatalf("notify calls = %d, want 2", notifier.notifyCalls)
	}
	if account.calls != 2 {
		t.Fatalf("account readiness calls = %d, want startup plus reconnect", account.calls)
	}
	if sleeper.durations[0] != time.Second {
		t.Fatalf("notification retry backoff = %v, want 1s", sleeper.durations[0])
	}
}

func TestRunnerStopsAfterBoundedNotificationRetries(t *testing.T) {
	contact := binding.Contact{ID: "contact-1"}
	logger := &fakeLogger{}
	runner, err := NewRunner(Config{PollInterval: time.Minute, InitialBackoff: time.Second, MaxBackoff: 2 * time.Second, MaxNotifyAttempts: 2}, Dependencies{
		Account:   &fakeAccount{},
		Pairer:    &fakePairer{bound: &contact},
		Collector: &fakeCollector{samples: [][]collector.Sample{{}}},
		Evaluator: &fakeEvaluator{decisions: [][]alert.Decision{{{Kind: alert.KindAlert, Metric: collector.MetricDiskUsedPercent}}}},
		Notifier:  &fakeNotifier{errors: []error{errors.New("send failed"), errors.New("still failed")}},
		Sleeper:   &fakeSleeper{},
		Logger:    logger,
	})
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}

	err = runner.Run(context.Background())
	if err == nil {
		t.Fatal("Run returned nil error, want bounded retry failure")
	}
	if !strings.Contains(err.Error(), "next action") || !strings.Contains(err.Error(), "Delta Chat") {
		t.Fatalf("error %q does not include operator next action", err)
	}
	if !logger.has("notification_failed") {
		t.Fatalf("missing notification_failed event in %#v", logger.events)
	}
}

func TestRunnerStopsAfterBoundedStatusReportRetries(t *testing.T) {
	contact := binding.Contact{ID: "contact-1"}
	logger := &fakeLogger{}
	account := &fakeAccount{}
	notifier := &fakeNotifier{reportErrors: []error{errors.New("send failed"), errors.New("still failed")}}
	runner, err := NewRunner(Config{PollInterval: time.Minute, InitialBackoff: time.Second, MaxBackoff: 2 * time.Second, MaxNotifyAttempts: 2}, Dependencies{
		Account:   account,
		Pairer:    &fakePairer{bound: &contact},
		Collector: &fakeCollector{samples: [][]collector.Sample{{{Metric: collector.MetricDiskUsedPercent, Target: "/", Value: 42}}}},
		Evaluator: &fakeEvaluator{},
		Notifier:  notifier,
		Sleeper:   &fakeSleeper{},
		Logger:    logger,
	})
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}

	err = runner.Run(context.Background())
	if err == nil {
		t.Fatal("Run returned nil error, want bounded report retry failure")
	}
	if !strings.Contains(err.Error(), "status report delivery failed") || !strings.Contains(err.Error(), "next action") {
		t.Fatalf("error %q does not include status report next action", err)
	}
	if notifier.reportCalls != 2 {
		t.Fatalf("report calls = %d, want 2", notifier.reportCalls)
	}
	if account.calls != 2 {
		t.Fatalf("account readiness calls = %d, want startup plus report retry readiness", account.calls)
	}
	if !logger.has("status_report_failed") {
		t.Fatalf("missing status_report_failed event in %#v", logger.events)
	}
}

func TestRunnerBoundsReadinessAfterNotificationFailure(t *testing.T) {
	contact := binding.Contact{ID: "contact-1"}
	runner, err := NewRunner(Config{PollInterval: time.Minute, InitialBackoff: time.Second, MaxBackoff: 2 * time.Second, MaxNotifyAttempts: 2}, Dependencies{
		Account:   &fakeAccount{errors: []error{nil, errors.New("account offline")}},
		Pairer:    &fakePairer{bound: &contact},
		Collector: &fakeCollector{samples: [][]collector.Sample{{}}},
		Evaluator: &fakeEvaluator{decisions: [][]alert.Decision{{{Kind: alert.KindAlert, Metric: collector.MetricDiskUsedPercent}}}},
		Notifier:  &fakeNotifier{errors: []error{errors.New("send failed")}},
		Sleeper:   &fakeSleeper{},
	})
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}

	err = runner.Run(context.Background())
	if err == nil {
		t.Fatal("Run returned nil error, want bounded readiness failure")
	}
	if !strings.Contains(err.Error(), "account not ready") || !strings.Contains(err.Error(), "next action") {
		t.Fatalf("error %q does not include bounded readiness next action", err)
	}
}

func TestRunnerRedactsRuntimeFailureLogsAndOperatorError(t *testing.T) {
	var out bytes.Buffer
	secretErr := errors.New("send failed setup_code=123456 token=abc body=raw message dcaccount:secret")
	contact := binding.Contact{ID: "contact-1"}
	runner, err := NewRunner(Config{PollInterval: time.Minute, MaxNotifyAttempts: 1}, Dependencies{
		Account:   &fakeAccount{},
		Pairer:    &fakePairer{bound: &contact},
		Collector: &fakeCollector{samples: [][]collector.Sample{{}}},
		Evaluator: &fakeEvaluator{decisions: [][]alert.Decision{{{Kind: alert.KindAlert, Metric: collector.MetricDiskUsedPercent}}}},
		Notifier:  &fakeNotifier{errors: []error{secretErr}},
		Sleeper:   &fakeSleeper{},
		Logger:    NewJSONLogger(&out),
	})
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}

	err = runner.Run(context.Background())
	if err == nil {
		t.Fatal("Run returned nil error, want notification failure")
	}
	combined := out.String() + err.Error()
	for _, secret := range []string{"123456", "token=abc", "raw message", "dcaccount:secret"} {
		if strings.Contains(combined, secret) {
			t.Fatalf("runtime output leaked secret %q in %q", secret, combined)
		}
	}
}

func TestRunnerRejectsPendingDecisionQueueOverflow(t *testing.T) {
	contact := binding.Contact{ID: "contact-1"}
	logger := &fakeLogger{}
	runner, err := NewRunner(Config{PollInterval: time.Minute, MaxPendingNotifications: 1}, Dependencies{
		Account:   &fakeAccount{},
		Pairer:    &fakePairer{bound: &contact},
		Collector: &fakeCollector{samples: [][]collector.Sample{{}}},
		Evaluator: &fakeEvaluator{decisions: [][]alert.Decision{{{Kind: alert.KindAlert, Metric: collector.MetricDiskUsedPercent}, {Kind: alert.KindRecovery, Metric: collector.MetricMemoryPressurePercent}}}},
		Notifier:  &fakeNotifier{},
		Sleeper:   &fakeSleeper{},
		Logger:    logger,
	})
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}

	err = runner.Run(context.Background())
	if err == nil {
		t.Fatal("Run returned nil error, want queue limit error")
	}
	if !strings.Contains(err.Error(), "pending notifications") || !strings.Contains(err.Error(), "next action") {
		t.Fatalf("error %q does not explain queue limit next action", err)
	}
	if !logger.has("notification_queue_full") {
		t.Fatalf("missing notification_queue_full event in %#v", logger.events)
	}
}

func TestNewRunnerRejectsNegativeDurations(t *testing.T) {
	deps := Dependencies{
		Account:   &fakeAccount{},
		Pairer:    &fakePairer{bound: &binding.Contact{ID: "contact-1"}},
		Collector: &fakeCollector{},
		Evaluator: &fakeEvaluator{},
		Notifier:  &fakeNotifier{},
		Sleeper:   &fakeSleeper{},
	}
	tests := []Config{
		{PollInterval: -time.Second},
		{InitialBackoff: -time.Second},
		{MaxBackoff: -time.Second},
	}
	for _, config := range tests {
		if _, err := NewRunner(config, deps); err == nil {
			t.Fatalf("NewRunner(%#v) returned nil error, want validation error", config)
		}
	}
}

func TestNewRunnerRejectsInvalidQueueAndRetryLimits(t *testing.T) {
	deps := Dependencies{
		Account:   &fakeAccount{},
		Pairer:    &fakePairer{bound: &binding.Contact{ID: "contact-1"}},
		Collector: &fakeCollector{},
		Evaluator: &fakeEvaluator{},
		Notifier:  &fakeNotifier{},
		Sleeper:   &fakeSleeper{},
	}
	tests := []Config{
		{MaxNotifyAttempts: -1},
		{MaxPendingNotifications: -1},
	}
	for _, config := range tests {
		if _, err := NewRunner(config, deps); err == nil {
			t.Fatalf("NewRunner(%#v) returned nil error, want validation error", config)
		}
	}
}

func newTestRunner(t *testing.T, deps Dependencies) *Runner {
	t.Helper()
	runner, err := NewRunner(Config{PollInterval: time.Minute, InitialBackoff: time.Second, MaxBackoff: 5 * time.Second}, deps)
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}
	return runner
}

type fakeAccount struct {
	errors []error
	calls  int
}

func (a *fakeAccount) Ready(context.Context) error {
	a.calls++
	if len(a.errors) == 0 {
		return nil
	}
	err := a.errors[0]
	a.errors = a.errors[1:]
	return err
}

type fakePairer struct {
	bound  *binding.Contact
	paired binding.Contact
	waits  int
}

func (p *fakePairer) BoundContact() (binding.Contact, bool) {
	if p.bound == nil {
		return binding.Contact{}, false
	}
	return *p.bound, true
}

func (p *fakePairer) WaitForPairing(context.Context) (binding.Contact, error) {
	p.waits++
	return p.paired, nil
}

type fakeCollector struct {
	samples [][]collector.Sample
	calls   int
}

func (c *fakeCollector) Collect(context.Context) ([]collector.Sample, error) {
	c.calls++
	if len(c.samples) == 0 {
		return nil, nil
	}
	samples := c.samples[0]
	c.samples = c.samples[1:]
	return samples, nil
}

type fakeEvaluator struct {
	decisions [][]alert.Decision
}

func (e *fakeEvaluator) Evaluate([]collector.Sample) []alert.Decision {
	if len(e.decisions) == 0 {
		return nil
	}
	decisions := e.decisions[0]
	e.decisions = e.decisions[1:]
	return decisions
}

type fakeNotifier struct {
	errors       []error
	reportErrors []error
	calls        int
	notifyCalls  int
	reportCalls  int
	sent         []sentDecision
	reports      []sentReport
	sequence     []string
}

func (n *fakeNotifier) Notify(_ context.Context, contact binding.Contact, decision alert.Decision) error {
	n.calls++
	n.notifyCalls++
	n.sequence = append(n.sequence, "notify")
	if len(n.errors) > 0 {
		err := n.errors[0]
		n.errors = n.errors[1:]
		return err
	}
	n.sent = append(n.sent, sentDecision{contact: contact, decision: decision})
	return nil
}

func (n *fakeNotifier) Report(_ context.Context, contact binding.Contact, report Report) error {
	n.calls++
	n.reportCalls++
	n.sequence = append(n.sequence, "report")
	if len(n.reportErrors) > 0 {
		err := n.reportErrors[0]
		n.reportErrors = n.reportErrors[1:]
		return err
	}
	n.reports = append(n.reports, sentReport{contact: contact, report: report})
	return nil
}

type sentDecision struct {
	contact  binding.Contact
	decision alert.Decision
}

type sentReport struct {
	contact binding.Contact
	report  Report
}

type fakeSleeper struct {
	durations []time.Duration
	onSleep   func(call int, duration time.Duration)
}

func (s *fakeSleeper) Sleep(ctx context.Context, duration time.Duration) error {
	s.durations = append(s.durations, duration)
	if s.onSleep != nil {
		s.onSleep(len(s.durations), duration)
	}
	return ctx.Err()
}

type fakeSignals struct {
	done chan struct{}
	once sync.Once
}

type fakeLogger struct {
	events []LogEvent
}

func (l *fakeLogger) Log(_ context.Context, event LogEvent) {
	l.events = append(l.events, event)
}

func (l *fakeLogger) has(name string) bool {
	for _, event := range l.events {
		if event.Name == name {
			return true
		}
	}
	return false
}

func newFakeSignals() *fakeSignals {
	return &fakeSignals{done: make(chan struct{})}
}

func (s *fakeSignals) Done() <-chan struct{} {
	return s.done
}

func (s *fakeSignals) Stop() {
	s.once.Do(func() { close(s.done) })
}
