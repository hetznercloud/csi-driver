# Monitoring

Monitoring is an important part of managing any system, and the csi-driver components are no exception.
To help you keep an eye on how the components are performing, we've exposed Prometheus-compatible metrics on port 9189.
You can configure the endpoint for these metrics by setting the `METRICS_ENDPOINT` environment variable to the appropriate value for your system.

The metrics exposed include the following:

- Go Runtime
- gRPC Server for CSI calls
- HTTP calls made to Hetzner Cloud API

## Scraping on Kubernetes

There are two ways to scrape metrics on Kubernetes:

### Using `kubernetes_sd_configs`

If you're using `kubernetes_sd_configs`, you can configure scraping by adding some annotations to the csi-driver workloads.
Specifically, you'll need to add these annotations to the Deployment `hcloud-csi-controller` and the DaemonSet `hcloud-csi-node`:

```yaml
annotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "9189"
```

With these annotations in place, Prometheus should be able to scrape metrics from the csi-driver components.

### prometheus-operator `ServiceMonitor`

If you're using the prometheus-operator, you can use our `ServiceMonitors` to set up scraping.
You can find these ServiceMonitors in [`deploy/kubernetes/service-monitor`](../deploy/kubernetes/service-monitor/).

To use these `ServiceMonitors`, you'll need to replace `ServiceMonitor.metadata.labels.release: YOUR_RELEASE` with the value that you've configured in your `Prometheus` resource.
This will ensure that the `ServiceMonitors` actually scrape the appropriate targets.

## Grafana Dashboard

In addition to scraping metrics, you'll also want a way to visualize those metrics.
To help with this, we provide a default Grafana dashboard that can be used to display the most important metrics.
This dashboard has been confirmed to work with kube-prometheus-stack, but it may require some tweaking to work correctly in your specific environment.
You can find the dashboard at [`deploy/monitoring/grafana-dashboard.json`](../deploy/monitoring/grafana-dashboard.json).
