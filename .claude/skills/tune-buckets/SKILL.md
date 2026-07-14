---
name: tune-buckets
description: Change the Prometheus histogram bucket boundaries for request_duration_seconds safely. Use when asked to add, remove, or retune latency buckets in prometheus.go. Guides the edit, explains the downstream blast radius (dashboards/alerts/SLOs), and verifies the result end-to-end.
---

# tune-buckets

The histogram buckets for `request_duration_seconds` are defined in one place — the `Buckets` slice inside `registerMetrics` in `prometheus.go`:

```go
Buckets: []float64{0.1, 0.2, 0.3, 0.5, 0.75, 1, 1.5, 2, 3, 5},
```

These boundaries have been retuned across several PRs (see git history: "Update prom buckets", "remove 10, 15, 20 from bucket", "remove 7 from bucket"). Treat every change as significant — not cosmetic.

## Why this is delicate

Buckets are **cumulative** (`le` = "less than or equal to") and Prometheus auto-appends `+Inf`. Consumers of this library build latency dashboards, `histogram_quantile()` queries, and alert thresholds against these exact boundaries. Changing them:

- **Breaks historical continuity** — a removed bucket leaves gaps in old time series; quantile estimates shift.
- **Can silently degrade alerts** — an SLO alert keyed on `le="7"` goes dead if you remove the `7` bucket.
- **Affects every consumer** on upgrade, since this is a published module.

So the bar for a change is: it must be intentional, the values must stay **strictly ascending**, and the result must be verified.

## Steps

1. **Confirm the requested boundaries.** Get the exact target slice from the user. If they only say "add X" / "remove Y", restate the full resulting slice back to them before editing.

2. **Edit only the `Buckets` slice** in `registerMetrics` (`prometheus.go`). Keep values **strictly ascending**; do not add `+Inf` (Prometheus adds it). Do not touch the metric name, `Help`, or label set unless asked.

3. **Verify end-to-end** with the `verify-metrics` skill. Confirm the scraped `le=` boundaries in `/metrics` exactly match the new slice (plus `+Inf`).

4. **Flag the blast radius in your summary.** Explicitly list which boundaries were added/removed so the reviewer knows what downstream dashboards/alerts to check. If a boundary was *removed*, call it out as a potentially breaking change for consumers.

## Release note

A bucket change is meaningful to consumers. When it ships, it warrants a new version tag (see the release workflow) and a one-line changelog entry describing the added/removed boundaries.
