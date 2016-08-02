package storage

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/digitalocean/vulcan/bus"
	"github.com/digitalocean/vulcan/convert"
)

type mockIndexer struct {
	SampleIndexer
	count int
}

func (mi *mockIndexer) IndexSample(*bus.Sample) error {
	mi.count++
	return nil
}

func (mi mockIndexer) Describe(ch chan<- *prometheus.Desc) {}

func (mi mockIndexer) Collect(ch chan<- prometheus.Metric) {}

func TestCachingIndexer(t *testing.T) {
	mi := &mockIndexer{}
	ci := NewCachingIndexer(&CachingIndexerConfig{
		Indexer:     mi,
		MaxDuration: time.Minute,
	})
	start := time.Now()
	tests := []struct {
		metric      string
		at          time.Time
		last        time.Time
		insertCount int
	}{
		{
			metric:      "test1",
			at:          start,
			last:        start,
			insertCount: 1,
		},
		{
			metric:      "test1",
			at:          start.Add(time.Second),
			last:        start,
			insertCount: 1,
		},
		{
			metric:      "test2",
			at:          start.Add(time.Second),
			last:        start.Add(time.Second),
			insertCount: 2,
		},
		{
			metric:      "test1",
			at:          start.Add(time.Minute * 2),
			last:        start.Add(time.Minute * 2),
			insertCount: 3,
		},
		{
			metric:      "test1",
			at:          start.Add(time.Minute*2 + time.Second),
			last:        start.Add(time.Minute * 2),
			insertCount: 3,
		},
		{
			metric:      "test2",
			at:          start.Add(time.Minute * 2),
			last:        start.Add(time.Minute * 2),
			insertCount: 4,
		},
	}
	for _, test := range tests {
		s := &bus.Sample{
			Metric: bus.Metric{
				Name:   test.metric,
				Labels: map[string]string{},
			},
			Datapoint: bus.Datapoint{
				Timestamp: bus.Timestamp(0),
				Value:     0.0,
			},
		}
		err := ci.indexSample(s, test.at)
		if err != nil {
			t.Error(err)
		}
		key, err := convert.MetricToKey(s.Metric)
		if err != nil {
			t.Error(err)
		}
		last, ok := ci.LastSeen[key]
		if !ok {
			t.Errorf("expected metric key to exist in cache but not found %s", key)
		}
		if last != test.last {
			t.Errorf("expected last seen metric time to be %d but received %d", test.last, last)
		}
		if mi.count != test.insertCount {
			t.Errorf("expected number of inserts past the cache to be %d but received %d", test.insertCount, mi.count)
		}
	}
}
