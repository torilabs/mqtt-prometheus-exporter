package config

import (
	"io/fs"
	"os"
	"reflect"
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
			want: "Desc{fqName: \"name\", help: \"help msg\", constLabels: {const_label=\"label_value\"}, variableLabels: [{topic <nil>} {device <nil>}]}",
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
					Host:    "",
					Port:    9641,
					Timeout: time.Second * 3,
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
					Host:     "ws://192.168.1.1",
					Port:     9001,
					Username: "user",
					Password: "passwd",
					Timeout:  time.Second * 4,
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
	for i := 0; i < 100; i++ {
		if got := tl.KeysInOrder(); !reflect.DeepEqual(got, refValue) {
			t.Errorf("KeysInOrder() = %v, want %v", got, refValue)
		}
	}
}
