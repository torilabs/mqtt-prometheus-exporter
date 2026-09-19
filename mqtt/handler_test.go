package mqtt

import (
	"fmt"
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

func (c *fakeCollector) Observe(metric config.Metric, topic string, v float64, labelValues ...string) {
	c.observed = true
	c.obsMetric = metric
	c.obsTopic = topic
	c.obsValue = v
	c.obsLabelValues = labelValues
}

func (c *fakeCollector) Describe(chan<- *prometheus.Desc) {
}

func (c *fakeCollector) Collect(chan<- prometheus.Metric) {
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
		metric config.Metric
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
			name: "Raw value received and processed",
			args: args{
				metric: config.Metric{
					MqttTopic:   "/topic/level2/level3/#",
					TopicLabels: map[string]int{"customTopic": 2, "customTopic2": 3, "customTopic3": 4},
				},
			},
			msg: fakeMessage{
				topic:   "/topic/level2/level3/device",
				payload: []byte("25.12"),
			},
			wantObserved:    true,
			wantValue:       25.12,
			wantLabelValues: []string{"/topic/level2/level3/device", "level2", "level3", "device"},
		},
		{
			name: "Raw value received and failed to parse",
			args: args{
				metric: config.Metric{
					MqttTopic: "/topic/level2/level3/#",
				},
			},
			msg: fakeMessage{
				topic:   "/topic/level2/level3/device",
				payload: []byte("not a number"),
			},
			wantObserved: false,
		},
		{
			name: "JSON value on 1st level parsed",
			args: args{
				metric: config.Metric{
					MqttTopic:   "/topic/level2/level3/#",
					TopicLabels: map[string]int{"customTopic": 2, "customTopic2": 3},
					JSONField:   "size",
				},
			},
			msg: fakeMessage{
				topic:   "/topic/level2/level3/device",
				payload: []byte(`{"city":"Tokyo", "temperatures": {"out": 12.5, "in": 22.15}, "size": -5}`),
			},
			wantObserved:    true,
			wantValue:       -5,
			wantLabelValues: []string{"/topic/level2/level3/device", "level2", "level3"},
		},
		{
			name: "JSON value on 2nd level parsed",
			args: args{
				metric: config.Metric{
					MqttTopic: "/topic/level2/level3/#",
					JSONField: "temperatures.out",
				},
			},
			msg: fakeMessage{
				topic:   "/topic/level2/level3/device",
				payload: []byte(`{"city":"Tokyo", "temperatures": {"out": 12.5, "in": 22.15}, "size": -5}`),
			},
			wantObserved:    true,
			wantValue:       12.5,
			wantLabelValues: []string{"/topic/level2/level3/device"},
		},
		{
			name: "JSON value as object failed to parse",
			args: args{
				metric: config.Metric{
					MqttTopic: "/topic/level2/level3/#",
					JSONField: "temperatures",
				},
			},
			msg: fakeMessage{
				topic:   "/topic/level2/level3/device",
				payload: []byte(`{"city":"Tokyo", "temperatures": {"out": 12.5, "in": 22.15}, "size": -5}`),
			},
			wantObserved: false,
		},
		{
			name: "JSON value as non numeric failed to parse",
			args: args{
				metric: config.Metric{
					MqttTopic: "/topic/level2/level3/#",
					JSONField: "city",
				},
			},
			msg: fakeMessage{
				topic:   "/topic/level2/level3/device",
				payload: []byte(`{"city":"Tokyo", "temperatures": {"out": 12.5, "in": 22.15}, "size": -5}`),
			},
			wantObserved: false,
		},
		{
			name: "JSON labels from nested properties combined with topic labels",
			args: args{
				metric: config.Metric{
					MqttTopic:   "/topic/level2/level3/#",
					TopicLabels: map[string]int{"customTopic": 2, "another": 3},
					JSONField:   "size",
					JSONLabels:  map[string]string{"zone": "location.zone", "city": "city", "deep": "a.b.c"},
				},
			},
			msg: fakeMessage{
				topic:   "/topic/level2/level3/device",
				payload: []byte(`{"city":"Tokyo", "location": {"zone": "north"}, "a": {"b": {"c": "deep value"}}, "size": 7}`),
			},
			wantObserved:    true,
			wantValue:       7,
			wantLabelValues: []string{"/topic/level2/level3/device", "level3", "level2", "Tokyo", "deep value", "north"},
		},
		{
			name: "JSON labels scalar conversion",
			args: args{
				metric: config.Metric{
					MqttTopic:  "/topic",
					JSONField:  "size",
					JSONLabels: map[string]string{"a_str": "s", "b_int": "i", "c_float": "f", "d_true": "t", "e_false": "b", "f_empty": "e", "g_big": "big"},
				},
			},
			msg: fakeMessage{
				topic:   "/topic",
				payload: []byte(`{"size": 1, "s": "text", "i": 42, "f": 12.5, "t": true, "b": false, "e": "", "big": 1e21}`),
			},
			wantObserved:    true,
			wantValue:       1,
			wantLabelValues: []string{"/topic", "text", "42", "12.5", "true", "false", "", "1000000000000000000000"},
		},
		{
			name: "JSON label property missing",
			args: args{
				metric: config.Metric{MqttTopic: "/topic", JSONField: "size", JSONLabels: map[string]string{"room": "room"}},
			},
			msg:          fakeMessage{topic: "/topic", payload: []byte(`{"size": 1}`)},
			wantObserved: false,
		},
		{
			name: "JSON label nested property missing",
			args: args{
				metric: config.Metric{MqttTopic: "/topic", JSONField: "size", JSONLabels: map[string]string{"room": "a.b"}},
			},
			msg:          fakeMessage{topic: "/topic", payload: []byte(`{"size": 1, "a": {"c": "x"}}`)},
			wantObserved: false,
		},
		{
			name: "JSON label property null",
			args: args{
				metric: config.Metric{MqttTopic: "/topic", JSONField: "size", JSONLabels: map[string]string{"room": "room"}},
			},
			msg:          fakeMessage{topic: "/topic", payload: []byte(`{"size": 1, "room": null}`)},
			wantObserved: false,
		},
		{
			name: "JSON label property array",
			args: args{
				metric: config.Metric{MqttTopic: "/topic", JSONField: "size", JSONLabels: map[string]string{"room": "room"}},
			},
			msg:          fakeMessage{topic: "/topic", payload: []byte(`{"size": 1, "room": ["a"]}`)},
			wantObserved: false,
		},
		{
			name: "JSON label property object",
			args: args{
				metric: config.Metric{MqttTopic: "/topic", JSONField: "size", JSONLabels: map[string]string{"room": "room"}},
			},
			msg:          fakeMessage{topic: "/topic", payload: []byte(`{"size": 1, "room": {"a": "b"}}`)},
			wantObserved: false,
		},
		{
			name: "JSON labels not observed when value is missing",
			args: args{
				metric: config.Metric{MqttTopic: "/topic", JSONField: "size", JSONLabels: map[string]string{"room": "room"}},
			},
			msg:          fakeMessage{topic: "/topic", payload: []byte(`{"room": "kitchen"}`)},
			wantObserved: false,
		},
		{
			name:            "JSON boolean true converted to 1",
			args:            args{metric: config.Metric{MqttTopic: "/topic", JSONField: "door.open"}},
			msg:             fakeMessage{topic: "/topic", payload: []byte(`{"door": {"open": true}}`)},
			wantObserved:    true,
			wantValue:       1,
			wantLabelValues: []string{"/topic"},
		},
		{
			name:            "JSON boolean false converted to 0",
			args:            args{metric: config.Metric{MqttTopic: "/topic", JSONField: "open"}},
			msg:             fakeMessage{topic: "/topic", payload: []byte(`{"open": false}`)},
			wantObserved:    true,
			wantValue:       0,
			wantLabelValues: []string{"/topic"},
		},
		{
			name:            "JSON string boolean ON converted to 1",
			args:            args{metric: config.Metric{MqttTopic: "/topic", JSONField: "state"}},
			msg:             fakeMessage{topic: "/topic", payload: []byte(`{"state": "ON"}`)},
			wantObserved:    true,
			wantValue:       1,
			wantLabelValues: []string{"/topic"},
		},
		{
			name:            "JSON string boolean no converted to 0",
			args:            args{metric: config.Metric{MqttTopic: "/topic", JSONField: "state"}},
			msg:             fakeMessage{topic: "/topic", payload: []byte(`{"state": " No "}`)},
			wantObserved:    true,
			wantValue:       0,
			wantLabelValues: []string{"/topic"},
		},
		{
			name: "JSON boolean combined with topic and JSON labels",
			args: args{
				metric: config.Metric{
					MqttTopic:   "/home/+/state",
					TopicLabels: map[string]int{"device": 2},
					JSONField:   "door.open",
					JSONLabels:  map[string]string{"room": "location.room", "armed": "armed"},
				},
			},
			msg: fakeMessage{
				topic:   "/home/door1/state",
				payload: []byte(`{"door": {"open": "yes"}, "location": {"room": "hall"}, "armed": true}`),
			},
			wantObserved:    true,
			wantValue:       1,
			wantLabelValues: []string{"/home/door1/state", "door1", "true", "hall"},
		},
		{
			name:         "JSON string not a boolean failed to parse",
			args:         args{metric: config.Metric{MqttTopic: "/topic", JSONField: "state"}},
			msg:          fakeMessage{topic: "/topic", payload: []byte(`{"state": "maybe"}`)},
			wantObserved: false,
		},
		{
			name:         "JSON null value failed to parse",
			args:         args{metric: config.Metric{MqttTopic: "/topic", JSONField: "state"}},
			msg:          fakeMessage{topic: "/topic", payload: []byte(`{"state": null}`)},
			wantObserved: false,
		},
		{
			name:         "JSON array value failed to parse",
			args:         args{metric: config.Metric{MqttTopic: "/topic", JSONField: "state"}},
			msg:          fakeMessage{topic: "/topic", payload: []byte(`{"state": [true]}`)},
			wantObserved: false,
		},
	}
	for _, tt := range tests {
		for i := range 100 {
			t.Run(fmt.Sprintf("%s-%d", tt.name, i+1), func(t *testing.T) {
				collector := fakeCollector{}
				mh := NewMessageHandler(tt.args.metric, &collector)
				mh(&fakeClient{}, &tt.msg)

				if tt.wantObserved != collector.observed {
					t.Errorf("observe = %v, want %v", collector.observed, tt.wantObserved)
					return
				}

				if collector.observed {
					if tt.msg.topic != collector.obsTopic {
						t.Errorf("topic = %v, want %v", collector.obsTopic, tt.msg.topic)
					}
					if tt.wantValue != collector.obsValue {
						t.Errorf("value = %v, want %v", collector.obsValue, tt.wantValue)
					}
					if !reflect.DeepEqual(tt.args.metric, collector.obsMetric) {
						t.Errorf("metric = %v, want %v", collector.obsMetric, tt.args.metric)
					}
					if !reflect.DeepEqual(tt.wantLabelValues, collector.obsLabelValues) {
						t.Errorf("labelValues = %v, want %v", collector.obsLabelValues, tt.wantLabelValues)
					}
				}
			})
		}
	}
}
