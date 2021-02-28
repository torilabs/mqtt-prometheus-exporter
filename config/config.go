package config

import (
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
)

// TopicLabels is metric configuration.
type TopicLabels map[string]int

// KeysInOrder sort keys always the same way.
func (tl TopicLabels) KeysInOrder() []string {
	keys := make([]string, len(tl))
	i := 0
	for k := range tl {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

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
	PrometheusName string            `mapstructure:"prom_name" validate:"nonzero,regexp=^[a-zA-Z_:]([a-zA-Z0-9_:])*$"`
	MqttTopic      string            `mapstructure:"mqtt_topic" validate:"nonzero"`
	Help           string            `mapstructure:"help"`
	MetricType     string            `mapstructure:"type"`
	ConstantLabels prometheus.Labels `mapstructure:"const_labels"`
	TopicLabels    TopicLabels       `mapstructure:"topic_labels"`
	JSONField      string            `mapstructure:"json_field"`
}

// PrometheusDescription constructs description.
func (m *Metric) PrometheusDescription() *prometheus.Desc {
	varLabels := []string{"topic"}
	varLabels = append(varLabels, m.TopicLabels.KeysInOrder()...)

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

	viper.SetDefault("mqtt.port", 9641)
	viper.SetDefault("mqtt.timeout", "3s")

	viper.SetDefault("cache.expiration", "60s")
}
