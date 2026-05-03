package notify

import "context"

// Transport is the boundary between DeltaOps core behavior and Delta Chat.
type Transport interface {
	// Account returns the configured bot identity and contact data to show during setup.
	Account(context.Context) (Account, error)
	// Receive blocks until the next inbound message or context cancellation.
	Receive(context.Context) (IncomingMessage, error)
	// Send delivers a message to a previously observed or persisted contact.
	Send(context.Context, Contact, OutgoingMessage) error
}

type Account struct {
	ContactURI string
	Address    string
}

type Contact struct {
	// ID is the transport-specific stable identifier and should be treated as opaque.
	ID          string
	Address     string
	DisplayName string
}

type IncomingMessage struct {
	From Contact
	Text string
}

type OutgoingMessage struct {
	Text string
}
