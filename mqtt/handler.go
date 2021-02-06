package mqtt

import (
	"strconv"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/torilabs/mqtt-prometheus-exporter/config"
	"github.com/torilabs/mqtt-prometheus-exporter/log"
	"github.com/torilabs/mqtt-prometheus-exporter/prometheus"
	"go.uber.org/zap"
)

type messageHandler struct {
	metric    config.Metric
	collector prometheus.Collector
}

// NewMessageHandler constructs handler for single metric.
func NewMessageHandler(metric config.Metric, collector prometheus.Collector) pahomqtt.MessageHandler {
	mh := &messageHandler{
		metric:    metric,
		collector: collector,
	}
	return mh.getMessageHandler()
}

func (h *messageHandler) getMessageHandler() pahomqtt.MessageHandler {
	return func(_ pahomqtt.Client, msg pahomqtt.Message) {
		strValue := string(msg.Payload())
		log.Logger.Debugf("Received MQTT msg '%s' from '%s' topic. Listener for: '%s'.", strValue, msg.Topic(), h.metric.MqttTopic)
		floatValue, err := strconv.ParseFloat(strValue, 64)
		if err != nil {
			log.Logger.With(zap.Error(err)).Warnf("Got data with unexpected value '%s' and failed to parse to float.", strValue)
			return
		}
		labelValues := []string{msg.Topic()}
		for _, idx := range h.metric.TopicLabels {
			labelValues = append(labelValues, getTopicPart(msg.Topic(), idx))
		}
		h.collector.Observe(&h.metric, msg.Topic(), floatValue, labelValues)
	}
}
