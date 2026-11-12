package server

import (
	nethttp "net/http"

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

func NewMetrics() (*Metrics, error) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
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
