package dcrpc

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"deltaops/internal/notify"

	chat "github.com/chatmail/rpc-client-go/v2/deltachat"
)

func TestEnsureAccountCreatesBotFromProvisioningURL(t *testing.T) {
	core := &fakeCore{}
	accountID, err := EnsureAccount(context.Background(), core, "dcaccount:secret")
	if err != nil {
		t.Fatalf("EnsureAccount returned error: %v", err)
	}
	if accountID != 1 {
		t.Fatalf("accountID = %d, want 1", accountID)
	}
	if core.addTransportQR != "dcaccount:secret" {
		t.Fatalf("AddTransportFromQr input = %q", core.addTransportQR)
	}
	if core.config["bot"] == nil || *core.config["bot"] != "1" {
		t.Fatalf("bot config = %#v, want 1", core.config["bot"])
	}
	if core.startCount != 0 {
		t.Fatalf("EnsureAccount started I/O %d times, want 0", core.startCount)
	}
}

func TestEnsureAccountReusesConfiguredAccountWithoutProvisioning(t *testing.T) {
	core := &fakeCore{accountIDs: []uint32{7}, configured: true}
	accountID, err := EnsureAccount(context.Background(), core, "dcaccount:secret")
	if err != nil {
		t.Fatalf("EnsureAccount returned error: %v", err)
	}
	if accountID != 7 {
		t.Fatalf("accountID = %d, want 7", accountID)
	}
	if core.addTransportQR != "" {
		t.Fatalf("AddTransportFromQr called with %q for configured account", core.addTransportQR)
	}
}

func TestTransportAccountReturnsContactData(t *testing.T) {
	addr := "bot@example.test"
	core := &fakeCore{secureJoin: "OPENPGP4FPR:bot#setup", config: map[string]*string{"addr": &addr}}
	transport := NewTransport(core, 1, nil)

	account, err := transport.Account(context.Background())
	if err != nil {
		t.Fatalf("Account returned error: %v", err)
	}
	if account.ContactURI != "OPENPGP4FPR:bot#setup" || account.Address != addr {
		t.Fatalf("account = %#v", account)
	}
}

func TestTransportReceiveReturnsIncomingMessage(t *testing.T) {
	core := &fakeCore{
		events: []chat.Event{{ContextId: 1, Event: &chat.EventTypeIncomingMsg{MsgId: 42, ChatId: 3}}},
		messages: map[uint32]chat.Message{42: {
			FromId: 11,
			Text:   "pair 123456",
			Sender: chat.Contact{Id: 11, Address: "operator@example.test", DisplayName: "Operator"},
		}},
	}
	transport := NewTransport(core, 1, nil)

	message, err := transport.Receive(context.Background())
	if err != nil {
		t.Fatalf("Receive returned error: %v", err)
	}
	if message.Text != "pair 123456" {
		t.Fatalf("Text = %q", message.Text)
	}
	if message.From.ID != "11" || message.From.Address != "operator@example.test" || message.From.DisplayName != "Operator" {
		t.Fatalf("From = %#v", message.From)
	}
	if len(core.markedSeen) != 1 || core.markedSeen[0] != 42 {
		t.Fatalf("markedSeen = %v, want [42]", core.markedSeen)
	}
}

func TestTransportReceiveSkipsSpecialContactsAndOtherAccounts(t *testing.T) {
	core := &fakeCore{
		events: []chat.Event{
			{ContextId: 2, Event: &chat.EventTypeIncomingMsg{MsgId: 1}},
			{ContextId: 1, Event: &chat.EventTypeIncomingMsg{MsgId: 2}},
			{ContextId: 1, Event: &chat.EventTypeIncomingMsg{MsgId: 3}},
		},
		messages: map[uint32]chat.Message{
			2: {FromId: chat.ContactInfo, Text: "system", Sender: chat.Contact{Id: chat.ContactInfo}},
			3: {FromId: 12, Text: "pair", Sender: chat.Contact{Id: 12}},
		},
	}
	transport := NewTransport(core, 1, nil)

	message, err := transport.Receive(context.Background())
	if err != nil {
		t.Fatalf("Receive returned error: %v", err)
	}
	if message.From.ID != "12" {
		t.Fatalf("From.ID = %q, want 12", message.From.ID)
	}
}

func TestTransportSendUsesExistingOrCreatedChat(t *testing.T) {
	core := &fakeCore{chatIDs: map[uint32]*uint32{11: uint32Ptr(44)}}
	transport := NewTransport(core, 1, nil)

	if err := transport.Send(context.Background(), notify.Contact{ID: "11"}, notify.OutgoingMessage{Text: "alert"}); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if core.sentChatID != 44 || core.sentText != "alert" {
		t.Fatalf("sent chat/text = %d/%q", core.sentChatID, core.sentText)
	}

	core.chatIDs[12] = nil
	if err := transport.Send(context.Background(), notify.Contact{ID: "12"}, notify.OutgoingMessage{Text: "second"}); err != nil {
		t.Fatalf("Send without chat returned error: %v", err)
	}
	if core.createdChatContact != 12 || core.sentChatID != 99 || core.sentText != "second" {
		t.Fatalf("created/sent = %d/%d/%q", core.createdChatContact, core.sentChatID, core.sentText)
	}
}

func TestTransportRejectsInvalidContactID(t *testing.T) {
	transport := NewTransport(&fakeCore{}, 1, nil)
	err := transport.Send(context.Background(), notify.Contact{ID: "not-a-number"}, notify.OutgoingMessage{Text: "alert"})
	if err == nil || !strings.Contains(err.Error(), "contact ID") {
		t.Fatalf("Send error = %v, want contact ID error", err)
	}
}

func TestReadyRequiresConfiguredAccount(t *testing.T) {
	transport := NewTransport(&fakeCore{configured: false}, 1, nil)
	err := transport.Ready(context.Background())
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("Ready error = %v, want not configured", err)
	}
}

func TestReadyStartsIOOnceAndCloseStopsIt(t *testing.T) {
	core := &fakeCore{configured: true}
	closeCount := 0
	transport := NewTransport(core, 1, func() { closeCount++ })
	if err := transport.Ready(context.Background()); err != nil {
		t.Fatalf("Ready returned error: %v", err)
	}
	if err := transport.Ready(context.Background()); err != nil {
		t.Fatalf("second Ready returned error: %v", err)
	}
	if core.startCount != 1 {
		t.Fatalf("startCount = %d, want 1", core.startCount)
	}
	if err := transport.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := transport.Close(); err != nil {
		t.Fatalf("second Close returned error: %v", err)
	}
	if core.stopCount != 1 || closeCount != 1 {
		t.Fatalf("stopCount/closeCount = %d/%d, want 1/1", core.stopCount, closeCount)
	}
	if !core.stopHadDeadline {
		t.Fatal("StopIo context did not have a deadline")
	}
}

func TestOpenWiresHelperAndClosesOnProvisioningFailure(t *testing.T) {
	helpers := t.TempDir()
	accounts := filepath.Join(t.TempDir(), "accounts")
	rpcTransport := &fakeRPCTransport{failAddTransport: true}
	var gotCmd, gotAccounts string
	var gotStderr io.Writer

	_, err := Open(context.Background(), Options{
		GOOS:         "linux",
		GOARCH:       "amd64",
		AccountsDir:  accounts,
		HelperDir:    helpers,
		DCAccountURL: "dcaccount:secret",
		Stderr:       io.Discard,
		Assets:       []HelperAsset{{Filename: "deltachat-rpc-server-x86_64-linux", Data: []byte("helper")}},
		OpenRPC: func(cmd, accountsDir string, stderr io.Writer) (RPCTransport, error) {
			gotCmd = cmd
			gotAccounts = accountsDir
			gotStderr = stderr
			return rpcTransport, nil
		},
	})
	if err == nil {
		t.Fatal("Open returned nil error, want provisioning failure")
	}
	if !rpcTransport.closed {
		t.Fatal("Open did not close RPC transport after provisioning failure")
	}
	if gotAccounts != accounts || gotStderr != io.Discard {
		t.Fatalf("OpenRPC accounts/stderr = %q/%#v", gotAccounts, gotStderr)
	}
	if gotCmd == "" || filepath.Base(gotCmd) != "deltachat-rpc-server-x86_64-linux" {
		t.Fatalf("OpenRPC cmd = %q", gotCmd)
	}
	if _, err := os.Stat(gotCmd); err != nil {
		t.Fatalf("helper path was not extracted: %v", err)
	}
}

type fakeCore struct {
	accountIDs         []uint32
	configured         bool
	config             map[string]*string
	addTransportQR     string
	startCount         int
	stopCount          int
	stopHadDeadline    bool
	secureJoin         string
	events             []chat.Event
	messages           map[uint32]chat.Message
	markedSeen         []uint32
	chatIDs            map[uint32]*uint32
	createdChatContact uint32
	sentChatID         uint32
	sentText           string
}

func (f *fakeCore) GetAllAccountIds(context.Context) ([]uint32, error) {
	return f.accountIDs, nil
}

func (f *fakeCore) AddAccount(context.Context) (uint32, error) {
	f.accountIDs = append(f.accountIDs, 1)
	return 1, nil
}

func (f *fakeCore) SelectAccount(context.Context, uint32) error { return nil }

func (f *fakeCore) IsConfigured(context.Context, uint32) (bool, error) {
	return f.configured, nil
}

func (f *fakeCore) SetConfig(_ context.Context, _ uint32, key string, value *string) error {
	if f.config == nil {
		f.config = make(map[string]*string)
	}
	f.config[key] = value
	return nil
}

func (f *fakeCore) AddTransportFromQr(_ context.Context, _ uint32, qr string) error {
	f.addTransportQR = qr
	f.configured = true
	return nil
}

func (f *fakeCore) StartIo(context.Context, uint32) error {
	f.startCount++
	return nil
}

func (f *fakeCore) StopIo(ctx context.Context, _ uint32) error {
	f.stopCount++
	_, f.stopHadDeadline = ctx.Deadline()
	return nil
}

func (f *fakeCore) GetConfig(_ context.Context, _ uint32, key string) (*string, error) {
	if f.config == nil {
		return nil, nil
	}
	return f.config[key], nil
}

func (f *fakeCore) GetChatSecurejoinQrCode(context.Context, uint32, *uint32) (string, error) {
	return f.secureJoin, nil
}

func (f *fakeCore) GetNextEvent(context.Context) (chat.Event, error) {
	if len(f.events) == 0 {
		return chat.Event{}, errors.New("no events")
	}
	event := f.events[0]
	f.events = f.events[1:]
	return event, nil
}

func (f *fakeCore) GetMessage(_ context.Context, _ uint32, msgID uint32) (chat.Message, error) {
	message, ok := f.messages[msgID]
	if !ok {
		return chat.Message{}, errors.New("missing message")
	}
	return message, nil
}

func (f *fakeCore) MarkseenMsgs(_ context.Context, _ uint32, msgIDs []uint32) error {
	f.markedSeen = append(f.markedSeen, msgIDs...)
	return nil
}

func (f *fakeCore) GetChatIdByContactId(_ context.Context, _ uint32, contactID uint32) (*uint32, error) {
	if f.chatIDs == nil {
		return nil, nil
	}
	return f.chatIDs[contactID], nil
}

func (f *fakeCore) CreateChatByContactId(_ context.Context, _ uint32, contactID uint32) (uint32, error) {
	f.createdChatContact = contactID
	return 99, nil
}

func (f *fakeCore) SendMsg(_ context.Context, _ uint32, chatID uint32, data chat.MessageData) (uint32, error) {
	f.sentChatID = chatID
	if data.Text != nil {
		f.sentText = *data.Text
	}
	return 100, nil
}

func uint32Ptr(value uint32) *uint32 { return &value }

type fakeRPCTransport struct {
	failAddTransport bool
	closed           bool
}

func (f *fakeRPCTransport) Call(_ context.Context, method string, params ...any) error {
	switch method {
	case "select_account", "set_config":
		return nil
	case "add_transport_from_qr":
		if f.failAddTransport {
			return errors.New("configure failed dcaccount:secret")
		}
		return nil
	default:
		return nil
	}
}

func (f *fakeRPCTransport) CallResult(_ context.Context, result any, method string, params ...any) error {
	switch method {
	case "get_all_account_ids":
		*(result.(*[]uint32)) = nil
	case "add_account":
		*(result.(*uint32)) = 1
	case "is_configured":
		*(result.(*bool)) = false
	}
	return nil
}

func (f *fakeRPCTransport) Close() { f.closed = true }
