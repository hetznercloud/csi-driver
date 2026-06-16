# Monitoring

Monitoring is an important part of managing any system, and the csi-driver components are no exception. To help you keep an eye on how the components are performing, we've exposed Prometheus-compatible metrics on port 9189. You can configure the endpoint for these metrics by setting the `METRICS_ENDPOINT` environment variable to the appropriate value for your system.

The metrics expose details about the following components:

- Go Runtime
- gRPC Server for CSI calls
- HTTP calls made to Hetzner Cloud API

## Scraping

There are multiple ways to scrape the metrics on Kubernetes:

### prometheus-operator `ServiceMonitor`

If you're using the prometheus-operator, you can use our `ServiceMonitors` to set up scraping.
You can configure these ServiceMonitors in the Helm chart:

```yaml
metrics:
  enabled: true
  serviceMonitor:
    enabled: true
```

Prometheus only scrapes `ServiceMonitors` whose labels match the `serviceMonitorSelector` of your `Prometheus` resource. If your Prometheus selects on a `release` label (the default for the kube-prometheus-stack), set it via `metrics.serviceMonitor.labels` so the target is actually scraped:

```yaml
metrics:
  enabled: true
  serviceMonitor:
    enabled: true
    labels:
      release: YOUR_RELEASE
```

> [!TIP]
> To learn more about how the Prometheus operator works:
>
> - https://prometheus-operator.dev/docs/getting-started/introduction/
> - https://prometheus-operator.dev/docs/getting-started/design/#servicemonitor

### Using `kubernetes_sd_configs`

If you're running Prometheus without the operator, you can configure scraping with `kubernetes_sd_configs` by adding annotations to the csi-driver workloads. Add these annotations to the `hcloud-csi-controller` Deployment and the `hcloud-csi-node` DaemonSet:

```yaml
annotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "9189"
```

With these annotations in place, Prometheus can discover and scrape metrics from the csi-driver components.

> [!TIP]
> To learn more about `kubernetes_sd_configs`:
>
> - https://prometheus.io/docs/prometheus/latest/getting_started/
> - https://prometheus.io/docs/prometheus/latest/configuration/configuration/#kubernetes_sd_config

## Grafana Dashboard

In addition to scraping metrics, you'll also want a way to visualize those metrics.
To help with this, we provide a default Grafana dashboard that can be used to display the most important metrics.
This dashboard has been confirmed to work with kube-prometheus-stack, but it may require some tweaking to work correctly in your specific environment.
You can find the dashboard at [`deploy/monitoring/grafana-dashboard.json`](../../../deploy/monitoring/grafana-dashboard.json).

> [!TIP]
> To learn more about Grafana dashboards:
>
> - https://grafana.com/docs/grafana/latest/visualizations/dashboards
> - https://grafana.com/docs/grafana/latest/visualizations/dashboards/build-dashboards/import-dashboards/
