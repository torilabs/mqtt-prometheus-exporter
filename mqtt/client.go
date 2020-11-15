package mqtt

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"github.com/torilabs/mqtt-prometheus-exporter/log"
)

const clientIDPrefix = "mqtt-prometheus-exporter-"

// Listener provides actions over MQTT client.
type Listener interface {
	Subscribe(topic string, mh pahomqtt.MessageHandler) error
	Close()
	Check(ctx context.Context) error
}

type listener struct {
	c       pahomqtt.Client
	timeout time.Duration
}

// ListenerOption allows to configure MQTT client.
type ListenerOption func(options *pahomqtt.ClientOptions)

// WithHostAndPort is option that defines MQTT broker.
func WithHostAndPort(host string, port int) ListenerOption {
	return func(opts *pahomqtt.ClientOptions) {
		opts.AddBroker(fmt.Sprintf("%s:%d", host, port))
	}
}

// WithUsername is option that sets connection credentials.
func WithUsername(username string) ListenerOption {
	return func(opts *pahomqtt.ClientOptions) {
		opts.SetUsername(username)
	}
}

// WithPassword is option that sets connection credentials.
func WithPassword(password string) ListenerOption {
	return func(opts *pahomqtt.ClientOptions) {
		opts.SetPassword(password)
	}
}

// WithTimeout is option that sets MQTT client timeout.
func WithTimeout(timeout time.Duration) ListenerOption {
	return func(opts *pahomqtt.ClientOptions) {
		opts.SetPingTimeout(timeout)
	}
}

// NewListener creates listener over MQTT client.
func NewListener(lo ...ListenerOption) (Listener, error) {
	opts := pahomqtt.NewClientOptions()
	for _, o := range lo {
		o(opts)
	}
	opts.SetClientID(fmt.Sprintf("%s%d", clientIDPrefix, rand.Intn(math.MaxInt16)))

	log.Logger.Infof("Will connect to MQTT Brokers '%v'.", opts.Servers)
	client := pahomqtt.NewClient(opts)
	token := client.Connect()

	if ok := token.WaitTimeout(opts.PingTimeout); !ok {
		return nil, errors.Errorf("MQTT connection timed out in '%v'", opts.PingTimeout)
	}

	if err := token.Error(); err != nil {
		return nil, errors.Wrap(err, "MQTT connection failed")
	}

	if !client.IsConnected() {
		return nil, errors.Errorf("MQTT connection unsuccessful to brokers '%v'", opts.Servers)
	}
	log.Logger.Infof("Connected to MQTT Brokers '%v'.", opts.Servers)

	return &listener{c: client, timeout: opts.PingTimeout}, nil
}

func (l *listener) Subscribe(topic string, mh pahomqtt.MessageHandler) error {
	log.Logger.Infof("Will subscribe to topic '%s'.", topic)
	token := l.c.Subscribe(topic, 0, mh)

	if ok := token.WaitTimeout(l.timeout); !ok {
		return errors.Errorf("MQTT topic '%s' subscription timed out in '%v'", topic, l.timeout)
	}

	return errors.Wrapf(token.Error(), "MQTT topic '%s' subscription failed", topic)
}

func (l *listener) Close() {
	if l.c.IsConnected() {
		l.c.Disconnect(100)
		log.Logger.Info("MQTT Brokers disconnected.")
	}
}

func (l *listener) Check(_ context.Context) error {
	if !l.c.IsConnectionOpen() {
		return errors.New("MQTT client disconnected")
	}
	return nil
}
