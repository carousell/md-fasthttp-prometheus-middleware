# fasthttp prometheus-middleware
Prometheus middleware for [fasthttp](https://github.com/valyala/fasthttp) 

Exports metrics for request duration ```request_duration_seconds``` 
with http status code as ```code``` and http request method + endpoint/route as ```path``` 
f.e ```code="200",path="GET_/health"```, ```code="201",path="POST_/foo"``` 

## Example 
using fasthttp/router

    package main

    import (
	"log"

	fasthttpprom "github.com/carousell/fasthttp-prometheus-middleware"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	)

    func main() {

		r := router.New()
		p := fasthttpprom.NewPrometheus("")
		p.Use(r)

		r.GET("/health", func(ctx *fasthttp.RequestCtx) {
			ctx.SetStatusCode(200)
			ctx.SetBody([]byte(`{"status": "pass"}`))
			log.Println(string(ctx.Request.URI().Path()))
		})

		log.Println("main is listening on ", "8080")
		log.Fatal(fasthttp.ListenAndServe(":"+"8080", p.Handler))
	
    }

Example metrics for above code in /metrics endpoint

```request_duration_seconds_bucket{code="200",path="GET_/health",le="0.005"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="0.01"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="0.02"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="0.04"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="0.06"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="0.08"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="0.1"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="0.15"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="0.25"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="0.4"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="0.6"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="0.8"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="1"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="1.5"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="2"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="3"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="5"} 25063
request_duration_seconds_bucket{code="200",path="GET_/health",le="+Inf"} 25063
request_duration_seconds_sum{code="200",path="GET_/health"} 0.14781658099999923
request_duration_seconds_count{code="200",path="GET_/health"} 25063

## Custom Metrics

You can register your own custom Prometheus metrics (counters, gauges, histograms, summaries) to be exported alongside the default request duration metrics. All registration methods are **thread-safe** and protected by mutex.

### Basic Usage

```go
package main

import (
    "log"
    
    fasthttpprom "github.com/carousell/fasthttp-prometheus-middleware"
    "github.com/fasthttp/router"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/valyala/fasthttp"
)

func main() {
    r := router.New()
    p := fasthttpprom.NewPrometheus("myapp")
    
    // Create a custom counter
    requestCounter := prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "myapp_custom_requests_total",
            Help: "Total number of custom requests",
        },
        []string{"endpoint", "method"},
    )
    
    // Register it with the Prometheus middleware
    if err := p.RegisterMetric(requestCounter); err != nil {
        log.Fatalf("Failed to register metric: %v", err)
    }
    
    p.Use(r)
    
    // Use the metric in your handlers
    r.GET("/api/users", func(ctx *fasthttp.RequestCtx) {
        requestCounter.WithLabelValues("/api/users", "GET").Inc()
        ctx.SetStatusCode(200)
        ctx.SetBody([]byte(`{"users": []}`))
    })
    
    log.Fatal(fasthttp.ListenAndServe(":8080", p.Handler))
}
```

### Available Methods

- **`RegisterMetric(collector prometheus.Collector) error`** - Registers a custom metric. Returns an error if registration fails. Thread-safe.
- **`MustRegisterMetric(collector prometheus.Collector)`** - Like RegisterMetric but panics on error. Useful during initialization. Thread-safe.
- **`GetCustomMetrics() []prometheus.Collector`** - Returns a copy of all registered custom metrics. Thread-safe.

### Advanced Pattern: Metrics Module with Handler Functions

For larger applications, organize your metrics in a separate module and pass them to handler functions:

```go
// metrics/metrics.go
package metrics

import (
    "fmt"
    fasthttpprom "github.com/carousell/fasthttp-prometheus-middleware"
    "github.com/prometheus/client_golang/prometheus"
)

type AppMetrics struct {
    LoginAttempts  *prometheus.CounterVec
    CacheHits      prometheus.Counter
    ProcessingTime *prometheus.HistogramVec
}

func New() *AppMetrics {
    return &AppMetrics{
        LoginAttempts: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "app_login_attempts_total",
                Help: "Total login attempts",
            },
            []string{"status", "method"},
        ),
        CacheHits: prometheus.NewCounter(
            prometheus.CounterOpts{
                Name: "app_cache_hits_total",
                Help: "Total cache hits",
            },
        ),
        ProcessingTime: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name: "app_processing_seconds",
                Help: "Processing time",
            },
            []string{"operation"},
        ),
    }
}

func (m *AppMetrics) Register(p *fasthttpprom.Prometheus) error {
    if err := p.RegisterMetric(m.LoginAttempts); err != nil {
        return fmt.Errorf("failed to register LoginAttempts: %w", err)
    }
    if err := p.RegisterMetric(m.CacheHits); err != nil {
        return fmt.Errorf("failed to register CacheHits: %w", err)
    }
    if err := p.RegisterMetric(m.ProcessingTime); err != nil {
        return fmt.Errorf("failed to register ProcessingTime: %w", err)
    }
    return nil
}

// handlers/auth.go
package handlers

import (
    "myapp/metrics"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/valyala/fasthttp"
)

func HandleLogin(metrics *metrics.AppMetrics) fasthttp.RequestHandler {
    return func(ctx *fasthttp.RequestCtx) {
        username := string(ctx.FormValue("username"))
        if username == "" {
            metrics.LoginAttempts.WithLabelValues("failed", "password").Inc()
            ctx.SetStatusCode(400)
            ctx.SetBody([]byte(`{"error": "username required"}`))
            return
        }
        
        metrics.LoginAttempts.WithLabelValues("success", "password").Inc()
        ctx.SetStatusCode(200)
        ctx.SetBody([]byte(`{"token": "abc123"}`))
    }
}

func HandleData(metrics *metrics.AppMetrics) fasthttp.RequestHandler {
    return func(ctx *fasthttp.RequestCtx) {
        // Simulate cache lookup
        if data := checkCache(ctx); data != "" {
            metrics.CacheHits.Inc()
            ctx.SetStatusCode(200)
            ctx.SetBody([]byte(data))
            return
        }
        
        // Cache miss
        ctx.SetStatusCode(200)
        ctx.SetBody([]byte(`{"data": "from database"}`))
    }
}

func HandleProcess(metrics *metrics.AppMetrics) fasthttp.RequestHandler {
    return func(ctx *fasthttp.RequestCtx) {
        // Track processing time
        timer := prometheus.NewTimer(metrics.ProcessingTime.WithLabelValues("data_transformation"))
        defer timer.ObserveDuration()
        
        // Process request
        ctx.SetStatusCode(200)
        ctx.SetBody([]byte(`{"processed": true}`))
    }
}

// main.go
package main

import (
    "log"
    "myapp/metrics"
    "myapp/handlers"
    
    fasthttpprom "github.com/carousell/fasthttp-prometheus-middleware"
    "github.com/fasthttp/router"
    "github.com/valyala/fasthttp"
)

func main() {
    r := router.New()
    p := fasthttpprom.NewPrometheus("myapp")
    
    // Create and register all app metrics
    appMetrics := metrics.New()
    if err := appMetrics.Register(p); err != nil {
        log.Fatal(err)
    }
    
    p.Use(r)
    
    // Pass metrics to handler functions
    r.POST("/auth/login", handlers.HandleLogin(appMetrics))
    r.GET("/api/data", handlers.HandleData(appMetrics))
    r.POST("/api/process", handlers.HandleProcess(appMetrics))
    
    log.Println("Server listening on :8080")
    log.Fatal(fasthttp.ListenAndServe(":8080", p.Handler))
}
```

This pattern provides several benefits:
- **Separation of concerns**: Metrics are defined separately from business logic
- **Reusability**: Metrics can be shared across multiple handlers
- **Testability**: Handlers can be tested with mock metrics
- **Type safety**: Metrics struct provides compile-time safety
- **Thread safety**: All metric registration methods are protected by mutex

### Supported Metric Types

All Prometheus metric types are supported:
- **Counter** - Monotonically increasing value (e.g., request counts, login attempts)
- **Gauge** - Value that can go up and down (e.g., memory usage, active connections)
- **Histogram** - Samples observations and counts them in buckets (e.g., request sizes, response times)
- **Summary** - Similar to histogram but calculates quantiles

See the [example/custom_metrics_example.go](example/custom_metrics_example.go) file for complete runnable examples.

## Installation

```bash
go get github.com/carousell/fasthttp-prometheus-middleware@poc-cus-metrics-2025
```

For local development without a version tag, add this to your consuming repo's `go.mod`:

```go
replace github.com/carousell/fasthttp-prometheus-middleware => /path/to/local/md-fasthttp-prometheus-middleware
```

