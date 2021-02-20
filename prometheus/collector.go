package prometheus

import (
	"fmt"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/torilabs/mqtt-prometheus-exporter/log"
	"go.uber.org/zap"
)

// CollectorMetric represents configuration of observed metric.
type CollectorMetric interface {
	PrometheusDescription() *prometheus.Desc
	PrometheusValueType() prometheus.ValueType
	PrometheusName() string
}

// Collector is an extended interface of prometheus.Collector.
type Collector interface {
	prometheus.Collector
	Observe(metric CollectorMetric, topic string, v float64, labelValues []string)
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
func NewCollector(expiration time.Duration, possibleDescs []*prometheus.Desc) Collector {
	if len(possibleDescs) == 0 {
		log.Logger.Warn("No metrics are configured.")
	}
	return &memoryCachedCollector{
		cache:        gocache.New(expiration, expiration*10),
		descriptions: possibleDescs,
	}
}

func (c *memoryCachedCollector) Observe(metric CollectorMetric, topic string, v float64, labelValues []string) {
	m, err := prometheus.NewConstMetric(metric.PrometheusDescription(), metric.PrometheusValueType(), v, labelValues...)
	if err != nil {
		log.Logger.With(zap.Error(err)).Warnf("Creation of prometheus metric failed.")
		return
	}
	key := fmt.Sprintf("%s|%s", metric.PrometheusName(), topic)
	c.cache.SetDefault(key, &collectorEntry{m: m, ts: time.Now()})
}

func (c *memoryCachedCollector) Describe(ch chan<- *prometheus.Desc) {
	for i := range c.descriptions {
		ch <- c.descriptions[i]
	}
}

func (c *memoryCachedCollector) Collect(mc chan<- prometheus.Metric) {
	log.Logger.Debugf("Collecting. Returned '%d' metrics.", c.cache.ItemCount())
	for _, rawItem := range c.cache.Items() {
		item := rawItem.Object.(*collectorEntry)
		mc <- prometheus.NewMetricWithTimestamp(item.ts, item.m)
	}
}
