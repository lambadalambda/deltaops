package binding

import (
	"errors"
	"strings"
	"sync"
)

type Contact struct {
	ID          string `json:"id"`
	Address     string `json:"address,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

type Message struct {
	From Contact
	Text string
}

type Binding struct {
	Contact Contact `json:"contact"`
}

type Outcome string

const (
	OutcomeIgnored      Outcome = "ignored"
	OutcomeBound        Outcome = "bound"
	OutcomeAlreadyBound Outcome = "already_bound"
)

type Result struct {
	Outcome Outcome
	Contact Contact
}

type Store interface {
	Load() (Binding, bool, error)
	Save(Binding) error
	Delete() error
}

type Manager struct {
	mu        sync.Mutex
	setupCode string
	store     Store
	binding   *Binding
}

func NewManager(setupCode string, store Store) (*Manager, error) {
	if store == nil {
		return nil, errors.New("binding store is required")
	}
	binding, ok, err := store.Load()
	if err != nil {
		return nil, err
	}
	if ok {
		if err := validateBinding(binding); err != nil {
			return nil, err
		}
		return &Manager{setupCode: strings.TrimSpace(setupCode), store: store, binding: &binding}, nil
	}

	setupCode = strings.TrimSpace(setupCode)
	if setupCode == "" {
		return nil, errors.New("setup code is required when no contact is bound")
	}
	return &Manager{setupCode: setupCode, store: store}, nil
}

func (m *Manager) Handle(message Message) (Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.binding != nil {
		return Result{Outcome: OutcomeAlreadyBound, Contact: m.binding.Contact}, nil
	}
	if m.setupCode == "" {
		return Result{}, errors.New("setup code is required when no contact is bound")
	}
	if !strings.Contains(message.Text, m.setupCode) {
		return Result{Outcome: OutcomeIgnored}, nil
	}
	if err := validateContact(message.From); err != nil {
		return Result{}, err
	}

	binding := Binding{Contact: message.From}
	if err := m.store.Save(binding); err != nil {
		return Result{}, err
	}
	m.binding = &binding
	return Result{Outcome: OutcomeBound, Contact: binding.Contact}, nil
}

func (m *Manager) BoundContact() (Contact, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.binding == nil {
		return Contact{}, false
	}
	return m.binding.Contact, true
}

func (m *Manager) Reset() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.store.Delete(); err != nil {
		return err
	}
	m.binding = nil
	m.setupCode = ""
	return nil
}

func validateBinding(binding Binding) error {
	if err := validateContact(binding.Contact); err != nil {
		return err
	}
	return nil
}

func validateContact(contact Contact) error {
	if strings.TrimSpace(contact.ID) == "" {
		return errors.New("contact ID is required to bind alerts")
	}
	return nil
}
