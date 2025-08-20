package monitoring

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Config holds Prometheus monitoring configuration
type Config struct {
	Enabled     bool   `json:"enabled" mapstructure:"enabled"`
	Namespace   string `json:"namespace" mapstructure:"namespace"`
	MetricsPath string `json:"metrics_path" mapstructure:"metrics_path"`
	ListenAddr  string `json:"listen_addr" mapstructure:"listen_addr"`
}

// Metrics holds all Prometheus metrics
type Metrics struct {
	// HTTP metrics
	HTTPRequests     *prometheus.CounterVec
	HTTPDuration     *prometheus.HistogramVec
	HTTPRequestSize  *prometheus.HistogramVec
	HTTPResponseSize *prometheus.HistogramVec

	// Database metrics
	DBConnections   *prometheus.GaugeVec
	DBQueries       *prometheus.CounterVec
	DBQueryDuration *prometheus.HistogramVec

	// Message broker metrics
	MBMessages    *prometheus.CounterVec
	MBDuration    *prometheus.HistogramVec
	MBConnections *prometheus.GaugeVec

	// Cache metrics
	CacheOperations *prometheus.CounterVec
	CacheDuration   *prometheus.HistogramVec
	CacheHitRate    *prometheus.GaugeVec

	// Application metrics
	AppInfo         *prometheus.GaugeVec
	UserSessions    *prometheus.GaugeVec
	ActiveUsers     *prometheus.GaugeVec
	BusinessMetrics *prometheus.CounterVec

	// System metrics
	GoInfo       *prometheus.GaugeVec
	GoMemstats   prometheus.Collector
	GoGoroutines prometheus.Gauge
	ProcessInfo  *prometheus.GaugeVec

	registry *prometheus.Registry
}

// PrometheusMonitor handles all Prometheus monitoring
type PrometheusMonitor struct {
	config   *Config
	metrics  *Metrics
	registry *prometheus.Registry
}

// NewPrometheusMonitor creates a new Prometheus monitor
func NewPrometheusMonitor(config *Config) (*PrometheusMonitor, error) {
	if !config.Enabled {
		return &PrometheusMonitor{config: config}, nil
	}

	registry := prometheus.NewRegistry()

	metrics := &Metrics{
		// HTTP metrics
		HTTPRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: config.Namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status_code"},
		),
		HTTPDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: config.Namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "endpoint", "status_code"},
		),
		HTTPRequestSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: config.Namespace,
				Name:      "http_request_size_bytes",
				Help:      "HTTP request size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "endpoint"},
		),
		HTTPResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: config.Namespace,
				Name:      "http_response_size_bytes",
				Help:      "HTTP response size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "endpoint", "status_code"},
		),

		// Database metrics
		DBConnections: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Name:      "database_connections",
				Help:      "Number of database connections",
			},
			[]string{"database", "state"},
		),
		DBQueries: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: config.Namespace,
				Name:      "database_queries_total",
				Help:      "Total number of database queries",
			},
			[]string{"database", "operation", "status"},
		),
		DBQueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: config.Namespace,
				Name:      "database_query_duration_seconds",
				Help:      "Database query duration in seconds",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10},
			},
			[]string{"database", "operation"},
		),

		// Message broker metrics
		MBMessages: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: config.Namespace,
				Name:      "message_broker_messages_total",
				Help:      "Total number of message broker messages",
			},
			[]string{"driver", "operation", "topic", "status"},
		),
		MBDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: config.Namespace,
				Name:      "message_broker_operation_duration_seconds",
				Help:      "Message broker operation duration in seconds",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5},
			},
			[]string{"driver", "operation", "topic"},
		),
		MBConnections: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Name:      "message_broker_connections",
				Help:      "Number of message broker connections",
			},
			[]string{"driver", "state"},
		),

		// Cache metrics
		CacheOperations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: config.Namespace,
				Name:      "cache_operations_total",
				Help:      "Total number of cache operations",
			},
			[]string{"operation", "status"},
		),
		CacheDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: config.Namespace,
				Name:      "cache_operation_duration_seconds",
				Help:      "Cache operation duration in seconds",
				Buckets:   []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1},
			},
			[]string{"operation"},
		),
		CacheHitRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Name:      "cache_hit_rate",
				Help:      "Cache hit rate percentage",
			},
			[]string{"cache_type"},
		),

		// Application metrics
		AppInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Name:      "app_info",
				Help:      "Application information",
			},
			[]string{"version", "environment", "service"},
		),
		UserSessions: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Name:      "user_sessions_active",
				Help:      "Number of active user sessions",
			},
			[]string{"session_type"},
		),
		ActiveUsers: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Name:      "users_active",
				Help:      "Number of active users",
			},
			[]string{"time_window"},
		),
		BusinessMetrics: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: config.Namespace,
				Name:      "business_events_total",
				Help:      "Total number of business events",
			},
			[]string{"event_type", "status"},
		),

		// System metrics
		GoInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Name:      "go_info",
				Help:      "Go runtime information",
			},
			[]string{"version"},
		),
		GoMemstats: prometheus.NewGoCollector(),
		GoGoroutines: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Name:      "go_goroutines",
			Help:      "Number of goroutines",
		}),
		ProcessInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Name:      "process_info",
				Help:      "Process information",
			},
			[]string{"pid"},
		),

		registry: registry,
	}

	// Register all metrics
	registry.MustRegister(
		metrics.HTTPRequests,
		metrics.HTTPDuration,
		metrics.HTTPRequestSize,
		metrics.HTTPResponseSize,
		metrics.DBConnections,
		metrics.DBQueries,
		metrics.DBQueryDuration,
		metrics.MBMessages,
		metrics.MBDuration,
		metrics.MBConnections,
		metrics.CacheOperations,
		metrics.CacheDuration,
		metrics.CacheHitRate,
		metrics.AppInfo,
		metrics.UserSessions,
		metrics.ActiveUsers,
		metrics.BusinessMetrics,
		metrics.GoInfo,
		metrics.GoMemstats,
		metrics.GoGoroutines,
		metrics.ProcessInfo,
	)

	monitor := &PrometheusMonitor{
		config:   config,
		metrics:  metrics,
		registry: registry,
	}

	return monitor, nil
}

// GetMetrics returns the metrics instance
func (m *PrometheusMonitor) GetMetrics() *Metrics {
	return m.metrics
}

// GetHandler returns the Prometheus metrics HTTP handler
func (m *PrometheusMonitor) GetHandler() http.Handler {
	if !m.config.Enabled {
		return http.NotFoundHandler()
	}
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// GinMiddleware returns a Gin middleware for HTTP metrics
func (m *PrometheusMonitor) GinMiddleware() gin.HandlerFunc {
	if !m.config.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		start := time.Now()

		// Get request size
		requestSize := float64(0)
		if c.Request.ContentLength > 0 {
			requestSize = float64(c.Request.ContentLength)
		}

		// Process request
		c.Next()

		// Calculate metrics
		duration := time.Since(start).Seconds()
		statusCode := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = "unknown"
		}

		// Update metrics
		m.metrics.HTTPRequests.WithLabelValues(method, endpoint, statusCode).Inc()
		m.metrics.HTTPDuration.WithLabelValues(method, endpoint, statusCode).Observe(duration)

		if requestSize > 0 {
			m.metrics.HTTPRequestSize.WithLabelValues(method, endpoint).Observe(requestSize)
		}

		responseSize := float64(c.Writer.Size())
		if responseSize > 0 {
			m.metrics.HTTPResponseSize.WithLabelValues(method, endpoint, statusCode).Observe(responseSize)
		}
	}
}

// RecordDBOperation records database operation metrics
func (m *PrometheusMonitor) RecordDBOperation(database, operation, status string, duration time.Duration) {
	if !m.config.Enabled {
		return
	}

	m.metrics.DBQueries.WithLabelValues(database, operation, status).Inc()
	m.metrics.DBQueryDuration.WithLabelValues(database, operation).Observe(duration.Seconds())
}

// RecordDBConnections records database connection metrics
func (m *PrometheusMonitor) RecordDBConnections(database, state string, count int) {
	if !m.config.Enabled {
		return
	}

	m.metrics.DBConnections.WithLabelValues(database, state).Set(float64(count))
}

// RecordMessageBrokerOperation records message broker operation metrics
func (m *PrometheusMonitor) RecordMessageBrokerOperation(driver, operation, topic, status string, duration time.Duration) {
	if !m.config.Enabled {
		return
	}

	m.metrics.MBMessages.WithLabelValues(driver, operation, topic, status).Inc()
	m.metrics.MBDuration.WithLabelValues(driver, operation, topic).Observe(duration.Seconds())
}

// RecordMessageBrokerConnections records message broker connection metrics
func (m *PrometheusMonitor) RecordMessageBrokerConnections(driver, state string, count int) {
	if !m.config.Enabled {
		return
	}

	m.metrics.MBConnections.WithLabelValues(driver, state).Set(float64(count))
}

// RecordCacheOperation records cache operation metrics
func (m *PrometheusMonitor) RecordCacheOperation(operation, status string, duration time.Duration) {
	if !m.config.Enabled {
		return
	}

	m.metrics.CacheOperations.WithLabelValues(operation, status).Inc()
	m.metrics.CacheDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordCacheHitRate records cache hit rate metrics
func (m *PrometheusMonitor) RecordCacheHitRate(cacheType string, hitRate float64) {
	if !m.config.Enabled {
		return
	}

	m.metrics.CacheHitRate.WithLabelValues(cacheType).Set(hitRate)
}

// RecordBusinessEvent records business event metrics
func (m *PrometheusMonitor) RecordBusinessEvent(eventType, status string) {
	if !m.config.Enabled {
		return
	}

	m.metrics.BusinessMetrics.WithLabelValues(eventType, status).Inc()
}

// RecordUserSessions records user session metrics
func (m *PrometheusMonitor) RecordUserSessions(sessionType string, count int) {
	if !m.config.Enabled {
		return
	}

	m.metrics.UserSessions.WithLabelValues(sessionType).Set(float64(count))
}

// RecordActiveUsers records active user metrics
func (m *PrometheusMonitor) RecordActiveUsers(timeWindow string, count int) {
	if !m.config.Enabled {
		return
	}

	m.metrics.ActiveUsers.WithLabelValues(timeWindow).Set(float64(count))
}

// SetAppInfo sets application information metrics
func (m *PrometheusMonitor) SetAppInfo(version, environment, service string) {
	if !m.config.Enabled {
		return
	}

	m.metrics.AppInfo.WithLabelValues(version, environment, service).Set(1)
}

// HealthCheck checks if Prometheus is healthy
func (m *PrometheusMonitor) HealthCheck(ctx context.Context) error {
	if !m.config.Enabled {
		return nil
	}

	// Simple health check - just verify metrics are accessible
	gatherer := prometheus.Gatherers{m.registry}
	_, err := gatherer.Gather()
	return err
}

// Close performs cleanup
func (m *PrometheusMonitor) Close() error {
	// Prometheus doesn't require explicit cleanup
	return nil
}

// StartMetricsServer starts a separate HTTP server for metrics
func (m *PrometheusMonitor) StartMetricsServer() error {
	if !m.config.Enabled {
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle(m.config.MetricsPath, m.GetHandler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:    m.config.ListenAddr,
		Handler: mux,
	}

	return server.ListenAndServe()
}
