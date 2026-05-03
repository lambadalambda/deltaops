package binding

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestManagerIgnoresMessagesWithoutSetupCode(t *testing.T) {
	manager := newTestManager(t, "pair-123")

	result, err := manager.Handle(Message{
		From: Contact{ID: "contact-1", Address: "operator@example.test"},
		Text: "hello",
	})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if result.Outcome != OutcomeIgnored {
		t.Fatalf("Outcome = %q, want %q", result.Outcome, OutcomeIgnored)
	}
	if _, ok := manager.BoundContact(); ok {
		t.Fatal("manager bound contact without setup code")
	}
}

func TestManagerBindsFirstMessageWithSetupCode(t *testing.T) {
	manager := newTestManager(t, "pair-123")
	contact := Contact{ID: "contact-1", Address: "operator@example.test", DisplayName: "Operator"}

	result, err := manager.Handle(Message{From: contact, Text: "deltaops pair-123"})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if result.Outcome != OutcomeBound {
		t.Fatalf("Outcome = %q, want %q", result.Outcome, OutcomeBound)
	}
	if result.Contact != contact {
		t.Fatalf("Contact = %#v, want %#v", result.Contact, contact)
	}
	bound, ok := manager.BoundContact()
	if !ok {
		t.Fatal("BoundContact returned no contact")
	}
	if bound != contact {
		t.Fatalf("BoundContact = %#v, want %#v", bound, contact)
	}
}

func TestManagerBindingSurvivesRestart(t *testing.T) {
	store := newTestStore(t)
	manager, err := NewManager("pair-123", store)
	if err != nil {
		t.Fatalf("NewManager returned error: %v", err)
	}
	contact := Contact{ID: "contact-1", Address: "operator@example.test"}
	if _, err := manager.Handle(Message{From: contact, Text: "pair-123"}); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	restarted, err := NewManager("different-code", store)
	if err != nil {
		t.Fatalf("NewManager after restart returned error: %v", err)
	}
	bound, ok := restarted.BoundContact()
	if !ok {
		t.Fatal("restarted manager did not load binding")
	}
	if bound != contact {
		t.Fatalf("restarted bound contact = %#v, want %#v", bound, contact)
	}
}

func TestManagerRejectsLaterContacts(t *testing.T) {
	manager := newTestManager(t, "pair-123")
	first := Contact{ID: "contact-1", Address: "operator@example.test"}
	second := Contact{ID: "contact-2", Address: "attacker@example.test"}

	if _, err := manager.Handle(Message{From: first, Text: "pair-123"}); err != nil {
		t.Fatalf("Handle first returned error: %v", err)
	}
	result, err := manager.Handle(Message{From: second, Text: "pair-123"})
	if err != nil {
		t.Fatalf("Handle second returned error: %v", err)
	}
	if result.Outcome != OutcomeAlreadyBound {
		t.Fatalf("Outcome = %q, want %q", result.Outcome, OutcomeAlreadyBound)
	}
	if result.Contact != first {
		t.Fatalf("Contact = %#v, want existing binding %#v", result.Contact, first)
	}
}

func TestManagerResetRequiresNewSetupCode(t *testing.T) {
	store := newTestStore(t)
	manager, err := NewManager("pair-123", store)
	if err != nil {
		t.Fatalf("NewManager returned error: %v", err)
	}
	first := Contact{ID: "contact-1", Address: "operator@example.test"}
	second := Contact{ID: "contact-2", Address: "replacement@example.test"}

	if _, err := manager.Handle(Message{From: first, Text: "pair-123"}); err != nil {
		t.Fatalf("Handle first returned error: %v", err)
	}
	if err := manager.Reset(); err != nil {
		t.Fatalf("Reset returned error: %v", err)
	}
	if _, ok := manager.BoundContact(); ok {
		t.Fatal("BoundContact returned contact after reset")
	}
	if _, err := manager.Handle(Message{From: second, Text: "pair-123"}); err == nil {
		t.Fatal("Handle returned nil error after reset without a fresh setup code")
	}

	restarted, err := NewManager("pair-456", store)
	if err != nil {
		t.Fatalf("NewManager after reset returned error: %v", err)
	}
	ignored, err := restarted.Handle(Message{From: second, Text: "pair-123"})
	if err != nil {
		t.Fatalf("Handle old code returned error: %v", err)
	}
	if ignored.Outcome != OutcomeIgnored {
		t.Fatalf("old code outcome = %q, want %q", ignored.Outcome, OutcomeIgnored)
	}

	result, err := restarted.Handle(Message{From: second, Text: "pair-456"})
	if err != nil {
		t.Fatalf("Handle second returned error: %v", err)
	}
	if result.Outcome != OutcomeBound {
		t.Fatalf("Outcome = %q, want %q", result.Outcome, OutcomeBound)
	}
	if result.Contact != second {
		t.Fatalf("Contact = %#v, want %#v", result.Contact, second)
	}
}

func TestManagerResetAfterRestartWithEmptySetupCodeCannotBind(t *testing.T) {
	store := newTestStore(t)
	manager, err := NewManager("pair-123", store)
	if err != nil {
		t.Fatalf("NewManager returned error: %v", err)
	}
	if _, err := manager.Handle(Message{From: Contact{ID: "contact-1"}, Text: "pair-123"}); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	restarted, err := NewManager("", store)
	if err != nil {
		t.Fatalf("NewManager with existing binding returned error: %v", err)
	}
	if err := restarted.Reset(); err != nil {
		t.Fatalf("Reset returned error: %v", err)
	}
	if _, err := restarted.Handle(Message{From: Contact{ID: "contact-2"}, Text: "anything"}); err == nil {
		t.Fatal("Handle returned nil error after reset with empty setup code")
	}
}

func TestManagerConcurrentValidMessagesOnlyBindOneContact(t *testing.T) {
	manager := newTestManager(t, "pair-123")
	const attempts = 20
	start := make(chan struct{})
	results := make(chan Result, attempts)
	errs := make(chan error, attempts)
	var wg sync.WaitGroup

	for i := range attempts {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			result, err := manager.Handle(Message{
				From: Contact{ID: "contact-" + string(rune('a'+i))},
				Text: "pair-123",
			})
			results <- result
			errs <- err
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("Handle returned error: %v", err)
		}
	}

	boundCount := 0
	var bound Contact
	for result := range results {
		if result.Outcome == OutcomeBound {
			boundCount++
			bound = result.Contact
		}
	}
	if boundCount != 1 {
		t.Fatalf("bound outcomes = %d, want 1", boundCount)
	}
	stored, ok := manager.BoundContact()
	if !ok {
		t.Fatal("manager has no bound contact")
	}
	if stored != bound {
		t.Fatalf("stored contact = %#v, want bound result %#v", stored, bound)
	}
}

func TestManagerRequiresSetupCodeWhenUnbound(t *testing.T) {
	_, err := NewManager("", newTestStore(t))
	if err == nil {
		t.Fatal("NewManager returned nil error, want setup code error")
	}
}

func TestManagerRejectsPersistedBindingWithoutContactID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "binding.json")
	if err := os.WriteFile(path, []byte(`{"contact":{}}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := NewManager("pair-123", NewFileStore(path))
	if err == nil {
		t.Fatal("NewManager returned nil error, want invalid binding error")
	}
}

func newTestManager(t *testing.T, code string) *Manager {
	t.Helper()
	manager, err := NewManager(code, newTestStore(t))
	if err != nil {
		t.Fatalf("NewManager returned error: %v", err)
	}
	return manager
}

func newTestStore(t *testing.T) *FileStore {
	t.Helper()
	return NewFileStore(filepath.Join(t.TempDir(), "binding.json"))
}
