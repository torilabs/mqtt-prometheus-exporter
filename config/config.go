package config

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

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

// JSONLabels maps a prometheus label name to a dotted path of a property in the JSON message.
type JSONLabels map[string]string

// KeysInOrder sort keys always the same way.
func (jl JSONLabels) KeysInOrder() []string {
	keys := make([]string, 0, len(jl))
	for k := range jl {
		keys = append(keys, k)
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
	JSONLabels     JSONLabels        `mapstructure:"json_labels"`
}

// topicLabelName is the name of the label always added to metrics.
const topicLabelName = "topic"

var labelNameRegexp = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// ValidateLabels checks that json_labels can be used together with the rest of the metric configuration.
func (m *Metric) ValidateLabels() error {
	if len(m.JSONLabels) == 0 {
		return nil
	}
	if m.JSONField == "" {
		return errors.New("json_labels require json_field to be set")
	}
	for _, name := range m.JSONLabels.KeysInOrder() {
		if !labelNameRegexp.MatchString(name) || strings.HasPrefix(name, "__") {
			return fmt.Errorf("json label name %q is not a valid prometheus label name", name)
		}
		if strings.TrimSpace(m.JSONLabels[name]) == "" {
			return fmt.Errorf("json label %q has an empty JSON property path", name)
		}
		if name == topicLabelName {
			return fmt.Errorf("json label name %q collides with the built-in %q label", name, topicLabelName)
		}
		if _, ok := m.ConstantLabels[name]; ok {
			return fmt.Errorf("json label name %q collides with a constant label", name)
		}
		if _, ok := m.TopicLabels[name]; ok {
			return fmt.Errorf("json label name %q collides with a topic label", name)
		}
	}
	return nil
}

// PrometheusDescription constructs description.
func (m *Metric) PrometheusDescription() *prometheus.Desc {
	varLabels := []string{topicLabelName}
	varLabels = append(varLabels, m.TopicLabels.KeysInOrder()...)
	varLabels = append(varLabels, m.JSONLabels.KeysInOrder()...)

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
		return cfg, fmt.Errorf("failed to read configuration: %w", err)
	}

	setDefaults()

	if err := viper.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to deserialize config: %w", err)
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
