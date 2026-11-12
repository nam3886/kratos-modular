package server

import (
	nethttp "net/http"

	"github.com/nam3886/kratos-modular/internal/platform/db"
	"github.com/nam3886/kratos-modular/internal/platform/mq"

	"github.com/go-kratos/kratos/contrib/otel/v3/metrics"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// Metrics counts and times every request and exposes the result for
// Prometheus at GET /metrics.
type Metrics struct {
	middleware middleware.Middleware
	handler    nethttp.Handler
}

// NewRegistry holds every metric of the application: runtime, connection
// pools and the outbox backlog here, requests and messaging elsewhere.
func NewRegistry(dbs *db.Manager) *prometheus.Registry {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	for _, d := range dbs.All() {
		registry.MustRegister(
			collectors.NewDBStatsCollector(d.Pool, d.Name),
			prometheus.NewGaugeFunc(prometheus.GaugeOpts{
				Name:        "outbox_pending_messages",
				Help:        "Events committed but not yet delivered to RabbitMQ.",
				ConstLabels: prometheus.Labels{"db_name": d.Name},
			}, func() float64 { return mq.Pending(d.Pool) }),
		)
	}
	return registry
}

func NewMetrics(registry *prometheus.Registry) (*Metrics, error) {
	exporter, err := otelprom.New(otelprom.WithRegisterer(registry))
	if err != nil {
		return nil, err
	}
	meter := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
		sdkmetric.WithView(metrics.DefaultSecondsHistogramView(metrics.DefaultServerSecondsHistogramName)),
	).Meter("server")
	requests, err := metrics.DefaultRequestsCounter(meter, metrics.DefaultServerRequestsCounterName)
	if err != nil {
		return nil, err
	}
	seconds, err := metrics.DefaultSecondsHistogram(meter, metrics.DefaultServerSecondsHistogramName)
	if err != nil {
		return nil, err
	}
	return &Metrics{
		middleware: metrics.Server(metrics.WithRequests(requests), metrics.WithSeconds(seconds)),
		handler:    promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
	}, nil
}
