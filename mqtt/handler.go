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
		h.collector.Observe(h.metric, msg.Topic(), floatValue, h.topicLabelValues(msg.Topic())...)
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

		if value, ok := findInJSON(jsonMap, h.metric.JSONField); ok {
			floatValue, isBoolean, err := jsonValueToFloat(value)
			if err != nil {
				log.Logger.With(zap.Error(err)).Warnf("Got data with unexpected value %q and failed to parse to float.", fmt.Sprintf("%v", value))
				return
			}
			if isBoolean {
				log.Logger.Debugf("Converted boolean value %q of '%s' to %v.", fmt.Sprintf("%v", value), h.metric.JSONField, floatValue)
			}
			labelValues, err := h.jsonLabelValues(msg.Topic(), jsonMap)
			if err != nil {
				log.Logger.With(zap.Error(err)).Warnf("Skipping observation from '%s' topic.", msg.Topic())
				return
			}
			h.collector.Observe(h.metric, msg.Topic(), floatValue, labelValues...)
		}
	}
}

// topicLabelValues returns the topic followed by the values of topic labels in the alphabetical order of label names.
func (h *messageHandler) topicLabelValues(topic string) []string {
	labelValues := make([]string, 0, 1+len(h.metric.TopicLabels)+len(h.metric.JSONLabels))
	labelValues = append(labelValues, topic)
	for _, tl := range h.metric.TopicLabels.KeysInOrder() {
		labelValues = append(labelValues, getTopicPart(topic, h.metric.TopicLabels[tl]))
	}
	return labelValues
}

// jsonLabelValues returns topic label values followed by the values of JSON labels in the alphabetical order of label names.
// It fails when any configured property is missing, null, an array or an object.
func (h *messageHandler) jsonLabelValues(topic string, jsonMap map[string]interface{}) ([]string, error) {
	labelValues := h.topicLabelValues(topic)
	for _, name := range h.metric.JSONLabels.KeysInOrder() {
		path := h.metric.JSONLabels[name]
		raw, found := findInJSON(jsonMap, path)
		if !found {
			return nil, fmt.Errorf("property %q for label %q is missing", path, name)
		}
		value, ok := labelValueOf(raw)
		if !ok {
			return nil, fmt.Errorf("property %q for label %q is null or not a string, number or boolean", path, name)
		}
		labelValues = append(labelValues, value)
	}
	return labelValues, nil
}
