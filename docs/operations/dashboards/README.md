# 📈 Dashboard Artifacts

> Last updated: 2026-02-19

This directory contains starter Grafana dashboard JSON artifacts for local bootstrap.

## Artifacts

- `secrets-overview.json`: baseline request/error/latency view
- `secrets-rate-limiting.json`: `429` behavior and throttle pressure view

## Import

1. Open Grafana
2. Go to Dashboards -> Import
3. Upload one of the JSON files from this directory
4. Select your Prometheus datasource

## Notes

- Treat these dashboards as starter templates
- Adjust panel thresholds and time windows for your traffic profile
