package mqtt

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
)

type fakeToken struct {
	timeout bool
	error   bool
	result  map[string]byte
}

func (t *fakeToken) Result() map[string]byte {
	return t.result
}

func (t *fakeToken) Wait() bool {
	return false
}

func (t *fakeToken) WaitTimeout(time.Duration) bool {
	return !t.timeout
}

func (t *fakeToken) Done() <-chan struct{} {
	return nil
}

func (t *fakeToken) Error() error {
	if t.error {
		return errors.New("error")
	}
	return nil
}

type fakeClient struct {
	connected             bool
	connectionOpen        bool
	tokenTimeout          bool
	tokenError            bool
	subscribeResult       map[string]byte
	disconnectInvocations int
}

func (c *fakeClient) IsConnected() bool {
	return c.connected
}

func (c *fakeClient) IsConnectionOpen() bool {
	return c.connectionOpen
}

func (c *fakeClient) Connect() pahomqtt.Token {
	return nil
}

func (c *fakeClient) Disconnect(uint) {
	c.disconnectInvocations++
}

func (c *fakeClient) Publish(string, byte, bool, interface{}) pahomqtt.Token {
	return nil
}

func (c *fakeClient) Subscribe(string, byte, pahomqtt.MessageHandler) pahomqtt.Token {
	return &fakeToken{timeout: c.tokenTimeout, error: c.tokenError, result: c.subscribeResult}
}

func (c *fakeClient) SubscribeMultiple(map[string]byte, pahomqtt.MessageHandler) pahomqtt.Token {
	return nil
}

func (c *fakeClient) Unsubscribe(...string) pahomqtt.Token {
	return nil
}

func (c *fakeClient) AddRoute(string, pahomqtt.MessageHandler) {
}

func (c *fakeClient) OptionsReader() pahomqtt.ClientOptionsReader {
	return pahomqtt.ClientOptionsReader{}
}

func Test_listener_Close(t *testing.T) {
	type fields struct {
		c *fakeClient
	}
	tests := []struct {
		name                      string
		fields                    fields
		wantDisconnectInvocations int
	}{
		{
			name: "Closing connected listener",
			fields: fields{
				c: &fakeClient{connected: true},
			},
			wantDisconnectInvocations: 1,
		},
		{
			name: "Closing disconnected listener",
			fields: fields{
				c: &fakeClient{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &listener{
				c: tt.fields.c,
			}
			l.Close()
			if tt.fields.c.disconnectInvocations != tt.wantDisconnectInvocations {
				t.Errorf("Close() invocations = %v, wanted %v", tt.fields.c.disconnectInvocations, tt.wantDisconnectInvocations)
			}
		})
	}
}

func Test_listener_Subscribe(t *testing.T) {
	type fields struct {
		c pahomqtt.Client
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Subscribe topic",
			fields: fields{
				c: &fakeClient{},
			},
		},
		{
			name: "Subscription timed out",
			fields: fields{
				c: &fakeClient{tokenTimeout: true},
			},
			wantErr: true,
		},
		{
			name: "Subscription granted",
			fields: fields{
				c: &fakeClient{subscribeResult: map[string]byte{"topic": 0}},
			},
		},
		{
			name: "Subscription refused by the broker",
			fields: fields{
				c: &fakeClient{subscribeResult: map[string]byte{"topic": subscribeFailureCode}},
			},
			wantErr: true,
		},
		{
			name: "Subscription errored",
			fields: fields{
				c: &fakeClient{tokenError: true},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		mh := func(pahomqtt.Client, pahomqtt.Message) {}
		t.Run(tt.name, func(t *testing.T) {
			l := &listener{
				c: tt.fields.c,
			}
			if err := l.Subscribe("topic", mh); (err != nil) != tt.wantErr {
				t.Errorf("Subscribe() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_listener_Check(t *testing.T) {
	type fields struct {
		c pahomqtt.Client
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Client check is OK",
			fields: fields{
				c: &fakeClient{connectionOpen: true},
			},
			wantErr: false,
		},
		{
			name: "Client check failing",
			fields: fields{
				c: &fakeClient{connectionOpen: false},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &listener{
				c: tt.fields.c,
			}
			if err := l.Check(context.Background()); (err != nil) != tt.wantErr {
				t.Errorf("Check() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_connectionLostHandler(t *testing.T) {
	lost := make(chan error, 1)
	l := &listener{lost: lost}
	handler := newConnectionLostHandler(lost)

	select {
	case err := <-l.ConnectionLost():
		t.Fatalf("ConnectionLost() = %v before the connection was lost", err)
	default:
	}

	wantErr := errors.New("EOF")
	handler(&fakeClient{}, wantErr)
	// a second notification must not block the client goroutine
	handler(&fakeClient{}, errors.New("another"))

	select {
	case err := <-l.ConnectionLost():
		if !errors.Is(err, wantErr) {
			t.Errorf("ConnectionLost() = %v, want %v", err, wantErr)
		}
	case <-time.After(time.Second):
		t.Fatal("ConnectionLost() did not report the lost connection")
	}
}

func Test_connectionLostHandler_NilError(t *testing.T) {
	lost := make(chan error, 1)
	newConnectionLostHandler(lost)(&fakeClient{}, nil)
	if err := <-lost; err == nil {
		t.Error("expected a non nil error")
	}
}

func TestListenerOptions(t *testing.T) {
	opts := pahomqtt.NewClientOptions()
	for _, o := range []ListenerOption{
		WithTimeout(4 * time.Second),
		WithKeepAlive(45 * time.Second),
		WithPingTimeout(12 * time.Second),
	} {
		o(opts)
	}
	if opts.ConnectTimeout != 4*time.Second {
		t.Errorf("ConnectTimeout = %v, want 4s", opts.ConnectTimeout)
	}
	if opts.KeepAlive != 45 {
		t.Errorf("KeepAlive = %v, want 45", opts.KeepAlive)
	}
	if opts.PingTimeout != 12*time.Second {
		t.Errorf("PingTimeout = %v, want 12s", opts.PingTimeout)
	}
}

func Test_newClientID(t *testing.T) {
	// MQTT 3.1: at most 23 characters; MQTT 3.1.1 and 5.0: brokers must accept 1-23 bytes of 0-9a-zA-Z.
	allowed := regexp.MustCompile(`^[0-9a-zA-Z]{1,23}$`)
	seen := make(map[string]struct{})
	for range 1000 {
		id := newClientID()
		if !allowed.MatchString(id) {
			t.Fatalf("newClientID() = %q, want 1 to 23 characters among 0-9a-zA-Z", id)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("newClientID() returned %q twice", id)
		}
		seen[id] = struct{}{}
	}
}
