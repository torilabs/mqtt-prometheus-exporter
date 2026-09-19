package mqtt

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/torilabs/mqtt-prometheus-exporter/log"
	"go.uber.org/zap"
)

const (
	// clientIDPrefix and clientIDRandomLength keep the client ID within what every MQTT broker must accept
	// (MQTT 3.1, 3.1.1 and 5.0): 1 to 23 bytes made of the characters 0-9a-zA-Z only.
	clientIDPrefix       = "mqttprom"
	clientIDRandomLength = 12
	// subscribeFailureCode is the SUBACK return code sent by a broker that refused a subscription.
	subscribeFailureCode = 0x80
)

// Listener provides actions over MQTT client.
type Listener interface {
	Subscribe(topic string, mh pahomqtt.MessageHandler) error
	Close()
	Check(ctx context.Context) error
	// ConnectionLost receives the error that broke the connection to the broker.
	// The listener never reconnects on its own: the caller is expected to terminate and let the orchestrator restart it.
	ConnectionLost() <-chan error
}

type listener struct {
	c       pahomqtt.Client
	timeout time.Duration
	lost    chan error
}

// subscriptionResult is implemented by tokens of subscribe operations.
type subscriptionResult interface {
	Result() map[string]byte
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

// WithTimeout is option that sets timeout of connection and subscription to MQTT broker.
func WithTimeout(timeout time.Duration) ListenerOption {
	return func(opts *pahomqtt.ClientOptions) {
		opts.SetConnectTimeout(timeout)
	}
}

// WithKeepAlive is option that sets the interval of keep alive messages sent to MQTT broker.
func WithKeepAlive(keepAlive time.Duration) ListenerOption {
	return func(opts *pahomqtt.ClientOptions) {
		opts.SetKeepAlive(keepAlive)
	}
}

// WithPingTimeout is option that sets how long MQTT client waits for a keep alive response before considering the connection lost.
func WithPingTimeout(timeout time.Duration) ListenerOption {
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
	opts.SetClientID(newClientID())

	// Subscriptions are not restored by a reconnection (clean session), so a lost connection must end the process
	// instead of leaving a connected client that silently receives nothing.
	lost := make(chan error, 1)
	opts.SetAutoReconnect(false)
	opts.SetConnectRetry(false)
	opts.SetConnectionLostHandler(newConnectionLostHandler(lost))

	log.Logger.Infof("Will connect to MQTT Brokers '%v'.", opts.Servers)
	client := pahomqtt.NewClient(opts)
	token := client.Connect()

	if ok := token.WaitTimeout(opts.ConnectTimeout); !ok {
		return nil, fmt.Errorf("MQTT connection timed out in '%v'", opts.ConnectTimeout)
	}

	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("MQTT connection failed: %w", err)
	}

	if !client.IsConnected() {
		return nil, fmt.Errorf("MQTT connection unsuccessful to brokers '%v'", opts.Servers)
	}
	log.Logger.Infof("Connected to MQTT Brokers '%v'.", opts.Servers)

	return &listener{c: client, timeout: opts.ConnectTimeout, lost: lost}, nil
}

// newClientID generates a unique client ID. Two clients sharing an ID make the broker drop the older connection,
// which would make both exporters terminate in turns, so the random part is taken from a cryptographic source.
func newClientID() string {
	return clientIDPrefix + rand.Text()[:clientIDRandomLength]
}

func newConnectionLostHandler(lost chan<- error) pahomqtt.ConnectionLostHandler {
	return func(_ pahomqtt.Client, err error) {
		if err == nil {
			err = errors.New("unknown reason")
		}
		log.Logger.With(zap.Error(err)).Error("MQTT connection lost.")
		select {
		case lost <- err:
		default:
		}
	}
}

func (l *listener) Subscribe(topic string, mh pahomqtt.MessageHandler) error {
	log.Logger.Infof("Will subscribe to topic '%s'.", topic)
	token := l.c.Subscribe(topic, 0, mh)

	if ok := token.WaitTimeout(l.timeout); !ok {
		return fmt.Errorf("MQTT topic '%s' subscription timed out in '%v'", topic, l.timeout)
	}

	if token.Error() != nil {
		return fmt.Errorf("MQTT topic '%s' subscription failed: %w", topic, token.Error())
	}

	// A refused subscription (e.g. by an ACL) is reported only in the return code, not as an error of the token.
	if res, ok := token.(subscriptionResult); ok {
		if qos, found := res.Result()[topic]; found && qos == subscribeFailureCode {
			return fmt.Errorf("MQTT topic '%s' subscription refused by the broker", topic)
		}
	}
	return nil
}

func (l *listener) Close() {
	if l.c.IsConnected() {
		l.c.Disconnect(100)
		log.Logger.Info("MQTT Brokers disconnected.")
	}
}

func (l *listener) Check(_ context.Context) error {
	if !l.c.IsConnectionOpen() {
		return fmt.Errorf("MQTT client disconnected")
	}
	return nil
}

func (l *listener) ConnectionLost() <-chan error {
	return l.lost
}
