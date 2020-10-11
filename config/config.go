package config

import (
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
)

// Logger configuration structure.
type Logger struct {
	Level           string
	DevelopmentMode bool
}

// Server configuration structure.
type Server struct {
	Port int
}

// MQTT configuration structure.
type MQTT struct {
	ClientID string
	Host     string
	Port     int
	Username string
	Password string
	Timeout  time.Duration
}

// Cache configuration structure.
type Cache struct {
	Expiration time.Duration `mapstructure:"expiration"`
}

// Metric is a mapping between a metric send on mqtt to a prometheus metric.
type Metric struct {
	PrometheusName string            `mapstructure:"prom_name"`
	MqttTopic      string            `mapstructure:"mqtt_topic"`
	Help           string            `mapstructure:"help"`
	MetricType     string            `mapstructure:"type"`
	ConstantLabels map[string]string `mapstructure:"const_labels"`
	TopicLabels    map[string]int    `mapstructure:"topic_labels"`
}

// PrometheusDescription constructs description.
func (m *Metric) PrometheusDescription() *prometheus.Desc {
	varLabels := []string{"topic"}
	for tl := range m.TopicLabels {
		varLabels = append(varLabels, tl)
	}
	return prometheus.NewDesc(
		m.PrometheusName, m.Help, varLabels, m.ConstantLabels,
	)
}

// PrometheusValueType decodes type of prometheus metric.
func (m *Metric) PrometheusValueType() prometheus.ValueType {
	switch m.MetricType {
	case "gauge":
		return prometheus.GaugeValue
	case "counter":
		return prometheus.CounterValue
	default:
		return prometheus.UntypedValue
	}
}

// Configuration structure.
type Configuration struct {
	Logging Logger
	Server  Server
	MQTT    MQTT
	Metrics []Metric
	Cache   Cache
}

// Parse and validate viper config.
func Parse() (cfg Configuration, err error) {
	if err := viper.ReadInConfig(); err != nil {
		return cfg, errors.Wrap(err, "failed to read configuration")
	}

	setDefaults()

	if err := viper.Unmarshal(&cfg); err != nil {
		return cfg, errors.Wrap(err, "failed to deserialize config")
	}

	return cfg, nil
}

func setDefaults() {
	viper.SetDefault("logging.level", "info")

	viper.SetDefault("server.port", 8079)

	viper.SetDefault("mqtt.clientid", "mqtt-prometheus-exporter")
	viper.SetDefault("mqtt.host", "")
	viper.SetDefault("mqtt.port", 9641)
	viper.SetDefault("mqtt.timeout", "3s")

	viper.SetDefault("cache.expiration", "60s")
}
