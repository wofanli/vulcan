package scraper

import (
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/digitalocean/vulcan/config"

	"github.com/golang/protobuf/proto"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

type HTTPTarget struct {
	u url.URL
	i time.Duration
}

type HTTPTargetConfig struct {
	Interval time.Duration
	URL      url.URL
}

func NewHTTPTarget(config *HTTPTargetConfig) *HTTPTarget {
	return &HTTPTarget{
		u: config.URL,
		i: config.Interval,
	}
}

func (ht HTTPTarget) Equals(other Target) bool {
	ot, ok := other.(HTTPTarget)
	if !ok {
		return false
	}
	return ot.u == ht.u
}

func (ht HTTPTarget) Fetch() ([]*dto.MetricFamily, error) {
	at := time.Now() // timestamp metrics with time scraper initiated
	fam, err := ht.fetch()
	if err != nil {
		return fam, err
	}
	timestamp(fam, at)
	return fam, nil
}

func (ht HTTPTarget) Interval() time.Duration {
	return ht.i
}

func annotate(fams []*dto.MetricFamily, target config.Target) {
	for _, f := range fams {
		for _, m := range f.Metric {
			m.Label = append(m.Label, &dto.LabelPair{
				Name:  proto.String("job"),
				Value: proto.String(target.Job),
			})
			m.Label = append(m.Label, &dto.LabelPair{
				Name:  proto.String("instance"),
				Value: proto.String(target.Instance),
			})
		}
	}
}

func (ht HTTPTarget) fetch() ([]*dto.MetricFamily, error) {
	resp, err := http.Get(ht.u.String())
	if err != nil {
		return []*dto.MetricFamily{}, err
	}
	defer resp.Body.Close()
	// todo check return codes
	return parse(resp.Body, resp.Header)
}

func parse(in io.Reader, header http.Header) ([]*dto.MetricFamily, error) {
	dec := expfmt.NewDecoder(in, expfmt.Negotiate(header))
	fams := []*dto.MetricFamily{}
	var err error
	for {
		var f dto.MetricFamily
		err = dec.Decode(&f)
		if err != nil {
			break
		}
		fams = append(fams, &f)
	}
	if err == io.EOF {
		err = nil
	}
	return fams, err
}

func timestamp(fams []*dto.MetricFamily, at time.Time) {
	timestampMs := proto.Int64(at.UnixNano() / 1e6)
	for _, f := range fams {
		for _, m := range f.Metric {
			if m.TimestampMs == nil {
				m.TimestampMs = timestampMs
			}
		}
	}
}
