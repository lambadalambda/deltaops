package notify

import (
	"context"
	"errors"
	"strings"
	"testing"

	"deltaops/internal/alert"
	"deltaops/internal/binding"
	"deltaops/internal/collector"
	appruntime "deltaops/internal/runtime"
)

func TestPairerWaitsForValidSetupCode(t *testing.T) {
	manager := newMemoryManager(t, "123456")
	transport := &fakeRuntimeTransport{incoming: []IncomingMessage{
		{From: Contact{ID: "1"}, Text: "hello"},
		{From: Contact{ID: "2", Address: "operator@example.test"}, Text: "pair 123456"},
	}}
	pairer := RuntimePairer{Manager: manager, Transport: transport}

	contact, err := pairer.WaitForPairing(context.Background())
	if err != nil {
		t.Fatalf("WaitForPairing returned error: %v", err)
	}
	if contact.ID != "2" || contact.Address != "operator@example.test" {
		t.Fatalf("contact = %#v", contact)
	}
}

func TestPairerReturnsReceiveErrors(t *testing.T) {
	manager := newMemoryManager(t, "123456")
	pairer := RuntimePairer{Manager: manager, Transport: &fakeRuntimeTransport{err: errors.New("receive failed")}}

	_, err := pairer.WaitForPairing(context.Background())
	if err == nil || err.Error() != "receive failed" {
		t.Fatalf("WaitForPairing error = %v", err)
	}
}

func TestNotifierSendsDecisionMessage(t *testing.T) {
	transport := &fakeRuntimeTransport{}
	notifier := RuntimeNotifier{Transport: transport}
	decision := alert.Decision{Kind: alert.KindAlert, Host: "host1", Metric: "disk.used_percent", Target: "/", Value: 96, Threshold: 95, Severity: alert.SeverityCritical}

	if err := notifier.Notify(context.Background(), binding.Contact{ID: "2", Address: "operator@example.test"}, decision); err != nil {
		t.Fatalf("Notify returned error: %v", err)
	}
	if transport.sentTo.ID != "2" {
		t.Fatalf("sentTo = %#v", transport.sentTo)
	}
	if transport.sent.Text == "" || transport.sent.Text != decision.Message() {
		t.Fatalf("sent text = %q, want decision message", transport.sent.Text)
	}
}

func TestNotifierSendsReportMessage(t *testing.T) {
	transport := &fakeRuntimeTransport{}
	notifier := RuntimeNotifier{Transport: transport}
	report := appruntime.Report{Reason: appruntime.ReportReasonStartup, Host: "host1", Samples: []collector.Sample{{Metric: collector.MetricDiskUsedPercent, Target: "/", Value: 42}}}

	if err := notifier.Report(context.Background(), binding.Contact{ID: "2", Address: "operator@example.test"}, report); err != nil {
		t.Fatalf("Report returned error: %v", err)
	}
	if transport.sentTo.ID != "2" {
		t.Fatalf("sentTo = %#v", transport.sentTo)
	}
	if !strings.Contains(transport.sent.Text, "DeltaOps status report") || !strings.Contains(transport.sent.Text, "disk.used_percent") {
		t.Fatalf("sent text = %q, want report message", transport.sent.Text)
	}
}

type fakeRuntimeTransport struct {
	incoming []IncomingMessage
	err      error
	sentTo   Contact
	sent     OutgoingMessage
}

func (f *fakeRuntimeTransport) Account(context.Context) (Account, error) {
	return Account{}, nil
}

func (f *fakeRuntimeTransport) Receive(context.Context) (IncomingMessage, error) {
	if f.err != nil {
		return IncomingMessage{}, f.err
	}
	if len(f.incoming) == 0 {
		return IncomingMessage{}, errors.New("no messages")
	}
	message := f.incoming[0]
	f.incoming = f.incoming[1:]
	return message, nil
}

func (f *fakeRuntimeTransport) Send(_ context.Context, to Contact, message OutgoingMessage) error {
	f.sentTo = to
	f.sent = message
	return nil
}

func newMemoryManager(t *testing.T, code string) *binding.Manager {
	t.Helper()
	manager, err := binding.NewManager(code, &memoryStore{})
	if err != nil {
		t.Fatalf("NewManager returned error: %v", err)
	}
	return manager
}

type memoryStore struct {
	binding binding.Binding
	ok      bool
}

func (s *memoryStore) Load() (binding.Binding, bool, error) { return s.binding, s.ok, nil }
func (s *memoryStore) Save(binding binding.Binding) error {
	s.binding = binding
	s.ok = true
	return nil
}
func (s *memoryStore) Delete() error {
	s.binding = binding.Binding{}
	s.ok = false
	return nil
}
