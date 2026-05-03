package runtime

import (
	"context"
	"errors"
	"time"

	"deltaops/internal/alert"
	"deltaops/internal/binding"
	"deltaops/internal/collector"
)

const (
	DefaultPollInterval   = time.Minute
	DefaultInitialBackoff = time.Second
	DefaultMaxBackoff     = time.Minute
)

type Config struct {
	PollInterval   time.Duration
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
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
	config = config.withDefaults()
	return &Runner{config: config, deps: deps}, nil
}

func (r *Runner) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	r.watchSignals(ctx, cancel)

	if err := r.readyWithBackoff(ctx); err != nil {
		return graceful(err)
	}
	contact, ok := r.deps.Pairer.BoundContact()
	if !ok {
		paired, err := r.deps.Pairer.WaitForPairing(ctx)
		if err != nil {
			return graceful(err)
		}
		contact = paired
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

func (r *Runner) poll(ctx context.Context, contact binding.Contact) error {
	samples, err := r.deps.Collector.Collect(ctx)
	if err != nil {
		return err
	}
	for _, decision := range r.deps.Evaluator.Evaluate(samples) {
		if decision.Kind == alert.KindNoop {
			continue
		}
		if err := r.notifyWithBackoff(ctx, contact, decision); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) notifyWithBackoff(ctx context.Context, contact binding.Contact, decision alert.Decision) error {
	delay := r.config.InitialBackoff
	for {
		if err := r.deps.Notifier.Notify(ctx, contact, decision); err == nil {
			return nil
		}
		if err := r.readyWithBackoff(ctx); err != nil {
			return err
		}
		if err := r.deps.Sleeper.Sleep(ctx, delay); err != nil {
			return err
		}
		delay = nextBackoff(delay, r.config.MaxBackoff)
	}
}

func (r *Runner) readyWithBackoff(ctx context.Context) error {
	delay := r.config.InitialBackoff
	for {
		if err := r.deps.Account.Ready(ctx); err == nil {
			return nil
		}
		if err := r.deps.Sleeper.Sleep(ctx, delay); err != nil {
			return err
		}
		delay = nextBackoff(delay, r.config.MaxBackoff)
	}
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
