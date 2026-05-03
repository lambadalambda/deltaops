package runtime

import (
	"context"
	"errors"
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
		Collector: &fakeCollector{samples: [][]collector.Sample{{{Metric: collector.MetricDiskUsedPercent, Target: "/", Value: 96}}}},
		Evaluator: &fakeEvaluator{decisions: [][]alert.Decision{{{Kind: alert.KindAlert, Metric: collector.MetricDiskUsedPercent}}}},
		Notifier:  notifier,
		Sleeper:   sleeper,
	})

	if err := runner.Run(ctx); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(notifier.sent) != 1 {
		t.Fatalf("sent notifications = %d, want 1", len(notifier.sent))
	}
	if notifier.sent[0].contact != contact {
		t.Fatalf("notification contact = %#v, want %#v", notifier.sent[0].contact, contact)
	}
}

func TestRunnerWaitsForPairingWhenUnbound(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	contact := binding.Contact{ID: "contact-1"}
	pairer := &fakePairer{paired: contact}
	sleeper := &fakeSleeper{onSleep: func(int, time.Duration) { cancel() }}
	runner := newTestRunner(t, Dependencies{
		Account:   &fakeAccount{},
		Pairer:    pairer,
		Collector: &fakeCollector{samples: [][]collector.Sample{{}}},
		Evaluator: &fakeEvaluator{decisions: [][]alert.Decision{{}}},
		Notifier:  &fakeNotifier{},
		Sleeper:   sleeper,
	})

	if err := runner.Run(ctx); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if pairer.waits != 1 {
		t.Fatalf("pairing waits = %d, want 1", pairer.waits)
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
	if collector.calls != 2 {
		t.Fatalf("collector calls = %d, want 2", collector.calls)
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
	if notifier.calls != 2 {
		t.Fatalf("notifier calls = %d, want 2", notifier.calls)
	}
	if account.calls != 2 {
		t.Fatalf("account readiness calls = %d, want startup plus reconnect", account.calls)
	}
	if sleeper.durations[0] != time.Second {
		t.Fatalf("notification retry backoff = %v, want 1s", sleeper.durations[0])
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
	errors []error
	calls  int
	sent   []sentDecision
}

func (n *fakeNotifier) Notify(_ context.Context, contact binding.Contact, decision alert.Decision) error {
	n.calls++
	if len(n.errors) > 0 {
		err := n.errors[0]
		n.errors = n.errors[1:]
		return err
	}
	n.sent = append(n.sent, sentDecision{contact: contact, decision: decision})
	return nil
}

type sentDecision struct {
	contact  binding.Contact
	decision alert.Decision
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

func newFakeSignals() *fakeSignals {
	return &fakeSignals{done: make(chan struct{})}
}

func (s *fakeSignals) Done() <-chan struct{} {
	return s.done
}

func (s *fakeSignals) Stop() {
	s.once.Do(func() { close(s.done) })
}
