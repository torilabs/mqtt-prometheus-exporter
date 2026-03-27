package mqtt

import (
	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/torilabs/mqtt-prometheus-exporter/log"
	"go.uber.org/zap"
)

// DelegatingMessageHandler wraps multiple MessageHandlers and delegates
// message processing to all of them. This allows multiple metrics to be
// processed from a single MQTT topic subscription.
type DelegatingMessageHandler struct {
	handlers []pahomqtt.MessageHandler
}

// NewDelegatingMessageHandler creates a new delegating handler with the given handlers.
func NewDelegatingMessageHandler(handlers ...pahomqtt.MessageHandler) pahomqtt.MessageHandler {
	if len(handlers) == 0 {
		log.Logger.Warn("Creating DelegatingMessageHandler with no handlers")
	}

	dmh := &DelegatingMessageHandler{
		handlers: handlers,
	}
	return dmh.handle
}

// handle processes the message by delegating to all contained handlers.
func (d *DelegatingMessageHandler) handle(client pahomqtt.Client, msg pahomqtt.Message) {
	log.Logger.Debugf("DelegatingMessageHandler processing message from topic '%s' with %d handlers", msg.Topic(), len(d.handlers))

	for i, handler := range d.handlers {
		func(idx int, h pahomqtt.MessageHandler) {
			defer func() {
				if r := recover(); r != nil {
					log.Logger.With(zap.Any("panic", r)).Errorf("Handler %d panicked while processing message from topic '%s'", idx, msg.Topic())
				}
			}()
			h(client, msg)
		}(i, handler)
	}
}
