package prometheus

import (
	"strconv"
	"strings"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/torilabs/mqtt-prometheus-exporter/config"
	"github.com/torilabs/mqtt-prometheus-exporter/log"
	"go.uber.org/zap"
)

// Collector is an extended interface of prometheus.Collector.
type Collector interface {
	prometheus.Collector
	Observe(metric config.Metric, topic string, v float64, labelValues ...string)
}

type memoryCachedCollector struct {
	cache        *gocache.Cache
	descriptions []*prometheus.Desc
}

type collectorEntry struct {
	m  prometheus.Metric
	ts time.Time
}

// NewCollector constructs collector for incoming prometheus metrics.
func NewCollector(expiration time.Duration, possibleMetrics []config.Metric) Collector {
	if len(possibleMetrics) == 0 {
		log.Logger.Warn("No metrics are configured.")
	}
	var descs []*prometheus.Desc
	for _, m := range possibleMetrics {
		descs = append(descs, m.PrometheusDescription())
	}
	return &memoryCachedCollector{
		cache:        gocache.New(expiration, expiration*10),
		descriptions: descs,
	}
}

func (c *memoryCachedCollector) Observe(metric config.Metric, topic string, v float64, labelValues ...string) {
	m, err := prometheus.NewConstMetric(metric.PrometheusDescription(), metric.PrometheusValueType(), v, labelValues...)
	if err != nil {
		log.Logger.With(zap.Error(err)).Warnf("Creation of prometheus metric failed.")
		return
	}
	c.cache.SetDefault(cacheKey(metric.PrometheusName, topic, labelValues), &collectorEntry{m: m, ts: time.Now()})
}

// cacheKey identifies a series by the metric name, the topic and all variable label values.
// Every part is length-prefixed so that different label values can never produce the same key.
func cacheKey(name, topic string, labelValues []string) string {
	var sb strings.Builder
	for _, part := range append([]string{name, topic}, labelValues...) {
		sb.WriteString(strconv.Itoa(len(part)))
		sb.WriteByte(':')
		sb.WriteString(part)
	}
	return sb.String()
}

func (c *memoryCachedCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range c.descriptions {
		ch <- desc
	}
}

func (c *memoryCachedCollector) Collect(mc chan<- prometheus.Metric) {
	log.Logger.Debugf("Collecting. Returned '%d' metrics.", c.cache.ItemCount())
	for _, rawItem := range c.cache.Items() {
		item := rawItem.Object.(*collectorEntry)
		mc <- prometheus.NewMetricWithTimestamp(item.ts, item.m)
	}
}
