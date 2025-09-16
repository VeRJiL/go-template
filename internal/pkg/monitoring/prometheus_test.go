package monitoring

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	t.Run("should create config with correct values", func(t *testing.T) {
		config := &Config{
			Enabled:     true,
			Namespace:   "myapp",
			MetricsPath: "/metrics",
			ListenAddr:  ":9090",
		}

		assert.True(t, config.Enabled)
		assert.Equal(t, "myapp", config.Namespace)
		assert.Equal(t, "/metrics", config.MetricsPath)
		assert.Equal(t, ":9090", config.ListenAddr)
	})
}

func TestNewPrometheusMonitor(t *testing.T) {
	t.Run("should create monitor when enabled", func(t *testing.T) {
		config := &Config{
			Enabled:     true,
			Namespace:   "test_app",
			MetricsPath: "/metrics",
			ListenAddr:  ":9090",
		}

		monitor, err := NewPrometheusMonitor(config)

		require.NoError(t, err)
		assert.NotNil(t, monitor)
		assert.NotNil(t, monitor.config)
		assert.NotNil(t, monitor.metrics)
		assert.NotNil(t, monitor.registry)
		assert.Equal(t, config, monitor.config)
	})

	t.Run("should create disabled monitor when not enabled", func(t *testing.T) {
		config := &Config{
			Enabled:     false,
			Namespace:   "test_app",
			MetricsPath: "/metrics",
			ListenAddr:  ":9090",
		}

		monitor, err := NewPrometheusMonitor(config)

		require.NoError(t, err)
		assert.NotNil(t, monitor)
		assert.Equal(t, config, monitor.config)
		assert.Nil(t, monitor.metrics)
		assert.Nil(t, monitor.registry)
	})

	t.Run("should register all metrics when enabled", func(t *testing.T) {
		config := &Config{
			Enabled:   true,
			Namespace: "test",
		}

		monitor, err := NewPrometheusMonitor(config)

		require.NoError(t, err)
		assert.NotNil(t, monitor.metrics.HTTPRequests)
		assert.NotNil(t, monitor.metrics.HTTPDuration)
		assert.NotNil(t, monitor.metrics.HTTPRequestSize)
		assert.NotNil(t, monitor.metrics.HTTPResponseSize)
		assert.NotNil(t, monitor.metrics.DBConnections)
		assert.NotNil(t, monitor.metrics.DBQueries)
		assert.NotNil(t, monitor.metrics.DBQueryDuration)
		assert.NotNil(t, monitor.metrics.MBMessages)
		assert.NotNil(t, monitor.metrics.MBDuration)
		assert.NotNil(t, monitor.metrics.MBConnections)
		assert.NotNil(t, monitor.metrics.CacheOperations)
		assert.NotNil(t, monitor.metrics.CacheDuration)
		assert.NotNil(t, monitor.metrics.CacheHitRate)
		assert.NotNil(t, monitor.metrics.AppInfo)
		assert.NotNil(t, monitor.metrics.UserSessions)
		assert.NotNil(t, monitor.metrics.ActiveUsers)
		assert.NotNil(t, monitor.metrics.BusinessMetrics)
		assert.NotNil(t, monitor.metrics.GoInfo)
		assert.NotNil(t, monitor.metrics.GoMemstats)
		assert.NotNil(t, monitor.metrics.GoGoroutines)
		assert.NotNil(t, monitor.metrics.ProcessInfo)
	})

	t.Run("should use correct metric names with namespace", func(t *testing.T) {
		config := &Config{
			Enabled:   true,
			Namespace: "myapp",
		}

		monitor, err := NewPrometheusMonitor(config)

		require.NoError(t, err)

		// Check that metrics have the correct namespace
		// We can't directly access the metric name, but we can verify through gathering
		gatherer := prometheus.Gatherers{monitor.registry}
		families, err := gatherer.Gather()
		require.NoError(t, err)

		// Check that at least some metrics exist with the correct namespace
		found := false
		for _, family := range families {
			if strings.HasPrefix(*family.Name, "myapp_") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find metrics with namespace prefix")
	})
}

func TestPrometheusMonitorGetters(t *testing.T) {
	config := &Config{
		Enabled:   true,
		Namespace: "test",
	}

	monitor, err := NewPrometheusMonitor(config)
	require.NoError(t, err)

	t.Run("should return metrics", func(t *testing.T) {
		metrics := monitor.GetMetrics()
		assert.NotNil(t, metrics)
		assert.Equal(t, monitor.metrics, metrics)
	})

	t.Run("should return HTTP handler when enabled", func(t *testing.T) {
		handler := monitor.GetHandler()
		assert.NotNil(t, handler)

		// Test that the handler works
		req, err := http.NewRequest("GET", "/metrics", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "# HELP")
	})

	t.Run("should return NotFound handler when disabled", func(t *testing.T) {
		disabledConfig := &Config{Enabled: false}
		disabledMonitor, err := NewPrometheusMonitor(disabledConfig)
		require.NoError(t, err)

		handler := disabledMonitor.GetHandler()

		req, err := http.NewRequest("GET", "/metrics", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

func TestGinMiddleware(t *testing.T) {
	t.Run("should record HTTP metrics when enabled", func(t *testing.T) {
		config := &Config{
			Enabled:   true,
			Namespace: "test",
		}

		monitor, err := NewPrometheusMonitor(config)
		require.NoError(t, err)

		// Setup Gin engine with middleware
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(monitor.GinMiddleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "test"})
		})

		// Make a request
		req, err := http.NewRequest("GET", "/test", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)

		// Check that metrics were recorded
		requestsCount := testutil.ToFloat64(monitor.metrics.HTTPRequests.WithLabelValues("GET", "/test", "200"))
		assert.Equal(t, float64(1), requestsCount)

		// Check that duration histogram exists and was updated
		// For histograms, we can't easily check the exact value with testutil.ToFloat64
		// but we can verify the metric was recorded by checking the registry
		gatherer := prometheus.Gatherers{monitor.registry}
		families, err := gatherer.Gather()
		require.NoError(t, err)

		// Look for the http_request_duration_seconds metric
		found := false
		for _, family := range families {
			if strings.Contains(*family.Name, "http_request_duration_seconds") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find HTTP duration metric")
	})

	t.Run("should handle unknown endpoints", func(t *testing.T) {
		config := &Config{
			Enabled:   true,
			Namespace: "test",
		}

		monitor, err := NewPrometheusMonitor(config)
		require.NoError(t, err)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(monitor.GinMiddleware())

		// Make a request to unregistered endpoint
		req, err := http.NewRequest("GET", "/unknown", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		// Should record with "unknown" endpoint
		requestsCount := testutil.ToFloat64(monitor.metrics.HTTPRequests.WithLabelValues("GET", "unknown", "404"))
		assert.Equal(t, float64(1), requestsCount)
	})

	t.Run("should do nothing when disabled", func(t *testing.T) {
		config := &Config{Enabled: false}
		monitor, err := NewPrometheusMonitor(config)
		require.NoError(t, err)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(monitor.GinMiddleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "test"})
		})

		req, err := http.NewRequest("GET", "/test", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		// No assertions on metrics since monitor is disabled
	})

	t.Run("should record request and response sizes", func(t *testing.T) {
		config := &Config{
			Enabled:   true,
			Namespace: "test",
		}

		monitor, err := NewPrometheusMonitor(config)
		require.NoError(t, err)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(monitor.GinMiddleware())
		router.POST("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "received"})
		})

		// Make a request with body
		body := strings.NewReader(`{"test": "data"}`)
		req, err := http.NewRequest("POST", "/test", body)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)

		// Check that request size histogram exists
		gatherer := prometheus.Gatherers{monitor.registry}
		families, err := gatherer.Gather()
		require.NoError(t, err)

		// Look for size metrics
		requestSizeFound := false
		responseSizeFound := false
		for _, family := range families {
			if strings.Contains(*family.Name, "http_request_size_bytes") {
				requestSizeFound = true
			}
			if strings.Contains(*family.Name, "http_response_size_bytes") {
				responseSizeFound = true
			}
		}
		assert.True(t, requestSizeFound, "Should find HTTP request size metric")
		assert.True(t, responseSizeFound, "Should find HTTP response size metric")
	})
}

func TestDatabaseMetrics(t *testing.T) {
	config := &Config{
		Enabled:   true,
		Namespace: "test",
	}

	monitor, err := NewPrometheusMonitor(config)
	require.NoError(t, err)

	t.Run("should record database operations", func(t *testing.T) {
		database := "postgres"
		operation := "SELECT"
		status := "success"
		duration := 50 * time.Millisecond

		monitor.RecordDBOperation(database, operation, status, duration)

		queriesCount := testutil.ToFloat64(monitor.metrics.DBQueries.WithLabelValues(database, operation, status))
		assert.Equal(t, float64(1), queriesCount)

		// Verify duration histogram was updated by checking registry
		gatherer := prometheus.Gatherers{monitor.registry}
		families, err := gatherer.Gather()
		require.NoError(t, err)

		found := false
		for _, family := range families {
			if strings.Contains(*family.Name, "database_query_duration_seconds") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find database duration metric")
	})

	t.Run("should record database connections", func(t *testing.T) {
		database := "postgres"
		state := "active"
		count := 10

		monitor.RecordDBConnections(database, state, count)

		connections := testutil.ToFloat64(monitor.metrics.DBConnections.WithLabelValues(database, state))
		assert.Equal(t, float64(count), connections)
	})

	t.Run("should do nothing when disabled", func(t *testing.T) {
		disabledConfig := &Config{Enabled: false}
		disabledMonitor, err := NewPrometheusMonitor(disabledConfig)
		require.NoError(t, err)

		// Should not panic
		disabledMonitor.RecordDBOperation("postgres", "SELECT", "success", time.Millisecond)
		disabledMonitor.RecordDBConnections("postgres", "active", 5)
	})
}

func TestMessageBrokerMetrics(t *testing.T) {
	config := &Config{
		Enabled:   true,
		Namespace: "test",
	}

	monitor, err := NewPrometheusMonitor(config)
	require.NoError(t, err)

	t.Run("should record message broker operations", func(t *testing.T) {
		driver := "redis"
		operation := "publish"
		topic := "user.events"
		status := "success"
		duration := 5 * time.Millisecond

		monitor.RecordMessageBrokerOperation(driver, operation, topic, status, duration)

		messagesCount := testutil.ToFloat64(monitor.metrics.MBMessages.WithLabelValues(driver, operation, topic, status))
		assert.Equal(t, float64(1), messagesCount)

		// Verify duration histogram was updated by checking registry
		gatherer := prometheus.Gatherers{monitor.registry}
		families, err := gatherer.Gather()
		require.NoError(t, err)

		found := false
		for _, family := range families {
			if strings.Contains(*family.Name, "message_broker_operation_duration_seconds") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find message broker duration metric")
	})

	t.Run("should record message broker connections", func(t *testing.T) {
		driver := "rabbitmq"
		state := "connected"
		count := 3

		monitor.RecordMessageBrokerConnections(driver, state, count)

		connections := testutil.ToFloat64(monitor.metrics.MBConnections.WithLabelValues(driver, state))
		assert.Equal(t, float64(count), connections)
	})

	t.Run("should do nothing when disabled", func(t *testing.T) {
		disabledConfig := &Config{Enabled: false}
		disabledMonitor, err := NewPrometheusMonitor(disabledConfig)
		require.NoError(t, err)

		// Should not panic
		disabledMonitor.RecordMessageBrokerOperation("redis", "publish", "topic", "success", time.Millisecond)
		disabledMonitor.RecordMessageBrokerConnections("redis", "connected", 1)
	})
}

func TestCacheMetrics(t *testing.T) {
	config := &Config{
		Enabled:   true,
		Namespace: "test",
	}

	monitor, err := NewPrometheusMonitor(config)
	require.NoError(t, err)

	t.Run("should record cache operations", func(t *testing.T) {
		operation := "get"
		status := "hit"
		duration := 1 * time.Millisecond

		monitor.RecordCacheOperation(operation, status, duration)

		operationsCount := testutil.ToFloat64(monitor.metrics.CacheOperations.WithLabelValues(operation, status))
		assert.Equal(t, float64(1), operationsCount)

		// Verify duration histogram was updated by checking registry
		gatherer := prometheus.Gatherers{monitor.registry}
		families, err := gatherer.Gather()
		require.NoError(t, err)

		found := false
		for _, family := range families {
			if strings.Contains(*family.Name, "cache_operation_duration_seconds") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find cache duration metric")
	})

	t.Run("should record cache hit rate", func(t *testing.T) {
		cacheType := "user_cache"
		hitRate := 85.5

		monitor.RecordCacheHitRate(cacheType, hitRate)

		rate := testutil.ToFloat64(monitor.metrics.CacheHitRate.WithLabelValues(cacheType))
		assert.Equal(t, hitRate, rate)
	})

	t.Run("should do nothing when disabled", func(t *testing.T) {
		disabledConfig := &Config{Enabled: false}
		disabledMonitor, err := NewPrometheusMonitor(disabledConfig)
		require.NoError(t, err)

		// Should not panic
		disabledMonitor.RecordCacheOperation("get", "hit", time.Millisecond)
		disabledMonitor.RecordCacheHitRate("cache", 90.0)
	})
}

func TestApplicationMetrics(t *testing.T) {
	config := &Config{
		Enabled:   true,
		Namespace: "test",
	}

	monitor, err := NewPrometheusMonitor(config)
	require.NoError(t, err)

	t.Run("should record business events", func(t *testing.T) {
		eventType := "user_registration"
		status := "success"

		monitor.RecordBusinessEvent(eventType, status)

		eventsCount := testutil.ToFloat64(monitor.metrics.BusinessMetrics.WithLabelValues(eventType, status))
		assert.Equal(t, float64(1), eventsCount)
	})

	t.Run("should record user sessions", func(t *testing.T) {
		sessionType := "web"
		count := 25

		monitor.RecordUserSessions(sessionType, count)

		sessions := testutil.ToFloat64(monitor.metrics.UserSessions.WithLabelValues(sessionType))
		assert.Equal(t, float64(count), sessions)
	})

	t.Run("should record active users", func(t *testing.T) {
		timeWindow := "1h"
		count := 100

		monitor.RecordActiveUsers(timeWindow, count)

		activeUsers := testutil.ToFloat64(monitor.metrics.ActiveUsers.WithLabelValues(timeWindow))
		assert.Equal(t, float64(count), activeUsers)
	})

	t.Run("should set app info", func(t *testing.T) {
		version := "1.0.0"
		environment := "production"
		service := "api"

		monitor.SetAppInfo(version, environment, service)

		appInfo := testutil.ToFloat64(monitor.metrics.AppInfo.WithLabelValues(version, environment, service))
		assert.Equal(t, float64(1), appInfo)
	})

	t.Run("should do nothing when disabled", func(t *testing.T) {
		disabledConfig := &Config{Enabled: false}
		disabledMonitor, err := NewPrometheusMonitor(disabledConfig)
		require.NoError(t, err)

		// Should not panic
		disabledMonitor.RecordBusinessEvent("event", "success")
		disabledMonitor.RecordUserSessions("web", 10)
		disabledMonitor.RecordActiveUsers("1h", 50)
		disabledMonitor.SetAppInfo("1.0.0", "test", "api")
	})
}

func TestHealthCheck(t *testing.T) {
	t.Run("should pass health check when enabled", func(t *testing.T) {
		config := &Config{
			Enabled:   true,
			Namespace: "test",
		}

		monitor, err := NewPrometheusMonitor(config)
		require.NoError(t, err)

		ctx := context.Background()
		err = monitor.HealthCheck(ctx)
		assert.NoError(t, err)
	})

	t.Run("should pass health check when disabled", func(t *testing.T) {
		config := &Config{Enabled: false}

		monitor, err := NewPrometheusMonitor(config)
		require.NoError(t, err)

		ctx := context.Background()
		err = monitor.HealthCheck(ctx)
		assert.NoError(t, err)
	})

	t.Run("should handle context cancellation", func(t *testing.T) {
		config := &Config{
			Enabled:   true,
			Namespace: "test",
		}

		monitor, err := NewPrometheusMonitor(config)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Health check should still work since it doesn't actually use the context
		err = monitor.HealthCheck(ctx)
		assert.NoError(t, err)
	})
}

func TestClose(t *testing.T) {
	t.Run("should close successfully", func(t *testing.T) {
		config := &Config{
			Enabled:   true,
			Namespace: "test",
		}

		monitor, err := NewPrometheusMonitor(config)
		require.NoError(t, err)

		err = monitor.Close()
		assert.NoError(t, err)
	})

	t.Run("should close successfully when disabled", func(t *testing.T) {
		config := &Config{Enabled: false}

		monitor, err := NewPrometheusMonitor(config)
		require.NoError(t, err)

		err = monitor.Close()
		assert.NoError(t, err)
	})
}

func TestMetricsServerSetup(t *testing.T) {
	t.Run("should create metrics server when enabled", func(t *testing.T) {
		config := &Config{
			Enabled:     true,
			Namespace:   "test",
			MetricsPath: "/metrics",
			ListenAddr:  ":0", // Use random port
		}

		monitor, err := NewPrometheusMonitor(config)
		require.NoError(t, err)

		// We can't easily test StartMetricsServer without it blocking,
		// but we can test that the components are set up correctly
		handler := monitor.GetHandler()
		assert.NotNil(t, handler)

		// Test metrics endpoint
		req, err := http.NewRequest("GET", "/metrics", nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Contains(t, recorder.Header().Get("Content-Type"), "text/plain")
		assert.Contains(t, recorder.Body.String(), "# HELP")
	})

	t.Run("should handle disabled state", func(t *testing.T) {
		config := &Config{
			Enabled:     false,
			MetricsPath: "/metrics",
			ListenAddr:  ":9091",
		}

		monitor, err := NewPrometheusMonitor(config)
		require.NoError(t, err)

		// StartMetricsServer should return nil without starting server
		err = monitor.StartMetricsServer()
		assert.NoError(t, err)
	})
}

func TestMetricsIntegration(t *testing.T) {
	t.Run("should record multiple metrics correctly", func(t *testing.T) {
		config := &Config{
			Enabled:   true,
			Namespace: "integration_test",
		}

		monitor, err := NewPrometheusMonitor(config)
		require.NoError(t, err)

		// Record various metrics
		monitor.RecordDBOperation("postgres", "SELECT", "success", 10*time.Millisecond)
		monitor.RecordDBOperation("postgres", "INSERT", "success", 20*time.Millisecond)
		monitor.RecordDBOperation("postgres", "SELECT", "error", 5*time.Millisecond)

		monitor.RecordCacheOperation("get", "hit", time.Millisecond)
		monitor.RecordCacheOperation("get", "miss", 2*time.Millisecond)
		monitor.RecordCacheOperation("set", "success", 3*time.Millisecond)

		monitor.RecordBusinessEvent("user_login", "success")
		monitor.RecordBusinessEvent("user_login", "success")
		monitor.RecordBusinessEvent("user_login", "failed")

		// Verify metrics
		selectSuccessCount := testutil.ToFloat64(monitor.metrics.DBQueries.WithLabelValues("postgres", "SELECT", "success"))
		assert.Equal(t, float64(1), selectSuccessCount)

		insertSuccessCount := testutil.ToFloat64(monitor.metrics.DBQueries.WithLabelValues("postgres", "INSERT", "success"))
		assert.Equal(t, float64(1), insertSuccessCount)

		selectErrorCount := testutil.ToFloat64(monitor.metrics.DBQueries.WithLabelValues("postgres", "SELECT", "error"))
		assert.Equal(t, float64(1), selectErrorCount)

		cacheHitCount := testutil.ToFloat64(monitor.metrics.CacheOperations.WithLabelValues("get", "hit"))
		assert.Equal(t, float64(1), cacheHitCount)

		cacheMissCount := testutil.ToFloat64(monitor.metrics.CacheOperations.WithLabelValues("get", "miss"))
		assert.Equal(t, float64(1), cacheMissCount)

		cacheSetCount := testutil.ToFloat64(monitor.metrics.CacheOperations.WithLabelValues("set", "success"))
		assert.Equal(t, float64(1), cacheSetCount)

		loginSuccessCount := testutil.ToFloat64(monitor.metrics.BusinessMetrics.WithLabelValues("user_login", "success"))
		assert.Equal(t, float64(2), loginSuccessCount)

		loginFailedCount := testutil.ToFloat64(monitor.metrics.BusinessMetrics.WithLabelValues("user_login", "failed"))
		assert.Equal(t, float64(1), loginFailedCount)
	})

	t.Run("should work with real Gin server", func(t *testing.T) {
		config := &Config{
			Enabled:   true,
			Namespace: "gin_test",
		}

		monitor, err := NewPrometheusMonitor(config)
		require.NoError(t, err)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(monitor.GinMiddleware())

		router.GET("/users/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(200, gin.H{"user_id": id})
		})

		router.POST("/users", func(c *gin.Context) {
			c.JSON(201, gin.H{"message": "created"})
		})

		// Make multiple requests
		requests := []struct {
			method string
			path   string
			status int
		}{
			{"GET", "/users/123", 200},
			{"GET", "/users/456", 200},
			{"POST", "/users", 201},
			{"GET", "/nonexistent", 404},
		}

		for _, req := range requests {
			httpReq, err := http.NewRequest(req.method, req.path, nil)
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httpReq)

			assert.Equal(t, req.status, recorder.Code)
		}

		// Verify metrics were recorded
		getUsersCount := testutil.ToFloat64(monitor.metrics.HTTPRequests.WithLabelValues("GET", "/users/:id", "200"))
		assert.Equal(t, float64(2), getUsersCount)

		postUsersCount := testutil.ToFloat64(monitor.metrics.HTTPRequests.WithLabelValues("POST", "/users", "201"))
		assert.Equal(t, float64(1), postUsersCount)

		notFoundCount := testutil.ToFloat64(monitor.metrics.HTTPRequests.WithLabelValues("GET", "unknown", "404"))
		assert.Equal(t, float64(1), notFoundCount)
	})
}