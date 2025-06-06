package fasthttpprom

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/fasthttp/router"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

var defaultMetricPath = "/metrics"

// ListenerHandler url label
type ListenerHandler func(c *fasthttp.RequestCtx) string

// Prometheus contains the metrics gathered by the instance and its path
type Prometheus struct {
	reqDur        *prometheus.HistogramVec
	router        *router.Router
	listenAddress string
	MetricsPath   string
	Handler       fasthttp.RequestHandler
}

// NewPrometheus generates a new set of metrics with a certain subsystem name
func NewPrometheus(subsystem string) *Prometheus {
	p := &Prometheus{
		MetricsPath: defaultMetricPath,
	}
	p.registerMetrics(subsystem)

	return p
}

// SetListenAddress for exposing metrics on address. If not set, it will be exposed at the
// same address of api that is being used
func (p *Prometheus) SetListenAddress(address string) {
	p.listenAddress = address
	if p.listenAddress != "" {
		p.router = router.New()
	}
}

// SetListenAddressWithRouter for using a separate router to expose metrics. (this keeps things like GET /metrics out of
// your content's access log).
func (p *Prometheus) SetListenAddressWithRouter(listenAddress string, r *router.Router) {
	p.listenAddress = listenAddress
	if len(p.listenAddress) > 0 {
		p.router = r
	}
}

// SetMetricsPath set metrics paths for Custom path
func (p *Prometheus) SetMetricsPath(r *router.Router) {
	if p.listenAddress != "" {
		r.GET(p.MetricsPath, prometheusHandler())
		p.runServer()
	} else {
		r.GET(p.MetricsPath, prometheusHandler())
	}
}

func (p *Prometheus) runServer() {
	if p.listenAddress != "" {
		go fasthttp.ListenAndServe(p.listenAddress, p.router.Handler)
	}
}

func (p *Prometheus) registerMetrics(subsystem string) {
	p.reqDur = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: subsystem,
			Name:      "request_duration_seconds",
			Help:      "request latencies",
			Buckets:   []float64{0.1, 0.2, 0.3, 0.5, 0.75, 1, 1.5, 2, 3, 5},
		},
		[]string{"code", "path"},
	)

	prometheus.Register(p.reqDur)
}

// Custom adds the middleware to a fasthttp
func (p *Prometheus) Custom(r *router.Router) {
	p.router = r
	p.SetMetricsPath(r)
	p.Handler = p.HandlerFunc()
}

// Use adds the middleware to a fasthttp
func (p *Prometheus) Use(r *router.Router) {
	p.router = r
	r.GET(p.MetricsPath, prometheusHandler())
	p.Handler = p.HandlerFunc()
}

// HandlerFunc is onion or wraper to handler for fasthttp listenandserve
func (p *Prometheus) HandlerFunc() fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		uri := string(ctx.Request.URI().Path())
		if uri == p.MetricsPath {
			// next
			p.router.Handler(ctx)
			return
		}
		start := time.Now()
		// next
		p.router.Handler(ctx)

		status := strconv.Itoa(ctx.Response.StatusCode())
		elapsed := float64(time.Since(start)) / float64(time.Second)
		// get route pattern of url
		routeList := p.router.List()
		paths, ok := routeList[string(ctx.Request.Header.Method())]
		handler, _ := p.router.Lookup(string(ctx.Request.Header.Method()), uri, ctx)
		if ok {
			for _, v := range paths {
				tmp, _ := p.router.Lookup(string(ctx.Request.Header.Method()), v, ctx)
				if fmt.Sprintf("%v", tmp) == fmt.Sprintf("%v", handler) {
					uri = v
					break
				}
			}
		}
		ep := ""
		if status == "404" {
			ep = "404_" + string(ctx.Method())
		} else {
			ep = string(ctx.Method()) + "_" + uri
		}
		ob, err := p.reqDur.GetMetricWithLabelValues(status, ep)
		if err != nil {
			log.Printf("Fail to GetMetricWithLabelValues: %s\n", err)
			return
		}
		ob.Observe(elapsed)
	}
}

// since prometheus/client_golang use net/http we need this net/http adapter for fasthttp
func prometheusHandler() fasthttp.RequestHandler {
	return fasthttpadaptor.NewFastHTTPHandler(promhttp.Handler())
}
