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
```

## Claude Code skills

This repo ships two project skills (in `.claude/skills/`) for common maintenance work. In Claude Code, invoke them by name or with a slash command.

### `verify-metrics`
Boots the example server, fires representative requests, scrapes `/metrics`, and reports the emitted `request_duration_seconds` series (labels + bucket boundaries). Since the library has no unit tests, this is the primary way to confirm a change is correct.

Use it after any change to `prometheus.go` — bucket edits, dependency upgrades, or labeling logic. Ask Claude:

    /verify-metrics

It confirms, among other things, that `/values/123` collapses to the label `path="GET_/values/{id}"` (cardinality control) rather than exploding per-id.

### `tune-buckets`
Safely changes the histogram bucket boundaries for `request_duration_seconds` (the `Buckets` slice in `prometheus.go`). It restates the resulting slice, keeps values strictly ascending, verifies the output via `verify-metrics`, and flags the downstream blast radius (dashboards / alerts / SLOs) since these boundaries are consumed by everyone who upgrades. Ask Claude:

    /tune-buckets add a 0.05 bucket and remove 5
