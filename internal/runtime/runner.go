package runtime

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"deltaops/internal/alert"
	"deltaops/internal/binding"
	"deltaops/internal/collector"
)

const (
	DefaultPollInterval            = time.Minute
	DefaultInitialBackoff          = time.Second
	DefaultMaxBackoff              = time.Minute
	DefaultMaxNotifyAttempts       = 3
	DefaultMaxPendingNotifications = 32
)

type Config struct {
	Host                    string
	PollInterval            time.Duration
	InitialBackoff          time.Duration
	MaxBackoff              time.Duration
	MaxNotifyAttempts       int
	MaxPendingNotifications int
}

type Account interface {
	Ready(context.Context) error
}

type Pairer interface {
	BoundContact() (binding.Contact, bool)
	WaitForPairing(context.Context) (binding.Contact, error)
}

type Collector interface {
	Collect(context.Context) ([]collector.Sample, error)
}

type Evaluator interface {
	Evaluate([]collector.Sample) []alert.Decision
}

type Notifier interface {
	Notify(context.Context, binding.Contact, alert.Decision) error
	Report(context.Context, binding.Contact, Report) error
}

type Sleeper interface {
	Sleep(context.Context, time.Duration) error
}

type SignalSource interface {
	Done() <-chan struct{}
}

type Dependencies struct {
	Account   Account
	Pairer    Pairer
	Collector Collector
	Evaluator Evaluator
	Notifier  Notifier
	Sleeper   Sleeper
	Signals   SignalSource
	Logger    Logger
}

type Runner struct {
	config Config
	deps   Dependencies
}

type realSleeper struct{}

func NewRunner(config Config, deps Dependencies) (*Runner, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	if deps.Account == nil {
		return nil, errors.New("runtime account dependency is required")
	}
	if deps.Pairer == nil {
		return nil, errors.New("runtime pairer dependency is required")
	}
	if deps.Collector == nil {
		return nil, errors.New("runtime collector dependency is required")
	}
	if deps.Evaluator == nil {
		return nil, errors.New("runtime evaluator dependency is required")
	}
	if deps.Notifier == nil {
		return nil, errors.New("runtime notifier dependency is required")
	}
	if deps.Sleeper == nil {
		deps.Sleeper = realSleeper{}
	}
	if deps.Logger == nil {
		deps.Logger = discardLogger{}
	}
	config = config.withDefaults()
	return &Runner{config: config, deps: deps}, nil
}

func (r *Runner) Run(ctx context.Context) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	r.watchSignals(ctx, cancel)
	r.log(ctx, "runtime_starting", map[string]string{
		"poll_interval":   r.config.PollInterval.String(),
		"initial_backoff": r.config.InitialBackoff.String(),
		"max_backoff":     r.config.MaxBackoff.String(),
	})
	defer func() {
		fields := map[string]string{"reason": "stopped"}
		if err != nil {
			fields["reason"] = "error"
			fields["error"] = err.Error()
		}
		r.log(ctx, "runtime_shutdown", fields)
	}()

	if err := r.readyWithBackoff(ctx); err != nil {
		return graceful(err)
	}
	contact, ok := r.deps.Pairer.BoundContact()
	reportReason := ReportReasonStartup
	if !ok {
		r.log(ctx, "pairing_waiting", nil)
		paired, err := r.deps.Pairer.WaitForPairing(ctx)
		if err != nil {
			return graceful(err)
		}
		contact = paired
		reportReason = ReportReasonPaired
		r.log(ctx, "pairing_bound", map[string]string{"source": "paired"})
	} else {
		r.log(ctx, "pairing_bound", map[string]string{"source": "existing"})
	}
	if err := r.sendStatusReport(ctx, contact, reportReason); err != nil {
		return graceful(err)
	}

	for {
		if err := ctx.Err(); err != nil {
			return graceful(err)
		}
		if err := r.poll(ctx, contact); err != nil {
			return graceful(err)
		}
		if err := r.deps.Sleeper.Sleep(ctx, r.config.PollInterval); err != nil {
			return graceful(err)
		}
	}
}

func (r *Runner) sendStatusReport(ctx context.Context, contact binding.Contact, reason ReportReason) error {
	samples, err := r.deps.Collector.Collect(ctx)
	if err != nil {
		return err
	}
	return r.reportWithBackoff(ctx, contact, Report{Reason: reason, Host: r.config.Host, Samples: samples})
}

func (r *Runner) poll(ctx context.Context, contact binding.Contact) error {
	samples, err := r.deps.Collector.Collect(ctx)
	if err != nil {
		return err
	}
	pending := make([]alert.Decision, 0, len(samples))
	for _, decision := range r.deps.Evaluator.Evaluate(samples) {
		if decision.Kind == alert.KindNoop {
			continue
		}
		pending = append(pending, decision)
		if len(pending) > r.config.MaxPendingNotifications {
			r.log(ctx, "notification_queue_full", map[string]string{"limit": strconv.Itoa(r.config.MaxPendingNotifications)})
			return &OperatorError{Message: "too many pending notifications", NextAction: "increase max pending notifications or reduce alert fan-out"}
		}
	}
	for _, decision := range pending {
		r.log(ctx, "alert_decision", decisionFields(decision))
		if err := r.notifyWithBackoff(ctx, contact, decision); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) reportWithBackoff(ctx context.Context, contact binding.Contact, report Report) error {
	delay := r.config.InitialBackoff
	attempt := 1
	for {
		if err := r.deps.Notifier.Report(ctx, contact, report); err == nil {
			r.log(ctx, "status_report_sent", map[string]string{"attempt": strconv.Itoa(attempt), "reason": string(report.Reason), "sample_count": strconv.Itoa(len(report.Samples))})
			return nil
		} else {
			r.log(ctx, "status_report_failed", map[string]string{"attempt": strconv.Itoa(attempt), "reason": string(report.Reason), "error": err.Error()})
			if attempt >= r.config.MaxNotifyAttempts {
				return &OperatorError{Message: fmt.Sprintf("status report delivery failed after %d attempts", attempt), NextAction: "check Delta Chat account state, network connectivity, and local logs", Cause: err}
			}
		}
		if err := r.readyForNotificationRetry(ctx, r.config.MaxNotifyAttempts-attempt); err != nil {
			return err
		}
		r.log(ctx, "status_report_retrying", map[string]string{"delay": delay.String(), "next_attempt": strconv.Itoa(attempt + 1)})
		if err := r.deps.Sleeper.Sleep(ctx, delay); err != nil {
			return err
		}
		delay = nextBackoff(delay, r.config.MaxBackoff)
		attempt++
	}
}

func (r *Runner) notifyWithBackoff(ctx context.Context, contact binding.Contact, decision alert.Decision) error {
	delay := r.config.InitialBackoff
	attempt := 1
	for {
		if err := r.deps.Notifier.Notify(ctx, contact, decision); err == nil {
			r.log(ctx, "notification_sent", map[string]string{"attempt": strconv.Itoa(attempt), "metric": decision.Metric, "target": decision.Target})
			return nil
		} else {
			r.log(ctx, "notification_failed", map[string]string{"attempt": strconv.Itoa(attempt), "metric": decision.Metric, "target": decision.Target, "error": err.Error()})
			if attempt >= r.config.MaxNotifyAttempts {
				return &OperatorError{Message: fmt.Sprintf("notification delivery failed after %d attempts", attempt), NextAction: "check Delta Chat account state, network connectivity, and local logs", Cause: err}
			}
		}
		if err := r.readyForNotificationRetry(ctx, r.config.MaxNotifyAttempts-attempt); err != nil {
			return err
		}
		r.log(ctx, "notification_retrying", map[string]string{"delay": delay.String(), "next_attempt": strconv.Itoa(attempt + 1)})
		if err := r.deps.Sleeper.Sleep(ctx, delay); err != nil {
			return err
		}
		delay = nextBackoff(delay, r.config.MaxBackoff)
		attempt++
	}
}

func (r *Runner) readyWithBackoff(ctx context.Context) error {
	delay := r.config.InitialBackoff
	for {
		if err := r.deps.Account.Ready(ctx); err == nil {
			r.log(ctx, "account_ready", nil)
			return nil
		} else {
			r.log(ctx, "account_not_ready", map[string]string{"delay": delay.String(), "error": err.Error()})
		}
		if err := r.deps.Sleeper.Sleep(ctx, delay); err != nil {
			return err
		}
		delay = nextBackoff(delay, r.config.MaxBackoff)
	}
}

func (r *Runner) readyForNotificationRetry(ctx context.Context, attempts int) error {
	if attempts < 1 {
		attempts = 1
	}
	delay := r.config.InitialBackoff
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := r.deps.Account.Ready(ctx); err == nil {
			r.log(ctx, "account_ready", nil)
			return nil
		} else {
			r.log(ctx, "account_not_ready", map[string]string{"delay": delay.String(), "error": err.Error()})
			if attempt == attempts {
				return &OperatorError{Message: "account not ready after notification failure", NextAction: "check Delta Chat account state, network connectivity, and local logs", Cause: err}
			}
		}
		if err := r.deps.Sleeper.Sleep(ctx, delay); err != nil {
			return err
		}
		delay = nextBackoff(delay, r.config.MaxBackoff)
	}
	return nil
}

func (r *Runner) watchSignals(ctx context.Context, cancel context.CancelFunc) {
	if r.deps.Signals == nil {
		return
	}
	go func() {
		select {
		case <-ctx.Done():
		case <-r.deps.Signals.Done():
			cancel()
		}
	}()
}

func (c Config) withDefaults() Config {
	if c.PollInterval == 0 {
		c.PollInterval = DefaultPollInterval
	}
	if c.InitialBackoff == 0 {
		c.InitialBackoff = DefaultInitialBackoff
	}
	if c.MaxBackoff == 0 {
		c.MaxBackoff = DefaultMaxBackoff
	}
	if c.MaxBackoff < c.InitialBackoff {
		c.MaxBackoff = c.InitialBackoff
	}
	if c.MaxNotifyAttempts == 0 {
		c.MaxNotifyAttempts = DefaultMaxNotifyAttempts
	}
	if c.MaxPendingNotifications == 0 {
		c.MaxPendingNotifications = DefaultMaxPendingNotifications
	}
	return c
}

func validateConfig(config Config) error {
	if config.PollInterval < 0 {
		return errors.New("runtime poll interval must not be negative")
	}
	if config.InitialBackoff < 0 {
		return errors.New("runtime initial backoff must not be negative")
	}
	if config.MaxBackoff < 0 {
		return errors.New("runtime max backoff must not be negative")
	}
	if config.MaxNotifyAttempts < 0 {
		return errors.New("runtime max notify attempts must not be negative")
	}
	if config.MaxPendingNotifications < 0 {
		return errors.New("runtime max pending notifications must not be negative")
	}
	return nil
}

func nextBackoff(current, max time.Duration) time.Duration {
	next := current * 2
	if next > max {
		return max
	}
	return next
}

func graceful(err error) error {
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func (realSleeper) Sleep(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (r *Runner) log(ctx context.Context, name string, fields map[string]string) {
	r.deps.Logger.Log(ctx, LogEvent{Name: name, Fields: fields})
}

func decisionFields(decision alert.Decision) map[string]string {
	return map[string]string{
		"kind":     string(decision.Kind),
		"metric":   decision.Metric,
		"target":   decision.Target,
		"severity": string(decision.Severity),
	}
}
