package prometheus

import (
	"fmt"
	gocache "github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/torilabs/mqtt-prometheus-exporter/config"
	"github.com/torilabs/mqtt-prometheus-exporter/log"
	"go.uber.org/zap"
	"time"
)

// Collector is an extended interface of prometheus.Collector
type Collector interface {
	prometheus.Collector
	Observe(metric config.Metric, topic string, v float64)
}

type memoryCachedCollector struct {
	cache        *gocache.Cache
	descriptions []*prometheus.Desc
}

type collectorEntry struct {
	m  prometheus.Metric
	ts time.Time
}

// NewCollector constructs collector for incoming prometheus metrics
func NewCollector(defaultTimeout time.Duration, possibleMetrics []config.Metric) Collector {
	var descs []*prometheus.Desc
	for _, m := range possibleMetrics {
		descs = append(descs, m.PrometheusDescription())
	}
	return &memoryCachedCollector{
		cache:        gocache.New(defaultTimeout, defaultTimeout*10),
		descriptions: descs,
	}
}

func (c *memoryCachedCollector) Observe(metric config.Metric, topic string, v float64) {
	m, err := prometheus.NewConstMetric(metric.PrometheusDescription(), metric.PrometheusValueType(), v, topic)
	if err != nil {
		log.Logger.With(zap.Error(err)).Errorf("creation of prometheus metric failed")
		return
	}
	key := fmt.Sprintf("%s|%s", metric.PrometheusName, topic)
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
