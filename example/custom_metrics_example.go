package main

import (
	"fmt"
	"log"

	fasthttpprom "github.com/carousell/fasthttp-prometheus-middleware"
	"github.com/fasthttp/router"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/valyala/fasthttp"
)

// Advanced example: Creating metrics in a separate package/module
// This demonstrates how to organize custom metrics in a struct and pass them to handlers
type CustomMetrics struct {
	LoginAttempts  *prometheus.CounterVec
	CacheHitRate   prometheus.Counter
	ProcessingTime *prometheus.HistogramVec
}

// NewCustomMetrics creates and initializes a CustomMetrics instance with all metrics
// This should be called once during application startup
func NewCustomMetrics() *CustomMetrics {
	return &CustomMetrics{
		LoginAttempts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auth_login_attempts_total",
				Help: "Total number of login attempts",
			},
			[]string{"status", "method"},
		),
		CacheHitRate: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "cache_hits_total",
				Help: "Total number of cache hits",
			},
		),
		ProcessingTime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "business_logic_processing_seconds",
				Help:    "Time spent in business logic processing",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation"},
		),
	}
}

// Register registers all custom metrics with the Prometheus middleware
// This method is safe for concurrent use due to mutex protection in RegisterMetric
func (cm *CustomMetrics) Register(p *fasthttpprom.Prometheus) error {
	if err := p.RegisterMetric(cm.LoginAttempts); err != nil {
		return fmt.Errorf("failed to register LoginAttempts: %w", err)
	}
	if err := p.RegisterMetric(cm.CacheHitRate); err != nil {
		return fmt.Errorf("failed to register CacheHitRate: %w", err)
	}
	if err := p.RegisterMetric(cm.ProcessingTime); err != nil {
		return fmt.Errorf("failed to register ProcessingTime: %w", err)
	}
	return nil
}

// advancedExample demonstrates the recommended pattern for larger applications:
// 1. Create a metrics struct to organize related metrics
// 2. Register all metrics during initialization
// 3. Pass metrics to handler factory functions
// 4. Use metrics within handlers
//
// To run this example:
// - Rename this function to 'main' or call it from main.go
// - Run: go run example/*.go
// - Test endpoints: curl -X POST http://localhost:8080/auth/login?username=test
// - View metrics: curl http://localhost:8080/metrics
func advancedExample() {
	r := router.New()
	p := fasthttpprom.NewPrometheus("myapp")

	// Create and register custom metrics module
	customMetrics := NewCustomMetrics()
	if err := customMetrics.Register(p); err != nil {
		log.Fatalf("Failed to register custom metrics: %v", err)
	}

	p.Use(r)

	// Pass customMetrics to handler functions
	r.POST("/auth/login", handleLogin(customMetrics))
	r.GET("/api/data", handleData(customMetrics))
	r.POST("/api/process", handleProcess(customMetrics))

	log.Println("Server listening on :8080")
	log.Fatal(fasthttp.ListenAndServe(":8080", p.Handler))
}

// handleLogin is a handler factory that accepts metrics and returns a configured handler
// This pattern allows metrics to be injected and used within the handler closure
func handleLogin(metrics *CustomMetrics) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		// Simulate login logic
		username := string(ctx.FormValue("username"))
		if username == "" {
			metrics.LoginAttempts.WithLabelValues("failed", "password").Inc()
			ctx.SetStatusCode(400)
			ctx.SetBody([]byte(`{"error": "username required"}`))
			return
		}

		// Successful login
		metrics.LoginAttempts.WithLabelValues("success", "password").Inc()
		ctx.SetStatusCode(200)
		ctx.SetBody([]byte(`{"token": "abc123"}`))
	}
}

// handleData demonstrates using a counter metric to track cache hits
func handleData(metrics *CustomMetrics) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		dataID := string(ctx.QueryArgs().Peek("id"))

		// Simulate cache lookup
		cachedData := checkCache(dataID)
		if cachedData != "" {
			metrics.CacheHitRate.Inc()
			ctx.SetStatusCode(200)
			ctx.SetBody([]byte(cachedData))
			return
		}

		// Cache miss - fetch from database
		ctx.SetStatusCode(200)
		ctx.SetBody([]byte(`{"data": "from database"}`))
	}
}

// handleProcess demonstrates using a histogram metric to track processing duration
func handleProcess(metrics *CustomMetrics) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		// Track processing time for this operation
		timer := prometheus.NewTimer(metrics.ProcessingTime.WithLabelValues("data_transformation"))
		defer timer.ObserveDuration()

		// Simulate some processing work
		result := performBusinessLogic(string(ctx.PostBody()))

		ctx.SetStatusCode(200)
		ctx.SetBody([]byte(result))
	}
}

// Helper functions for the examples
func checkCache(id string) string {
	// Simulate cache lookup
	if id == "cached-123" {
		return `{"data": "from cache", "id": "cached-123"}`
	}
	return ""
}

func performBusinessLogic(input string) string {
	// Simulate business logic processing
	return fmt.Sprintf(`{"processed": true, "input_length": %d}`, len(input))
}

// customMetricsExample demonstrates basic usage with inline metrics creation
// This is suitable for simple applications with few custom metrics
//
// To run this example:
// - Rename this function to 'main' or call it from main.go
// - Run: go run example/*.go
// - Test endpoints: curl http://localhost:8080/api/users
// - View metrics: curl http://localhost:8080/metrics
func customMetricsExample() {
	r := router.New()
	p := fasthttpprom.NewPrometheus("myapp")

	// Create custom counter metric
	requestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "myapp_custom_requests_total",
			Help: "Total number of custom requests",
		},
		[]string{"endpoint", "method"},
	)

	// Register the custom counter with the Prometheus instance
	if err := p.RegisterMetric(requestCounter); err != nil {
		log.Fatalf("Failed to register custom metric: %v", err)
	}

	// Create another custom metric - a gauge
	activeConnections := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "myapp_active_connections",
			Help: "Number of active connections",
		},
	)

	// Use MustRegisterMetric if you want to panic on error
	p.MustRegisterMetric(activeConnections)

	// Create a histogram for response sizes
	responseSizeHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "myapp_response_size_bytes",
			Help:    "Response size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"endpoint"},
	)

	p.MustRegisterMetric(responseSizeHistogram)

	p.Use(r)

	// Use the custom metrics in your handlers
	r.GET("/api/users", func(ctx *fasthttp.RequestCtx) {
		// Increment the custom counter
		requestCounter.WithLabelValues("/api/users", "GET").Inc()

		// Simulate some work
		activeConnections.Inc()
		defer activeConnections.Dec()

		response := []byte(`{"users": ["alice", "bob"]}`)

		// Record response size
		responseSizeHistogram.WithLabelValues("/api/users").Observe(float64(len(response)))

		ctx.SetStatusCode(200)
		ctx.SetBody(response)
	})

	r.POST("/api/orders", func(ctx *fasthttp.RequestCtx) {
		// Track POST requests separately
		requestCounter.WithLabelValues("/api/orders", "POST").Inc()

		activeConnections.Inc()
		defer activeConnections.Dec()

		response := []byte(`{"order_id": "12345"}`)
		responseSizeHistogram.WithLabelValues("/api/orders").Observe(float64(len(response)))

		ctx.SetStatusCode(201)
		ctx.SetBody(response)
	})

	r.GET("/health", func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(200)
		ctx.SetBody([]byte(`{"status": "pass"}`))
	})

	log.Println("Server with custom metrics listening on :8080")
	log.Println("Metrics available at http://localhost:8080/metrics")
	log.Fatal(fasthttp.ListenAndServe(":8080", p.Handler))
}
