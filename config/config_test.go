package config

import (
	"io/fs"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"gopkg.in/validator.v2"
)

func TestMetric_PrometheusDescription(t *testing.T) {
	tests := []struct {
		name   string
		metric Metric
		want   string
	}{
		{
			name: "valid description",
			metric: Metric{
				PrometheusName: "name",
				Help:           "help msg",
				TopicLabels: map[string]int{
					"device": 1,
				},
				ConstantLabels: map[string]string{
					"const_label": "label_value",
				},
			},
			want: "Desc{fqName: \"name\", help: \"help msg\", constLabels: {const_label=\"label_value\"}, variableLabels: {topic,device}}",
		},
		{
			name: "JSON labels are appended after topic labels in alphabetical order",
			metric: Metric{
				PrometheusName: "name",
				Help:           "help msg",
				TopicLabels:    map[string]int{"device": 1, "area": 2},
				JSONLabels:     map[string]string{"zone": "location.zone", "firmware": "meta.fw"},
			},
			want: "Desc{fqName: \"name\", help: \"help msg\", constLabels: {}, variableLabels: {topic,area,device,firmware,zone}}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.metric.PrometheusDescription(); !reflect.DeepEqual(got.String(), tt.want) {
				t.Errorf("PrometheusDescription() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetric_PrometheusValueType(t *testing.T) {
	tests := []struct {
		name   string
		metric Metric
		want   prometheus.ValueType
	}{
		{
			name: "gauge type",
			metric: Metric{
				MetricType: "gauge",
			},
			want: prometheus.GaugeValue,
		},
		{
			name: "counter type",
			metric: Metric{
				MetricType: "counter",
			},
			want: prometheus.CounterValue,
		},
		{
			name: "other types",
			metric: Metric{
				MetricType: "",
			},
			want: prometheus.UntypedValue,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.metric.PrometheusValueType(); got != tt.want {
				t.Errorf("PrometheusValueType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		rawCfg  string
		wantCfg Configuration
		wantErr bool
	}{
		{
			name: "default configuration",
			wantCfg: Configuration{
				Logging: Logger{
					Level: "info",
				},
				Server: Server{
					Port: 8079,
				},
				MQTT: MQTT{
					Host:        "",
					Port:        9641,
					Timeout:     time.Second * 3,
					KeepAlive:   time.Second * 30,
					PingTimeout: time.Second * 10,
				},
				Cache: Cache{
					Expiration: time.Second * 60,
				},
			},
		},
		{
			name: "full configuration",
			rawCfg: `# Logger configuration
logging:
  level: DEBUG
  developmentMode: true
server:
  port: 8077
mqtt:
  host: "ws://192.168.1.1"
  port: 9001
  username: "user"
  password: "passwd"
  timeout: 4s
  keep_alive: 20s
  ping_timeout: 6s
cache:
  expiration: 100s
metrics:
  - mqtt_topic: "/home/+/memory"
    prom_name: "memory"
    type: "gauge"
    help: "free memory of a device"
    const_labels:
      - mylabel: "label value"
    topic_labels:
      - device: 2
      - device2: -3
  - mqtt_topic: "+/home/rpi/#"
    prom_name: "rpi"
    type: "gauge"
`,
			wantCfg: Configuration{
				Logging: Logger{
					Level:           "DEBUG",
					DevelopmentMode: true,
				},
				Server: Server{
					Port: 8077,
				},
				MQTT: MQTT{
					Host:        "ws://192.168.1.1",
					Port:        9001,
					Username:    "user",
					Password:    "passwd",
					Timeout:     time.Second * 4,
					KeepAlive:   time.Second * 20,
					PingTimeout: time.Second * 6,
				},
				Cache: Cache{
					Expiration: time.Second * 100,
				},
				Metrics: []Metric{
					{
						PrometheusName: "memory",
						MqttTopic:      "/home/+/memory",
						MetricType:     "gauge",
						Help:           "free memory of a device",
						ConstantLabels: map[string]string{
							"mylabel": "label value",
						},
						TopicLabels: map[string]int{
							"device":  2,
							"device2": -3,
						},
					},
					{
						PrometheusName: "rpi",
						MqttTopic:      "+/home/rpi/#",
						MetricType:     "gauge",
					},
				},
			},
		},
		{
			name: "JSON labels configuration",
			rawCfg: `metrics:
  - mqtt_topic: "/home/overview"
    prom_name: "sensor_count"
    json_field: "total.count"
    json_labels:
      room: "location.room"
      firmware: "meta.fw"
  - mqtt_topic: "/home/other"
    prom_name: "other"
    json_field: "value"
    json_labels:
      - room: "room"
`,
			wantCfg: Configuration{
				Logging: Logger{Level: "info"},
				Server:  Server{Port: 8079},
				MQTT:    MQTT{Port: 9641, Timeout: time.Second * 3, KeepAlive: time.Second * 30, PingTimeout: time.Second * 10},
				Cache:   Cache{Expiration: time.Second * 60},
				Metrics: []Metric{
					{
						PrometheusName: "sensor_count",
						MqttTopic:      "/home/overview",
						JSONField:      "total.count",
						JSONLabels:     map[string]string{"room": "location.room", "firmware": "meta.fw"},
					},
					{
						PrometheusName: "other",
						MqttTopic:      "/home/other",
						JSONField:      "value",
						JSONLabels:     map[string]string{"room": "room"},
					},
				},
			},
		},
		{
			name:    "invalid configuration",
			rawCfg:  `sth wrong`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := os.CreateTemp("/tmp", "mqtt-prometheus-exporter-*.yaml")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(file.Name())
			if err := os.WriteFile(file.Name(), []byte(tt.rawCfg), fs.ModePerm); err != nil {
				t.Fatal(err)
			}
			viper.Reset()
			viper.SetConfigFile(file.Name())

			gotCfg, err := Parse()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotCfg, tt.wantCfg) {
				t.Errorf("Parse() gotCfg = %v, want %v", gotCfg, tt.wantCfg)
			}
		})
	}
}

func TestParse_Errors(t *testing.T) {
	t.Run("file not exist", func(t *testing.T) {
		viper.Reset()
		viper.SetConfigFile("/tmp/not-exist.yaml")

		_, err := Parse()
		expErrMsg := "failed to read configuration: open /tmp/not-exist.yaml: no such file or directory"
		if err.Error() != expErrMsg {
			t.Errorf("Parse() error = %v, want '%s'", err, expErrMsg)
		}
	})

	t.Run("invalid file content", func(t *testing.T) {
		file, err := os.CreateTemp("/tmp", "mqtt-prometheus-exporter-*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(file.Name())
		if err := os.WriteFile(file.Name(), []byte("{"), fs.ModePerm); err != nil {
			t.Fatal(err)
		}

		viper.Reset()
		viper.SetConfigFile(file.Name())

		_, err = Parse()
		expErrMsg := "failed to read configuration: While parsing config: yaml: line 1: did not find expected node content"
		if err.Error() != expErrMsg {
			t.Errorf("Parse() error = %v, want '%s'", err, expErrMsg)
		}
	})
}

func TestMetricValidation(t *testing.T) {
	tests := []struct {
		name    string
		metric  Metric
		wantErr bool
	}{
		{
			name: "valid metric",
			metric: Metric{
				PrometheusName: "name_1:stat",
				MqttTopic:      "/home/+/memory",
			},
			wantErr: false,
		},
		{
			name: "invalid metric - missing metric name",
			metric: Metric{
				MqttTopic: "/home/+/memory",
			},
			wantErr: true,
		},
		{
			name: "invalid metric - missing MQTT topic",
			metric: Metric{
				PrometheusName: "name",
			},
			wantErr: true,
		},
		{
			name: "invalid metric - invalid character in a name",
			metric: Metric{
				PrometheusName: "name-stat",
				MqttTopic:      "/home/+/memory",
			},
			wantErr: true,
		},
		{
			name: "invalid metric - name starts with number",
			metric: Metric{
				PrometheusName: "1name",
				MqttTopic:      "/home/+/memory",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validate := validator.NewValidator()
			if err := validate.Validate(&tt.metric); (err != nil) != tt.wantErr {
				t.Errorf("validation error '%v', want %v", err, tt.wantErr)
			}
		})
	}
}

func TestTopicLabels_KeysInOrder(t *testing.T) {
	tl := TopicLabels{"key1": 5, "someKey": -1, "nKey": 25, "mKey": 0, "rKey": -6}
	refValue := tl.KeysInOrder()
	for range 100 {
		if got := tl.KeysInOrder(); !reflect.DeepEqual(got, refValue) {
			t.Errorf("KeysInOrder() = %v, want %v", got, refValue)
		}
	}
}

func TestJSONLabels_KeysInOrder(t *testing.T) {
	jl := JSONLabels{"zone": "a.b", "area": "c", "room": "d", "firmware": "e"}
	want := []string{"area", "firmware", "room", "zone"}
	for range 100 {
		if got := jl.KeysInOrder(); !reflect.DeepEqual(got, want) {
			t.Errorf("KeysInOrder() = %v, want %v", got, want)
		}
	}
	if got := (JSONLabels)(nil).KeysInOrder(); len(got) != 0 {
		t.Errorf("KeysInOrder() of nil = %v, want empty", got)
	}
}

func TestMetric_ValidateLabels(t *testing.T) {
	tests := []struct {
		name    string
		metric  Metric
		wantErr string
	}{
		{
			name:   "no JSON labels",
			metric: Metric{TopicLabels: map[string]int{"topic": 1}},
		},
		{
			name: "valid JSON labels",
			metric: Metric{
				JSONField:      "value",
				ConstantLabels: map[string]string{"const": "x"},
				TopicLabels:    map[string]int{"device": 1},
				JSONLabels:     map[string]string{"room": "location.room", "_private1": "a"},
			},
		},
		{
			name:    "JSON labels without json_field",
			metric:  Metric{JSONLabels: map[string]string{"room": "room"}},
			wantErr: "json_labels require json_field",
		},
		{
			name:    "label name with invalid character",
			metric:  Metric{JSONField: "v", JSONLabels: map[string]string{"my-label": "a"}},
			wantErr: "not a valid prometheus label name",
		},
		{
			name:    "label name starting with a number",
			metric:  Metric{JSONField: "v", JSONLabels: map[string]string{"1label": "a"}},
			wantErr: "not a valid prometheus label name",
		},
		{
			name:    "empty label name",
			metric:  Metric{JSONField: "v", JSONLabels: map[string]string{"": "a"}},
			wantErr: "not a valid prometheus label name",
		},
		{
			name:    "reserved label name prefix",
			metric:  Metric{JSONField: "v", JSONLabels: map[string]string{"__name": "a"}},
			wantErr: "not a valid prometheus label name",
		},
		{
			name:    "empty property path",
			metric:  Metric{JSONField: "v", JSONLabels: map[string]string{"room": " "}},
			wantErr: "empty JSON property path",
		},
		{
			name:    "collision with topic label",
			metric:  Metric{JSONField: "v", JSONLabels: map[string]string{"topic": "a"}},
			wantErr: "built-in",
		},
		{
			name: "collision with constant label",
			metric: Metric{
				JSONField:      "v",
				ConstantLabels: map[string]string{"room": "x"},
				JSONLabels:     map[string]string{"room": "a"},
			},
			wantErr: "constant label",
		},
		{
			name: "collision with topic labels",
			metric: Metric{
				JSONField:   "v",
				TopicLabels: map[string]int{"room": 1},
				JSONLabels:  map[string]string{"room": "a"},
			},
			wantErr: "topic label",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metric.ValidateLabels()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("ValidateLabels() unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ValidateLabels() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestMQTTValidation(t *testing.T) {
	valid := MQTT{Timeout: time.Second * 3, KeepAlive: time.Second * 30, PingTimeout: time.Second * 10}
	tests := []struct {
		name    string
		mutate  func(m *MQTT)
		wantErr bool
	}{
		{name: "valid", mutate: func(*MQTT) {}},
		{name: "zero timeout", mutate: func(m *MQTT) { m.Timeout = 0 }, wantErr: true},
		{name: "zero ping timeout", mutate: func(m *MQTT) { m.PingTimeout = 0 }, wantErr: true},
		{name: "zero keep alive", mutate: func(m *MQTT) { m.KeepAlive = 0 }, wantErr: true},
		{name: "keep alive below one second", mutate: func(m *MQTT) { m.KeepAlive = time.Millisecond * 500 }, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := valid
			tt.mutate(&m)
			cfg := Configuration{MQTT: m}
			if err := validator.NewValidator().Validate(&cfg); (err != nil) != tt.wantErr {
				t.Errorf("validation error '%v', want %v", err, tt.wantErr)
			}
		})
	}
}
