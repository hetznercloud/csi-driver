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

> [!TIP]
> To learn more about how the Prometheus operator works:
>
> - https://prometheus-operator.dev/docs/getting-started/introduction/
> - https://prometheus-operator.dev/docs/getting-started/design/#servicemonitor

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
