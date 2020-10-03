package mqtt

import (
	"errors"
	"testing"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
)

type fakeToken struct {
	timeout bool
	error   bool
}

func (t *fakeToken) Wait() bool {
	return false
}

func (t *fakeToken) WaitTimeout(time.Duration) bool {
	return !t.timeout
}

func (t *fakeToken) Error() error {
	if t.error {
		return errors.New("error")
	}
	return nil
}

type fakeClient struct {
	connected             bool
	tokenTimeout          bool
	tokenError            bool
	disconnectInvocations int
}

func (c *fakeClient) IsConnected() bool {
	return c.connected
}

func (c *fakeClient) IsConnectionOpen() bool {
	return false
}

func (c *fakeClient) Connect() pahomqtt.Token {
	return nil
}

func (c *fakeClient) Disconnect(quiesce uint) {
	c.disconnectInvocations++
}

func (c *fakeClient) Publish(topic string, qos byte, retained bool, payload interface{}) pahomqtt.Token {
	return nil
}

func (c *fakeClient) Subscribe(topic string, qos byte, callback pahomqtt.MessageHandler) pahomqtt.Token {
	return &fakeToken{timeout: c.tokenTimeout, error: c.tokenError}
}

func (c *fakeClient) SubscribeMultiple(filters map[string]byte, callback pahomqtt.MessageHandler) pahomqtt.Token {
	return nil
}

func (c *fakeClient) Unsubscribe(topics ...string) pahomqtt.Token {
	return nil
}

func (c *fakeClient) AddRoute(topic string, callback pahomqtt.MessageHandler) {
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
			name: "Subscription errored",
			fields: fields{
				c: &fakeClient{tokenError: true},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		mh := func(client pahomqtt.Client, msg pahomqtt.Message) {}
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
