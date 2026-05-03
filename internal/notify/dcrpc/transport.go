package dcrpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"deltaops/internal/notify"

	chat "github.com/chatmail/rpc-client-go/v2/deltachat"
)

type Core interface {
	GetAllAccountIds(context.Context) ([]uint32, error)
	AddAccount(context.Context) (uint32, error)
	SelectAccount(context.Context, uint32) error
	IsConfigured(context.Context, uint32) (bool, error)
	SetConfig(context.Context, uint32, string, *string) error
	AddTransportFromQr(context.Context, uint32, string) error
	StartIo(context.Context, uint32) error
	StopIo(context.Context, uint32) error
	GetConfig(context.Context, uint32, string) (*string, error)
	GetChatSecurejoinQrCode(context.Context, uint32, *uint32) (string, error)
	GetNextEvent(context.Context) (chat.Event, error)
	GetMessage(context.Context, uint32, uint32) (chat.Message, error)
	MarkseenMsgs(context.Context, uint32, []uint32) error
	GetChatIdByContactId(context.Context, uint32, uint32) (*uint32, error)
	CreateChatByContactId(context.Context, uint32, uint32) (uint32, error)
	SendMsg(context.Context, uint32, uint32, chat.MessageData) (uint32, error)
}

type RPCTransport interface {
	chat.RpcTransport
	Close()
}

type Options struct {
	GOOS         string
	GOARCH       string
	AccountsDir  string
	HelperDir    string
	DCAccountURL string
	Stderr       io.Writer
	Assets       []HelperAsset
	OpenRPC      func(cmd, accountsDir string, stderr io.Writer) (RPCTransport, error)
}

type Transport struct {
	mu        sync.Mutex
	core      Core
	accountID uint32
	close     func()
	started   bool
	closed    bool
}

const stopIOTimeout = 5 * time.Second

type rpcCore struct {
	transport chat.RpcTransport
}

func Open(ctx context.Context, options Options) (*Transport, error) {
	asset, err := SelectHelper(options.Assets, options.GOOS, options.GOARCH)
	if err != nil {
		return nil, err
	}
	helperPath, err := ExtractHelper(asset, options.HelperDir)
	if err != nil {
		return nil, err
	}
	openRPC := options.OpenRPC
	if openRPC == nil {
		openRPC = openIOTransport
	}
	rpcTransport, err := openRPC(helperPath, options.AccountsDir, options.Stderr)
	if err != nil {
		return nil, fmt.Errorf("open RPC helper: %w", err)
	}
	core := rpcCore{transport: rpcTransport}
	accountID, err := EnsureAccount(ctx, core, options.DCAccountURL)
	if err != nil {
		rpcTransport.Close()
		return nil, err
	}
	return NewTransport(core, accountID, rpcTransport.Close), nil
}

func NewTransport(core Core, accountID uint32, close func()) *Transport {
	return &Transport{core: core, accountID: accountID, close: close}
}

func EnsureAccount(ctx context.Context, core Core, dcAccountURL string) (uint32, error) {
	accountIDs, err := core.GetAllAccountIds(ctx)
	if err != nil {
		return 0, fmt.Errorf("list Delta Chat accounts: %w", err)
	}
	var accountID uint32
	if len(accountIDs) == 0 {
		accountID, err = core.AddAccount(ctx)
		if err != nil {
			return 0, fmt.Errorf("create Delta Chat account: %w", err)
		}
	} else {
		accountID = accountIDs[0]
	}
	if err := core.SelectAccount(ctx, accountID); err != nil {
		return 0, fmt.Errorf("select Delta Chat account: %w", err)
	}

	configured, err := core.IsConfigured(ctx, accountID)
	if err != nil {
		return 0, fmt.Errorf("check Delta Chat account configuration: %w", err)
	}
	if !configured {
		botFlag := "1"
		if err := core.SetConfig(ctx, accountID, "bot", &botFlag); err != nil {
			return 0, fmt.Errorf("set Delta Chat bot mode: %w", err)
		}
		if err := core.AddTransportFromQr(ctx, accountID, strings.TrimSpace(dcAccountURL)); err != nil {
			return 0, fmt.Errorf("configure Delta Chat transport from provisioning URL: %w", err)
		}
	}
	return accountID, nil
}

func (t *Transport) Ready(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	configured, err := t.core.IsConfigured(ctx, t.accountID)
	if err != nil {
		return fmt.Errorf("check Delta Chat account configuration: %w", err)
	}
	if !configured {
		return errors.New("Delta Chat account is not configured")
	}
	if t.started {
		return nil
	}
	if err := t.core.StartIo(ctx, t.accountID); err != nil {
		return fmt.Errorf("start Delta Chat I/O: %w", err)
	}
	t.started = true
	return nil
}

func (t *Transport) Account(ctx context.Context) (notify.Account, error) {
	uri, err := t.core.GetChatSecurejoinQrCode(ctx, t.accountID, nil)
	if err != nil {
		return notify.Account{}, fmt.Errorf("get Delta Chat contact URI: %w", err)
	}
	addr, err := t.core.GetConfig(ctx, t.accountID, "addr")
	if err != nil {
		return notify.Account{}, fmt.Errorf("get Delta Chat address: %w", err)
	}
	account := notify.Account{ContactURI: uri}
	if addr != nil {
		account.Address = *addr
	}
	return account, nil
}

func (t *Transport) Receive(ctx context.Context) (notify.IncomingMessage, error) {
	for {
		event, err := t.core.GetNextEvent(ctx)
		if err != nil {
			return notify.IncomingMessage{}, err
		}
		if event.ContextId != t.accountID {
			continue
		}
		incoming, ok := event.Event.(*chat.EventTypeIncomingMsg)
		if !ok {
			continue
		}
		message, err := t.core.GetMessage(ctx, t.accountID, incoming.MsgId)
		if err != nil {
			return notify.IncomingMessage{}, fmt.Errorf("get incoming Delta Chat message: %w", err)
		}
		if message.FromId <= chat.ContactLastSpecial {
			continue
		}
		_ = t.core.MarkseenMsgs(ctx, t.accountID, []uint32{incoming.MsgId})
		contact := message.Sender
		if contact.Id == 0 {
			contact.Id = message.FromId
		}
		return notify.IncomingMessage{
			From: contactFromDelta(contact),
			Text: message.Text,
		}, nil
	}
}

func (t *Transport) Send(ctx context.Context, to notify.Contact, message notify.OutgoingMessage) error {
	contactID, err := parseContactID(to.ID)
	if err != nil {
		return err
	}
	chatID, err := t.core.GetChatIdByContactId(ctx, t.accountID, contactID)
	if err != nil {
		return fmt.Errorf("get Delta Chat recipient chat: %w", err)
	}
	var targetChatID uint32
	if chatID == nil {
		targetChatID, err = t.core.CreateChatByContactId(ctx, t.accountID, contactID)
		if err != nil {
			return fmt.Errorf("create Delta Chat recipient chat: %w", err)
		}
	} else {
		targetChatID = *chatID
	}
	text := message.Text
	if _, err := t.core.SendMsg(ctx, t.accountID, targetChatID, chat.MessageData{Text: &text}); err != nil {
		return fmt.Errorf("send Delta Chat message: %w", err)
	}
	return nil
}

func (t *Transport) Close() error {
	if t == nil || t.close == nil {
		return nil
	}
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	started := t.started
	t.started = false
	closeFn := t.close
	t.mu.Unlock()
	defer closeFn()
	if started {
		ctx, cancel := context.WithTimeout(context.Background(), stopIOTimeout)
		defer cancel()
		_ = t.core.StopIo(ctx, t.accountID)
	}
	return nil
}

func contactFromDelta(contact chat.Contact) notify.Contact {
	return notify.Contact{
		ID:          strconv.FormatUint(uint64(contact.Id), 10),
		Address:     contact.Address,
		DisplayName: firstNonEmpty(contact.DisplayName, contact.Name),
	}
}

func parseContactID(id string) (uint32, error) {
	parsed, err := strconv.ParseUint(strings.TrimSpace(id), 10, 32)
	if err != nil || parsed == 0 {
		return 0, fmt.Errorf("invalid Delta Chat contact ID %q", id)
	}
	return uint32(parsed), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func openIOTransport(cmd, accountsDir string, stderr io.Writer) (RPCTransport, error) {
	transport := chat.NewIOTransport()
	transport.Cmd = cmd
	transport.AccountsDir = accountsDir
	if stderr != nil {
		transport.Stderr = stderr
	} else {
		transport.Stderr = io.Discard
	}
	if err := transport.Open(); err != nil {
		return nil, err
	}
	return transport, nil
}

func (c rpcCore) rpc(ctx context.Context) *chat.Rpc {
	return &chat.Rpc{Context: ctx, Transport: c.transport}
}

func (c rpcCore) GetAllAccountIds(ctx context.Context) ([]uint32, error) {
	return c.rpc(ctx).GetAllAccountIds()
}
func (c rpcCore) AddAccount(ctx context.Context) (uint32, error) { return c.rpc(ctx).AddAccount() }
func (c rpcCore) SelectAccount(ctx context.Context, id uint32) error {
	return c.rpc(ctx).SelectAccount(id)
}
func (c rpcCore) IsConfigured(ctx context.Context, id uint32) (bool, error) {
	return c.rpc(ctx).IsConfigured(id)
}
func (c rpcCore) SetConfig(ctx context.Context, id uint32, key string, value *string) error {
	return c.rpc(ctx).SetConfig(id, key, value)
}
func (c rpcCore) AddTransportFromQr(ctx context.Context, id uint32, qr string) error {
	return c.rpc(ctx).AddTransportFromQr(id, qr)
}
func (c rpcCore) StartIo(ctx context.Context, id uint32) error { return c.rpc(ctx).StartIo(id) }
func (c rpcCore) StopIo(ctx context.Context, id uint32) error  { return c.rpc(ctx).StopIo(id) }
func (c rpcCore) GetConfig(ctx context.Context, id uint32, key string) (*string, error) {
	return c.rpc(ctx).GetConfig(id, key)
}
func (c rpcCore) GetChatSecurejoinQrCode(ctx context.Context, id uint32, chatID *uint32) (string, error) {
	return c.rpc(ctx).GetChatSecurejoinQrCode(id, chatID)
}
func (c rpcCore) GetNextEvent(ctx context.Context) (chat.Event, error) {
	return c.rpc(ctx).GetNextEvent()
}
func (c rpcCore) GetMessage(ctx context.Context, accountID, msgID uint32) (chat.Message, error) {
	return c.rpc(ctx).GetMessage(accountID, msgID)
}
func (c rpcCore) MarkseenMsgs(ctx context.Context, accountID uint32, msgIDs []uint32) error {
	return c.rpc(ctx).MarkseenMsgs(accountID, msgIDs)
}
func (c rpcCore) GetChatIdByContactId(ctx context.Context, accountID, contactID uint32) (*uint32, error) {
	return c.rpc(ctx).GetChatIdByContactId(accountID, contactID)
}
func (c rpcCore) CreateChatByContactId(ctx context.Context, accountID, contactID uint32) (uint32, error) {
	return c.rpc(ctx).CreateChatByContactId(accountID, contactID)
}
func (c rpcCore) SendMsg(ctx context.Context, accountID, chatID uint32, data chat.MessageData) (uint32, error) {
	return c.rpc(ctx).SendMsg(accountID, chatID, data)
}
