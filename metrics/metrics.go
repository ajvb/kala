package metrics

import (
	"fmt"
	"net/http"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/negroni"
)

// NewMetrics returns an instance of the Metrics struct for prometheus monitoring
func NewMetrics(version string) *Metrics {
	m := &Metrics{
		StartTime: prom.NewGauge(prom.GaugeOpts{
			Name:        "start_time_seconds",
			Help:        "heartbeat and version metrics",
			ConstLabels: prom.Labels{"version": version},
		}),
		JobsTotal: prom.NewCounterVec(prom.CounterOpts{
			Name: "jobs_total",
			Help: "All the jobs that are scheduled, started, succeeded, and errored",
		}, []string{"status"}),
		JobsDuration: prom.NewHistogram(prom.HistogramOpts{
			Name: "jobs_duration_seconds",
			Help: "Jobs duration histogram",
			Buckets: []float64{
				0.5, 1, 2, 3.5, 7, 15, 30, 60, 120, 240, 480,
			},
		}),
		RequestsTotal: prom.NewCounterVec(prom.CounterOpts{
			Name: "requests_total",
			Help: "Total number of requests based on code, method, path",
		}, []string{"code", "method", "path"}),
		RequestsDuration: prom.NewHistogramVec(prom.HistogramOpts{
			Name: "requests_duration_seconds",
			Help: "Total request time based on code, method, path",
			Buckets: []float64{
				0.1, 0.5, 1, 2,
			},
		}, []string{"code", "method", "path"}),
	}
	m.StartTime.Set(float64(time.Now().UnixNano()) / float64(time.Second))
	return m
}

type Metrics struct {
	StartTime        prom.Gauge
	JobsTotal        *prom.CounterVec
	JobsDuration     prom.Histogram
	RequestsTotal    *prom.CounterVec
	RequestsDuration *prom.HistogramVec
}

func (m *Metrics) Collectors() []prom.Collector {
	return []prom.Collector{
		m.StartTime,
		m.JobsTotal,
		m.JobsDuration,
		m.RequestsTotal,
		m.RequestsDuration,
	}
}

func (m *Metrics) Handler() http.Handler {
	for _, c := range m.Collectors() {
		prom.Register(c)
	}
	return promhttp.Handler()
}

func (m *Metrics) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	start := time.Now()
	next(rw, r)
	if r.URL.Path != "/metrics" {
		res := negroni.NewResponseWriter(rw)
		m.RequestsTotal.WithLabelValues(fmt.Sprintf("%d", res.Status()), r.Method, r.URL.Path).Inc()
		m.RequestsDuration.WithLabelValues(fmt.Sprintf("%d", res.Status()), r.Method, r.URL.Path).Observe(time.Since(start).Seconds())
	}
}
