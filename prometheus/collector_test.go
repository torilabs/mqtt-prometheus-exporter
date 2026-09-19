package prometheus

import (
	"testing"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/torilabs/mqtt-prometheus-exporter/config"
)

func gather(t *testing.T, c Collector) map[string]float64 {
	t.Helper()
	reg := prom.NewPedanticRegistry()
	if err := reg.Register(c); err != nil {
		t.Fatalf("register: %v", err)
	}
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	series := make(map[string]float64)
	for _, mf := range families {
		for _, m := range mf.GetMetric() {
			series[seriesID(m)] = m.GetGauge().GetValue()
		}
	}
	return series
}

func seriesID(m *dto.Metric) string {
	id := ""
	for _, l := range m.GetLabel() {
		id += l.GetName() + "=" + l.GetValue() + ","
	}
	return id
}

func jsonLabelsMetric() config.Metric {
	return config.Metric{
		PrometheusName: "temperature",
		MqttTopic:      "/home/+/state",
		MetricType:     "gauge",
		TopicLabels:    map[string]int{"device": 2},
		JSONField:      "temp",
		JSONLabels:     map[string]string{"room": "room"},
	}
}

func TestCollector_JSONLabelsProduceSeparateSeries(t *testing.T) {
	m := jsonLabelsMetric()
	c := NewCollector(time.Minute, []config.Metric{m})

	c.Observe(m, "/home/d1/state", 20, "/home/d1/state", "d1", "kitchen")
	c.Observe(m, "/home/d1/state", 21, "/home/d1/state", "d1", "bedroom")

	got := gather(t, c)
	want := map[string]float64{
		"device=d1,room=kitchen,topic=/home/d1/state,": 20,
		"device=d1,room=bedroom,topic=/home/d1/state,": 21,
	}
	if len(got) != len(want) {
		t.Fatalf("series = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("series %q = %v, want %v (all: %v)", k, got[k], v, got)
		}
	}
}

func TestCollector_IdenticalLabelsUpdateSeries(t *testing.T) {
	m := jsonLabelsMetric()
	c := NewCollector(time.Minute, []config.Metric{m})

	c.Observe(m, "/home/d1/state", 20, "/home/d1/state", "d1", "kitchen")
	c.Observe(m, "/home/d1/state", 25, "/home/d1/state", "d1", "kitchen")

	got := gather(t, c)
	if len(got) != 1 {
		t.Fatalf("series = %v, want a single one", got)
	}
	if v := got["device=d1,room=kitchen,topic=/home/d1/state,"]; v != 25 {
		t.Errorf("value = %v, want 25", v)
	}
}

func TestCollector_MetricWithoutJSONLabelsStillOverwritesPerTopic(t *testing.T) {
	m := config.Metric{PrometheusName: "plain", MqttTopic: "/a/#", MetricType: "gauge"}
	c := NewCollector(time.Minute, []config.Metric{m})

	c.Observe(m, "/a/1", 1, "/a/1")
	c.Observe(m, "/a/2", 2, "/a/2")
	c.Observe(m, "/a/1", 3, "/a/1")

	got := gather(t, c)
	if len(got) != 2 || got["topic=/a/1,"] != 3 || got["topic=/a/2,"] != 2 {
		t.Errorf("series = %v", got)
	}
}

func TestCacheKey_NoAmbiguityBetweenLabelValues(t *testing.T) {
	keys := map[string][]string{
		"split-a": {"a\x00b", "c"},
		"split-b": {"a", "b\x00c"},
		"split-c": {"a|b", "c"},
		"split-d": {"a", "b|c"},
		"split-e": {"1:a", ""},
		"split-f": {"", "1:a"},
	}
	seen := make(map[string]string)
	for name, lv := range keys {
		k := cacheKey("metric", "/t", lv)
		if other, dup := seen[k]; dup {
			t.Errorf("cache key of %s collides with %s", name, other)
		}
		seen[k] = name
	}
}
