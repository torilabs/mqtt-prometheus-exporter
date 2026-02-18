package mqtt

import (
	"encoding/json"
	"fmt"
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
	if metric.JSONField != "" {
		return mh.getJSONMessageHandler()
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
		labelCount := 1 + len(h.metric.TopicLabels)
		labelValues := make([]string, 0, labelCount)
		labelValues = append(labelValues, msg.Topic())
		for _, tl := range h.metric.TopicLabels.KeysInOrder() {
			labelValues = append(labelValues, getTopicPart(msg.Topic(), h.metric.TopicLabels[tl]))
		}
		h.collector.Observe(h.metric, msg.Topic(), floatValue, labelValues...)
	}
}

func (h *messageHandler) getJSONMessageHandler() pahomqtt.MessageHandler {
	return func(_ pahomqtt.Client, msg pahomqtt.Message) {
		log.Logger.Debugf("Received MQTT msg '%s' from '%s' topic. Listener for: '%s'.", msg.Payload(), msg.Topic(), h.metric.MqttTopic)

		jsonMap := make(map[string]interface{})
		if err := json.Unmarshal(msg.Payload(), &jsonMap); err != nil {
			log.Logger.With(zap.Error(err)).Warnf("Got an invalid JSON value '%s' and failed to unmarshal.", msg.Payload())
			return
		}

		labelCount := 1 + len(h.metric.TopicLabels)
		labelValues := make([]string, 0, labelCount)
		labelValues = append(labelValues, msg.Topic())
		for _, tl := range h.metric.TopicLabels.KeysInOrder() {
			labelValues = append(labelValues, getTopicPart(msg.Topic(), h.metric.TopicLabels[tl]))
		}

		if value, ok := findInJSON(jsonMap, h.metric.JSONField); ok {
			floatValue, err := strconv.ParseFloat(fmt.Sprintf("%v", value), 64)
			if err != nil {
				log.Logger.With(zap.Error(err)).Warnf("Got data with unexpected value '%s' and failed to parse to float.", value)
				return
			}
			h.collector.Observe(h.metric, msg.Topic(), floatValue, labelValues...)
		}
	}
}
