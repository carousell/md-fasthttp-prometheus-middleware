---
name: verify-metrics
description: Boot the example fasthttp server, fire representative requests, scrape /metrics, and report the emitted request_duration_seconds series (labels + bucket boundaries). Use to prove a change to prometheus.go — bucket edits, dependency upgrades, or labeling logic — produces the metric output you expect. This repo has no tests, so this is the primary verification harness.
---

# verify-metrics

The library in `prometheus.go` has no unit tests. The only way to confirm a change is correct is to run it and inspect real metric output. This skill does that end-to-end.

## When to use

- After editing the `Buckets` slice in `registerMetrics` (see the `tune-buckets` skill).
- After a dependency bump (`fasthttp`, `fasthttp/router`, `prometheus/client_golang`) — the labeling logic in `HandlerFunc` calls `router.List()` / `router.Lookup()`, whose behavior can drift silently.
- After changing any labeling logic (the `code` / `path` labels, the `404_<METHOD>` special case).

## Steps

1. **Build first.** Bail early on compile errors:
   ```bash
   go build ./... && go vet ./...
   ```

2. **Start the example server in the background** (it listens on `:8080`, routes `/health` and `/values/{id}`, metrics at `/metrics`):
   ```bash
   go run ./example & pid=$!
   ```
   Use a background run and capture the PID in `pid` so you can stop it in step 5. Poll `http://localhost:8080/health` until it responds before firing traffic.

3. **Fire representative traffic** — one request per label case that matters in this codebase:
   ```bash
   curl -s http://localhost:8080/health        # 200, static route  -> path="GET_/health"
   curl -s http://localhost:8080/values/123    # 200, param route   -> path MUST be "GET_/values/{id}", not ".../123"
   curl -s http://localhost:8080/nope          # 404               -> path="404_GET" (method comes SECOND for 404s)
   curl -s -X POST http://localhost:8080/health # 405/404 non-GET  -> confirms method handling
   ```

4. **Scrape and inspect** the metric:
   ```bash
   curl -s http://localhost:8080/metrics | grep request_duration_seconds
   ```
   Report back:
   - The **`path` labels** — confirm `/values/123` collapsed to `GET_/values/{id}` (cardinality control working) and the 404 is `404_GET`.
   - The **`le` bucket boundaries** — list them and confirm they match the `Buckets` slice in `prometheus.go`.
   - The **`code` labels** map to the statuses you sent.

5. **Stop the server** (kill the background PID). Don't leave `:8080` bound.

## What "pass" looks like

- `/values/123` produces exactly one series with `path="GET_/values/{id}"` — never a per-id label. A per-id label is a cardinality regression and a failure.
- The `le` values in the output equal the code's `Buckets` slice plus `+Inf`.
- No panic in the server log (historically, invalid labels have panicked — see commit `cde5490`).

Report the actual scraped output, not just a pass/fail verdict.
