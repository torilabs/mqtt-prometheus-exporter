package mqtt

import (
	"fmt"
	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"github.com/torilabs/mqtt-prometheus-exporter/config"
	"github.com/torilabs/mqtt-prometheus-exporter/log"
	"time"
)

// Listener provides actions over MQTT client
type Listener interface {
	Subscribe(topic string) error
	Close()
}

type listener struct {
	c       pahomqtt.Client
	timeout time.Duration
}

// NewListener creates listener over MQTT client
func NewListener(cfg config.Mqtt) (Listener, error) {
	opts := createClientOptions(cfg)
	log.Logger.Infof("Will connect to MQTT Brokers '%v'.", opts.Servers)
	client := pahomqtt.NewClient(opts)
	token := client.Connect()

	if ok := token.WaitTimeout(cfg.Timeout); !ok {
		return nil, errors.Errorf("MQTT connection timed out in '%v'", cfg.Timeout)
	}

	if err := token.Error(); err != nil {
		return nil, errors.Wrap(err, "MQTT connection failed")
	}

	if !client.IsConnected() {
		return nil, errors.Errorf("MQTT connection unsuccessful to brokers '%v'", opts.Servers)
	}
	log.Logger.Infof("Connected to MQTT Brokers '%v'.", opts.Servers)

	return &listener{c: client, timeout: cfg.Timeout}, nil
}

func createClientOptions(cfg config.Mqtt) *pahomqtt.ClientOptions {
	opts := pahomqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))
	opts.SetUsername(cfg.Username)
	opts.SetPassword(cfg.Password)
	opts.SetClientID(cfg.ClientID)
	return opts
}

func (l *listener) Subscribe(topic string) error {
	log.Logger.Infof("Will subscribe to topic '%s'.", topic)
	token := l.c.Subscribe(topic, 0, func(client pahomqtt.Client, msg pahomqtt.Message) {
		log.Logger.Infof("* [%s] %s", msg.Topic(), string(msg.Payload()))
	})

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
