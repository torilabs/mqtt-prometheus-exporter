package mqtt

import (
	"reflect"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/torilabs/mqtt-prometheus-exporter/config"
)

type fakeCollector struct {
	observed       bool
	obsMetric      config.Metric
	obsTopic       string
	obsValue       float64
	obsLabelValues []string
}

func (c *fakeCollector) Observe(metric config.Metric, topic string, v float64, labelValues []string) {
	c.observed = true
	c.obsMetric = metric
	c.obsTopic = topic
	c.obsValue = v
	c.obsLabelValues = labelValues
}

func (c *fakeCollector) Describe(ch chan<- *prometheus.Desc) {
}

func (c *fakeCollector) Collect(ch chan<- prometheus.Metric) {
}

type fakeMessage struct {
	topic   string
	payload []byte
}

func (m *fakeMessage) Duplicate() bool {
	return false
}

func (m *fakeMessage) Qos() byte {
	return 0
}

func (m *fakeMessage) Retained() bool {
	return false
}

func (m *fakeMessage) Topic() string {
	return m.topic
}

func (m *fakeMessage) MessageID() uint16 {
	return 0
}

func (m *fakeMessage) Payload() []byte {
	return m.payload
}

func (m *fakeMessage) Ack() {
}

func Test_messageHandler(t *testing.T) {
	type args struct {
		metric    config.Metric
		collector fakeCollector
	}
	tests := []struct {
		name            string
		args            args
		msg             fakeMessage
		wantObserved    bool
		wantValue       float64
		wantLabelValues []string
	}{
		{
			name: "Value received and processed",
			args: args{
				metric: config.Metric{
					MqttTopic:   "/topic/level2/level3/#",
					TopicLabels: map[string]int{"customTopic": 2},
				},
				collector: fakeCollector{},
			},
			msg: fakeMessage{
				topic:   "/topic/level2/level3/device",
				payload: []byte("25.12"),
			},
			wantObserved:    true,
			wantValue:       25.12,
			wantLabelValues: []string{"/topic/level2/level3/device", "level2"},
		},
		{
			name: "Value received and failed to parse",
			args: args{
				metric: config.Metric{
					MqttTopic: "/topic/level2/level3/#",
				},
				collector: fakeCollector{},
			},
			msg: fakeMessage{
				topic:   "/topic/level2/level3/device",
				payload: []byte("not a number"),
			},
			wantObserved: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mh := NewMessageHandler(tt.args.metric, &tt.args.collector)
			mh(&fakeClient{}, &tt.msg)

			if tt.wantObserved != tt.args.collector.observed {
				t.Errorf("observe = %v, want %v", tt.args.collector.observed, tt.wantObserved)
				return
			}

			if tt.args.collector.observed {
				if tt.msg.topic != tt.args.collector.obsTopic {
					t.Errorf("topic = %v, want %v", tt.args.collector.obsTopic, tt.msg.topic)
				}
				if tt.wantValue != tt.args.collector.obsValue {
					t.Errorf("value = %v, want %v", tt.args.collector.obsValue, tt.wantValue)
				}
				if !reflect.DeepEqual(tt.args.metric, tt.args.collector.obsMetric) {
					t.Errorf("metric = %v, want %v", tt.args.collector.obsMetric, tt.args.metric)
				}
				if !reflect.DeepEqual(tt.wantLabelValues, tt.args.collector.obsLabelValues) {
					t.Errorf("labelValues = %v, want %v", tt.args.collector.obsLabelValues, tt.wantLabelValues)
				}
			}
		})
	}
}
