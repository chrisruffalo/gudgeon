package metrics

import (
	"testing"
)

func TestMetric(t *testing.T) {
	counter := &Metric{}
	counter.Inc(1)
	if int64(1) != counter.Value() {
		t.Errorf("Expected 1 but got %d", counter.Value())
	}
	counter.Set(250)
	if int64(250) != counter.Value() {
		t.Errorf("Expected 250 but got %d", counter.Value())
	}
	counter.Clear()
	if int64(0) != counter.Value() {
		t.Errorf("Expected 0 but got %d", counter.Value())
	}
	counter.Set(250495323123)
	if int64(250495323123) != counter.Value() {
		t.Errorf("Expected 250495323123 but got %d", counter.Value())
	}
}

func TestMetricMap(t *testing.T) {
	ms := &metrics{
		metricsMap: make(map[string]*Metric),
	}

	ms.Get("total-time").Inc(44444)
	if int64(44444) != ms.Get("total-time").Value() {
		t.Errorf("Expected 44444 but got %d", ms.Get("total-time").Value())
	}

	ma := ms.Get("tt")
	mb := ms.Get("tt")
	ma.Inc(2000)
	if int64(2000) != ms.Get("tt").Value() {
		t.Errorf("Expected 2000 but got %d", ms.Get("tt").Value())
	}
	if int64(2000) != mb.Value() {
		t.Errorf("Expected (mb=)2000 but got %d", mb.Value())
	}
}
