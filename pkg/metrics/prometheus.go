// prometheus.go
package metrics

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MetricsService defines the interface for interacting with metrics
type MetricsService interface {
	// Counter metrics
	IncrementCounter(name string, labels map[string]string)
	AddToCounter(name string, value float64, labels map[string]string)

	// Gauge metrics
	SetGauge(name string, value float64, labels map[string]string)
	IncrementGauge(name string, labels map[string]string)
	DecrementGauge(name string, labels map[string]string)

	// Histogram metrics
	ObserveHistogram(name string, value float64, labels map[string]string)

	// Summary metrics
	ObserveSummary(name string, value float64, labels map[string]string)

	// Timer functionality
	StartTimer(name string, labels map[string]string) func()

	// HTTP handler for metrics endpoint
	MetricsHandler() http.Handler

	// Middleware for REST API and gRPC
	HTTPMiddleware(next http.Handler) http.Handler
	RESTMiddleware(next http.HandlerFunc) http.HandlerFunc
	GRPCUnaryInterceptor() grpc.UnaryServerInterceptor
	GRPCStreamInterceptor() grpc.StreamServerInterceptor
}

// PrometheusMetrics implements the MetricsService interface using Prometheus
type PrometheusMetrics struct {
	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
	summaries  map[string]*prometheus.SummaryVec
	mu         sync.RWMutex
	namespace  string
}

// NewPrometheusMetrics creates a new PrometheusMetrics instance
func NewPrometheusMetrics(namespace string) *PrometheusMetrics {
	prometheus.DefaultRegisterer.MustRegister(prometheus.NewBuildInfoCollector())

	return &PrometheusMetrics{
		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
		summaries:  make(map[string]*prometheus.SummaryVec),
		namespace:  namespace,
	}
}

// getCounter returns an existing counter or creates a new one
func (p *PrometheusMetrics) getCounter(name string, labelNames []string) *prometheus.CounterVec {
	p.mu.RLock()
	counter, exists := p.counters[name]
	p.mu.RUnlock()

	if !exists {
		p.mu.Lock()
		defer p.mu.Unlock()

		// Double check in case another goroutine created it while we were waiting for the lock
		counter, exists = p.counters[name]
		if !exists {
			counter = promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: p.namespace,
					Name:      name,
					Help:      name + " counter",
				},
				labelNames,
			)
			p.counters[name] = counter
		}
	}

	return counter
}

// getGauge returns an existing gauge or creates a new one
func (p *PrometheusMetrics) getGauge(name string, labelNames []string) *prometheus.GaugeVec {
	p.mu.RLock()
	gauge, exists := p.gauges[name]
	p.mu.RUnlock()

	if !exists {
		p.mu.Lock()
		defer p.mu.Unlock()

		gauge, exists = p.gauges[name]
		if !exists {
			gauge = promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: p.namespace,
					Name:      name,
					Help:      name + " gauge",
				},
				labelNames,
			)
			p.gauges[name] = gauge
		}
	}

	return gauge
}

// getHistogram returns an existing histogram or creates a new one
func (p *PrometheusMetrics) getHistogram(name string, labelNames []string) *prometheus.HistogramVec {
	p.mu.RLock()
	histogram, exists := p.histograms[name]
	p.mu.RUnlock()

	if !exists {
		p.mu.Lock()
		defer p.mu.Unlock()

		histogram, exists = p.histograms[name]
		if !exists {
			histogram = promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: p.namespace,
					Name:      name,
					Help:      name + " histogram",
					Buckets:   prometheus.DefBuckets,
				},
				labelNames,
			)
			p.histograms[name] = histogram
		}
	}

	return histogram
}

// getSummary returns an existing summary or creates a new one
func (p *PrometheusMetrics) getSummary(name string, labelNames []string) *prometheus.SummaryVec {
	p.mu.RLock()
	summary, exists := p.summaries[name]
	p.mu.RUnlock()

	if !exists {
		p.mu.Lock()
		defer p.mu.Unlock()

		summary, exists = p.summaries[name]
		if !exists {
			summary = promauto.NewSummaryVec(
				prometheus.SummaryOpts{
					Namespace:  p.namespace,
					Name:       name,
					Help:       name + " summary",
					Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
				},
				labelNames,
			)
			p.summaries[name] = summary
		}
	}

	return summary
}

// IncrementCounter increments a counter by 1
func (p *PrometheusMetrics) IncrementCounter(name string, labels map[string]string) {
	p.AddToCounter(name, 1, labels)
}

// AddToCounter adds a value to a counter
func (p *PrometheusMetrics) AddToCounter(name string, value float64, labels map[string]string) {
	labelNames := make([]string, 0, len(labels))
	labelValues := make([]string, 0, len(labels))

	for k := range labels {
		labelNames = append(labelNames, k)
	}

	counter := p.getCounter(name, labelNames)

	for _, k := range labelNames {
		labelValues = append(labelValues, labels[k])
	}

	counter.WithLabelValues(labelValues...).Add(value)
}

// SetGauge sets a gauge to a value
func (p *PrometheusMetrics) SetGauge(name string, value float64, labels map[string]string) {
	labelNames := make([]string, 0, len(labels))
	labelValues := make([]string, 0, len(labels))

	for k := range labels {
		labelNames = append(labelNames, k)
	}

	gauge := p.getGauge(name, labelNames)

	for _, k := range labelNames {
		labelValues = append(labelValues, labels[k])
	}

	gauge.WithLabelValues(labelValues...).Set(value)
}

// IncrementGauge increments a gauge by 1
func (p *PrometheusMetrics) IncrementGauge(name string, labels map[string]string) {
	labelNames := make([]string, 0, len(labels))
	labelValues := make([]string, 0, len(labels))

	for k := range labels {
		labelNames = append(labelNames, k)
	}

	gauge := p.getGauge(name, labelNames)

	for _, k := range labelNames {
		labelValues = append(labelValues, labels[k])
	}

	gauge.WithLabelValues(labelValues...).Inc()
}

// DecrementGauge decrements a gauge by 1
func (p *PrometheusMetrics) DecrementGauge(name string, labels map[string]string) {
	labelNames := make([]string, 0, len(labels))
	labelValues := make([]string, 0, len(labels))

	for k := range labels {
		labelNames = append(labelNames, k)
	}

	gauge := p.getGauge(name, labelNames)

	for _, k := range labelNames {
		labelValues = append(labelValues, labels[k])
	}

	gauge.WithLabelValues(labelValues...).Dec()
}

// ObserveHistogram observes a value in a histogram
func (p *PrometheusMetrics) ObserveHistogram(name string, value float64, labels map[string]string) {
	labelNames := make([]string, 0, len(labels))
	labelValues := make([]string, 0, len(labels))

	for k := range labels {
		labelNames = append(labelNames, k)
	}

	histogram := p.getHistogram(name, labelNames)

	for _, k := range labelNames {
		labelValues = append(labelValues, labels[k])
	}

	histogram.WithLabelValues(labelValues...).Observe(value)
}

// ObserveSummary observes a value in a summary
func (p *PrometheusMetrics) ObserveSummary(name string, value float64, labels map[string]string) {
	labelNames := make([]string, 0, len(labels))
	labelValues := make([]string, 0, len(labels))

	for k := range labels {
		labelNames = append(labelNames, k)
	}

	summary := p.getSummary(name, labelNames)

	for _, k := range labelNames {
		labelValues = append(labelValues, labels[k])
	}

	summary.WithLabelValues(labelValues...).Observe(value)
}

// StartTimer starts a timer and returns a function to observe the duration
func (p *PrometheusMetrics) StartTimer(name string, labels map[string]string) func() {
	start := time.Now()
	return func() {
		duration := time.Since(start).Seconds()
		p.ObserveHistogram(name+"_duration_seconds", duration, labels)
	}
}

// MetricsHandler returns an HTTP handler for the metrics endpoint
func (p *PrometheusMetrics) MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// HTTPMiddleware provides a middleware for generic HTTP handlers
func (p *PrometheusMetrics) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		labels := map[string]string{
			"method": r.Method,
			"path":   r.URL.Path,
		}

		// Track request count
		p.IncrementCounter("http_requests_total", labels)

		// Track in-flight requests
		p.IncrementGauge("http_in_flight_requests", labels)
		defer p.DecrementGauge("http_in_flight_requests", labels)

		// Track request duration
		timer := p.StartTimer("http_request_duration_seconds", labels)
		defer timer()

		// Wrap response writer to capture status code
		wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrappedWriter, r)

		// Add status code label for completed requests
		labels["status"] = http.StatusText(wrappedWriter.statusCode)
		p.IncrementCounter("http_requests_completed_total", labels)
	})
}

// RESTMiddleware provides a middleware specifically for REST API handlers
func (p *PrometheusMetrics) RESTMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		labels := map[string]string{
			"method": r.Method,
			"path":   r.URL.Path,
			"type":   "rest",
		}

		// Track request count
		p.IncrementCounter("api_requests_total", labels)

		// Track in-flight requests
		p.IncrementGauge("api_in_flight_requests", labels)
		defer p.DecrementGauge("api_in_flight_requests", labels)

		// Track request duration
		timer := p.StartTimer("api_request_duration_seconds", labels)
		defer timer()

		// Wrap response writer to capture status code
		wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next(wrappedWriter, r)

		// Add status code label for completed requests
		labels["status"] = http.StatusText(wrappedWriter.statusCode)
		p.IncrementCounter("api_requests_completed_total", labels)
	}
}

// GRPCUnaryInterceptor provides an interceptor for unary gRPC methods
func (p *PrometheusMetrics) GRPCUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		labels := map[string]string{
			"method": info.FullMethod,
			"type":   "unary",
		}

		// Track request count
		p.IncrementCounter("grpc_requests_total", labels)

		// Track in-flight requests
		p.IncrementGauge("grpc_in_flight_requests", labels)
		defer p.DecrementGauge("grpc_in_flight_requests", labels)

		// Track request duration
		timer := p.StartTimer("grpc_request_duration_seconds", labels)
		defer timer()

		// Execute handler
		resp, err := handler(ctx, req)

		// Add status code label for completed requests
		if err != nil {
			st, ok := status.FromError(err)
			if ok {
				labels["status"] = st.Code().String()
			} else {
				labels["status"] = codes.Unknown.String()
			}
		} else {
			labels["status"] = codes.OK.String()
		}
		p.IncrementCounter("grpc_requests_completed_total", labels)

		return resp, err
	}
}

// GRPCStreamInterceptor provides an interceptor for streaming gRPC methods
func (p *PrometheusMetrics) GRPCStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		labels := map[string]string{
			"method": info.FullMethod,
			"type":   "stream",
		}

		// Track request count
		p.IncrementCounter("grpc_stream_requests_total", labels)

		// Track in-flight requests
		p.IncrementGauge("grpc_stream_in_flight_requests", labels)
		defer p.DecrementGauge("grpc_stream_in_flight_requests", labels)

		// Track request duration
		timer := p.StartTimer("grpc_stream_duration_seconds", labels)
		defer timer()

		// Execute handler
		err := handler(srv, ss)

		// Add status code label for completed requests
		if err != nil {
			st, ok := status.FromError(err)
			if ok {
				labels["status"] = st.Code().String()
			} else {
				labels["status"] = codes.Unknown.String()
			}
		} else {
			labels["status"] = codes.OK.String()
		}
		p.IncrementCounter("grpc_stream_requests_completed_total", labels)

		return err
	}
}

// responseWriter is a wrapper for http.ResponseWriter that captures the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// SetupMetricsServer sets up and starts a metrics server
func SetupMetricsServer(metricsService MetricsService, addr string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", metricsService.MetricsHandler())

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	return server
}

// RegisterWithRESTServer adds Prometheus middleware to a REST API server
func RegisterWithRESTServer(metrics MetricsService, mux *http.ServeMux, patterns map[string]http.HandlerFunc) {
	for pattern, handler := range patterns {
		mux.HandleFunc(pattern, metrics.RESTMiddleware(handler))
	}
}

// RegisterWithGRPCServer adds Prometheus interceptors to a gRPC server
func RegisterWithGRPCServer(metrics MetricsService) []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.UnaryInterceptor(metrics.GRPCUnaryInterceptor()),
		grpc.StreamInterceptor(metrics.GRPCStreamInterceptor()),
	}
}
