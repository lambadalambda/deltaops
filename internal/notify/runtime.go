package notify

import (
	"context"

	"deltaops/internal/alert"
	"deltaops/internal/binding"
)

type ReadyTransport interface {
	Ready(context.Context) error
}

type RuntimeAccount struct {
	Transport ReadyTransport
}

type RuntimePairer struct {
	Manager   *binding.Manager
	Transport Transport
}

type RuntimeNotifier struct {
	Transport Transport
}

func (a RuntimeAccount) Ready(ctx context.Context) error {
	return a.Transport.Ready(ctx)
}

func (p RuntimePairer) BoundContact() (binding.Contact, bool) {
	return p.Manager.BoundContact()
}

func (p RuntimePairer) WaitForPairing(ctx context.Context) (binding.Contact, error) {
	for {
		message, err := p.Transport.Receive(ctx)
		if err != nil {
			return binding.Contact{}, err
		}
		result, err := p.Manager.Handle(binding.Message{
			From: binding.Contact{ID: message.From.ID, Address: message.From.Address, DisplayName: message.From.DisplayName},
			Text: message.Text,
		})
		if err != nil {
			return binding.Contact{}, err
		}
		if result.Outcome == binding.OutcomeBound || result.Outcome == binding.OutcomeAlreadyBound {
			return result.Contact, nil
		}
	}
}

func (n RuntimeNotifier) Notify(ctx context.Context, contact binding.Contact, decision alert.Decision) error {
	return n.Transport.Send(ctx, Contact{ID: contact.ID, Address: contact.Address, DisplayName: contact.DisplayName}, OutgoingMessage{Text: decision.Message()})
}
