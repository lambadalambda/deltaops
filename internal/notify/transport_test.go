package notify

import (
	"context"
	"errors"
	"testing"
)

func TestTransportContractFake(t *testing.T) {
	ctx := context.Background()
	fake := &fakeTransport{
		account: Account{
			ContactURI: "OPENPGP4FPR:bot@example.test#setup",
			Address:    "bot@example.test",
		},
		incoming: []IncomingMessage{
			{
				From: Contact{ID: "contact-1", Address: "operator@example.test", DisplayName: "Operator"},
				Text: "pair 123456",
			},
		},
	}
	var transport Transport = fake

	account, err := transport.Account(ctx)
	if err != nil {
		t.Fatalf("Account returned error: %v", err)
	}
	if account.ContactURI == "" {
		t.Fatal("Account returned empty contact URI")
	}
	if account.Address == "" {
		t.Fatal("Account returned empty address")
	}

	message, err := transport.Receive(ctx)
	if err != nil {
		t.Fatalf("Receive returned error: %v", err)
	}
	if message.From.ID == "" {
		t.Fatal("Receive returned message without contact ID")
	}
	if message.Text == "" {
		t.Fatal("Receive returned message without text")
	}

	err = transport.Send(ctx, message.From, OutgoingMessage{Text: "disk critical on host"})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if len(fake.sent) != 1 {
		t.Fatalf("Send recorded %d messages, want 1", len(fake.sent))
	}
	if fake.sent[0].to != message.From {
		t.Fatalf("Send recipient = %#v, want %#v", fake.sent[0].to, message.From)
	}
	if fake.sent[0].message.Text != "disk critical on host" {
		t.Fatalf("Send text = %q, want %q", fake.sent[0].message.Text, "disk critical on host")
	}
}

type fakeTransport struct {
	account  Account
	incoming []IncomingMessage
	sent     []sentMessage
}

func (f *fakeTransport) Account(context.Context) (Account, error) {
	return f.account, nil
}

func (f *fakeTransport) Receive(context.Context) (IncomingMessage, error) {
	if len(f.incoming) == 0 {
		return IncomingMessage{}, errors.New("no incoming messages")
	}
	message := f.incoming[0]
	f.incoming = f.incoming[1:]
	return message, nil
}

func (f *fakeTransport) Send(_ context.Context, to Contact, message OutgoingMessage) error {
	f.sent = append(f.sent, sentMessage{to: to, message: message})
	return nil
}

type sentMessage struct {
	to      Contact
	message OutgoingMessage
}
