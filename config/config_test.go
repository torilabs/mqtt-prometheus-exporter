package config

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
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
			want: "Desc{fqName: \"name\", help: \"help msg\", constLabels: {const_label=\"label_value\"}, variableLabels: [topic device]}",
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
			file, err := ioutil.TempFile("/tmp", "mqtt-prometheus-exporter-*.yaml")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(file.Name())
			if err := ioutil.WriteFile(file.Name(), []byte(tt.rawCfg), 0777); err != nil {
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
