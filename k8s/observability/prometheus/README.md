# What is Prometheus?

Prometheus is an open-source monitoring and alerting toolkit to collect and store **metrics**.

## What is kube-prometheus-stack?

kube-prometheus-stack is a collection of Kubernetes manifests, Grafana dashboards, and Prometheus rules combined into a single package to provide a complete monitoring solution for Kubernetes clusters.

It contains the following components:

- Prometheus: for collecting and storing metrics
- Grafana: for visualizing metrics
- Kube-state-metrics: for collecting Kubernetes object metrics

## Why use kube-prometheus-stack?

kube-prometheus-stack automatically deploys and configures all the components needed for a complete monitoring solution for Kubernetes clusters.
Node Exporter is disabled. Host metrics come from `alloy-node` (`prometheus.exporter.unix`), scraped with `job="node-exporter"` so the bundled rules and dashboards keep working.

## Why is Grafana disabled?

We use kube-prometheus-stack because it provides a complete monitoring solution for Kubernetes clusters.
However, in order to separate concerns and manage resources more effectively, we can deploy Grafana as a standalone application.
Grafana is not only for prometheus but also for other data sources, including Loki for logs and Tempo for traces.

## Push receivers

| Receiver | Path | Sender |
| --- | --- | --- |
| OTLP | `/api/v1/otlp` | `alloy-gateway` (app OTLP metrics) |
| Remote write | `/api/v1/write` | `alloy-gateway` (blackbox probes) |
