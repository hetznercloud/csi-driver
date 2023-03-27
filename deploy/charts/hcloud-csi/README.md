
# Hcloud CSI

This deploys a production-ready helm chart for the Hcloud CSI.

## TL;DR

```console
helm repo add syself https://charts.syself.com
helm install csi syself/hcloud-csi
```

## Introduction


## Prerequisites

- Kubernetes 1.19+
- Helm 3.2.0+

## Installing the Chart

To install the chart with the release name `csi`:

```console
helm install csi syself/hcloud-csi
```

The command deploys hcloud-csi on the Kubernetes cluster in the default configuration. The [Parameters](#parameters) section lists the parameters that can be configured during installation.

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `csi` deployment:

```console
helm delete csi
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Parameters

### Global parameters

| Name                      | Description                                     | Value |
| ------------------------- | ----------------------------------------------- | ----- |
| `global.imageRegistry`    | Global Docker image registry                    | `""`  |
| `global.imagePullSecrets` | Global Docker registry secret names as an array | `[]`  |
| `global.storageClass`     | Global StorageClass for Persistent Volume(s)    | `""`  |

### Common parameters

| Name                | Description                                       | Value |
| ------------------- | ------------------------------------------------- | ----- |
| `nameOverride`      | String to partially override common.names.name    | `""`  |
| `fullnameOverride`  | String to fully override common.names.fullname    | `""`  |
| `namespaceOverride` | String to fully override common.names.namespace   | `""`  |
| `commonLabels`      | Labels to add to all deployed objects             | `{}`  |
| `commonAnnotations` | Annotations to add to all deployed objects        | `{}`  |
| `extraDeploy`       | Array of extra objects to deploy with the release | `[]`  |

### Controller Parameters

| Name                                            | Description                                                                                                                                                  | Value                            |
| ----------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------- |
| `controller.image.csiAttacher.registry`         | csi-attacher image registry                                                                                                                                  | `registry.k8s.io`                |
| `controller.image.csiAttacher.repository`       | csi-attacher image repository                                                                                                                                | `sig-storage/csi-attacher`       |
| `controller.image.csiAttacher.tag`              | csi-attacher image tag (immutable tags are recommended)                                                                                                      | `v4.1.0`                         |
| `controller.image.csiAttacher.digest`           | csi-attacher image digest in the way sha256:aa.... Please note this parameter, if set, will override the tag image tag (immutable tags are recommended)      | `""`                             |
| `controller.image.csiAttacher.pullPolicy`       | csi-attacher image pull policy                                                                                                                               | `IfNotPresent`                   |
| `controller.image.csiAttacher.pullSecrets`      | csi-attacher image pull secrets                                                                                                                              | `[]`                             |
| `controller.image.csiResizer.registry`          | csi-resizer image registry                                                                                                                                   | `registry.k8s.io`                |
| `controller.image.csiResizer.repository`        | csi-resizer image repository                                                                                                                                 | `sig-storage/csi-resizer`        |
| `controller.image.csiResizer.tag`               | csi-resizer image tag (immutable tags are recommended)                                                                                                       | `v1.7.0`                         |
| `controller.image.csiResizer.digest`            | csi-resizer image digest in the way sha256:aa.... Please note this parameter, if set, will override the tag image tag (immutable tags are recommended)       | `""`                             |
| `controller.image.csiResizer.pullPolicy`        | csi-resizer image pull policy                                                                                                                                | `IfNotPresent`                   |
| `controller.image.csiResizer.pullSecrets`       | csi-resizer image pull secrets                                                                                                                               | `[]`                             |
| `controller.image.csiProvisioner.registry`      | csi-provisioner image registry                                                                                                                               | `registry.k8s.io`                |
| `controller.image.csiProvisioner.repository`    | csi-provisioner image repository                                                                                                                             | `sig-storage/csi-provisioner`    |
| `controller.image.csiProvisioner.tag`           | csi-provisioner image tag (immutable tags are recommended)                                                                                                   | `v3.4.0`                         |
| `controller.image.csiProvisioner.digest`        | csi-provisioner image digest in the way sha256:aa.... Please note this parameter, if set, will override the tag image tag (immutable tags are recommended)   | `""`                             |
| `controller.image.csiProvisioner.pullPolicy`    | csi-provisioner image pull policy                                                                                                                            | `IfNotPresent`                   |
| `controller.image.csiProvisioner.pullSecrets`   | csi-provisioner image pull secrets                                                                                                                           | `[]`                             |
| `controller.image.livenessProbe.registry`       | liveness-probe image registry                                                                                                                                | `registry.k8s.io`                |
| `controller.image.livenessProbe.repository`     | liveness-probe image repository                                                                                                                              | `sig-storage/livenessprobe`      |
| `controller.image.livenessProbe.tag`            | liveness-probe image tag (immutable tags are recommended)                                                                                                    | `v2.9.0`                         |
| `controller.image.livenessProbe.digest`         | liveness-probe image digest in the way sha256:aa.... Please note this parameter, if set, will override the tag image tag (immutable tags are recommended)    | `""`                             |
| `controller.image.livenessProbe.pullPolicy`     | liveness-probe image pull policy                                                                                                                             | `IfNotPresent`                   |
| `controller.image.livenessProbe.pullSecrets`    | liveness-probe image pull secrets                                                                                                                            | `[]`                             |
| `controller.image.hcloudCSIDriver.registry`     | hcloud-csi-driver image registry                                                                                                                             | `docker.io`                      |
| `controller.image.hcloudCSIDriver.repository`   | hcloud-csi-driver image repository                                                                                                                           | `hetznercloud/hcloud-csi-driver` |
| `controller.image.hcloudCSIDriver.tag`          | hcloud-csi-driver image tag (immutable tags are recommended)                                                                                                 | `2.2.0`                          |
| `controller.image.hcloudCSIDriver.digest`       | hcloud-csi-driver image digest in the way sha256:aa.... Please note this parameter, if set, will override the tag image tag (immutable tags are recommended) | `""`                             |
| `controller.image.hcloudCSIDriver.pullPolicy`   | hcloud-csi-driver image pull policy                                                                                                                          | `IfNotPresent`                   |
| `controller.image.hcloudCSIDriver.pullSecrets`  | hcloud-csi-driver image pull secrets                                                                                                                         | `[]`                             |
| `controller.replicaCount`                       | Number of controller replicas to deploy                                                                                                                      | `2`                              |
| `controller.hcloudToken.value`                  | Specifies the value for the hcloudToken. Creates a secret from that value. If you have already a hcloud token secret leave this empty.                       | `""`                             |
| `controller.hcloudToken.extistingSecret.name`   | Specifies the name of an existing Secret for the hcloud Token                                                                                                | `hcloud`                         |
| `controller.hcloudToken.extistingSecret.key`    | Specifies the key of an existing Secret for the hcloud Token                                                                                                 | `token`                          |
| `controller.hcloudVolumeDefaultLocation`        | Set this to the location of your cluster. If set the controller could run anywhere. If leave empty the controller needs to run on a hcloud node.             | `""`                             |
| `controller.containerPorts.metrics`             | controller metrics container port                                                                                                                            | `9189`                           |
| `controller.containerPorts.healthz`             | controller healthz container port                                                                                                                            | `9808`                           |
| `controller.service.ports.metrics`              | controller service metrics port                                                                                                                              | `9189`                           |
| `controller.service.annotations`                | Additional custom annotations for controller service                                                                                                         | `{}`                             |
| `controller.rbac.create`                        | Specifies whether RBAC resources should be created                                                                                                           | `true`                           |
| `controller.rbac.rules`                         | Custom RBAC rules to set                                                                                                                                     | `[]`                             |
| `controller.livenessProbe.enabled`              | Enable livenessProbe on controller containers                                                                                                                | `true`                           |
| `controller.livenessProbe.initialDelaySeconds`  | Initial delay seconds for livenessProbe                                                                                                                      | `10`                             |
| `controller.livenessProbe.periodSeconds`        | Period seconds for livenessProbe                                                                                                                             | `2`                              |
| `controller.livenessProbe.timeoutSeconds`       | Timeout seconds for livenessProbe                                                                                                                            | `3`                              |
| `controller.livenessProbe.failureThreshold`     | Failure threshold for livenessProbe                                                                                                                          | `5`                              |
| `controller.livenessProbe.successThreshold`     | Success threshold for livenessProbe                                                                                                                          | `1`                              |
| `controller.customLivenessProbe`                | Custom livenessProbe that overrides the default one                                                                                                          | `{}`                             |
| `controller.customReadinessProbe`               | Custom readinessProbe that overrides the default one                                                                                                         | `{}`                             |
| `controller.customStartupProbe`                 | Custom startupProbe that overrides the default one                                                                                                           | `{}`                             |
| `controller.resources.csiAttacher.limits`       | The resources limits for the csiAttacher containers                                                                                                          | `{}`                             |
| `controller.resources.csiAttacher.requests`     | The requested resources for the csiAttacher containers                                                                                                       | `{}`                             |
| `controller.resources.csiResizer.limits`        | The resources limits for the csiResizer containers                                                                                                           | `{}`                             |
| `controller.resources.csiResizer.requests`      | The requested resources for the csiResizer containers                                                                                                        | `{}`                             |
| `controller.resources.csiProvisioner.limits`    | The resources limits for the csiProvisioner containers                                                                                                       | `{}`                             |
| `controller.resources.csiProvisioner.requests`  | The requested resources for the csiProvisioner containers                                                                                                    | `{}`                             |
| `controller.resources.livenessProbe.limits`     | The resources limits for the livenessProbe containers                                                                                                        | `{}`                             |
| `controller.resources.livenessProbe.requests`   | The requested resources for the livenessProbe containers                                                                                                     | `{}`                             |
| `controller.resources.hcloudCSIDriver.limits`   | The resources limits for the hcloudCSIDriver containers                                                                                                      | `{}`                             |
| `controller.resources.hcloudCSIDriver.requests` | The requested resources for the hcloudCSIDriver containers                                                                                                   | `{}`                             |
| `controller.podSecurityContext.enabled`         | Enabled controller pods' Security Context                                                                                                                    | `true`                           |
| `controller.podSecurityContext.fsGroup`         | Set controller pod's Security Context fsGroup                                                                                                                | `1001`                           |
| `controller.podLabels`                          | Extra labels for controller pods                                                                                                                             | `{}`                             |
| `controller.podAnnotations`                     | Annotations for controller pods                                                                                                                              | `{}`                             |
| `controller.pdb.create`                         | Enable PodDisruptionBudged for controller pods                                                                                                               | `true`                           |
| `controller.pdb.minAvailable`                   | Set minAvailable for controller pods                                                                                                                         | `1`                              |
| `controller.pdb.maxUnavailable`                 | Set maxUnavailable for controller pods                                                                                                                       | `""`                             |
| `controller.autoscaling.enabled`                | Enable autoscaling for controller                                                                                                                            | `false`                          |
| `controller.autoscaling.minReplicas`            | Minimum number of controller replicas                                                                                                                        | `""`                             |
| `controller.autoscaling.maxReplicas`            | Maximum number of controller replicas                                                                                                                        | `""`                             |
| `controller.autoscaling.targetCPU`              | Target CPU utilization percentage                                                                                                                            | `""`                             |
| `controller.autoscaling.targetMemory`           | Target Memory utilization percentage                                                                                                                         | `""`                             |
| `controller.affinity`                           | Affinity for controller pods assignment                                                                                                                      | `{}`                             |
| `controller.nodeSelector`                       | Node labels for controller pods assignment                                                                                                                   | `{}`                             |
| `controller.tolerations`                        | Tolerations for controller pods assignment                                                                                                                   | `[]`                             |
| `controller.updateStrategy.type`                | controller statefulset strategy type                                                                                                                         | `RollingUpdate`                  |
| `controller.priorityClassName`                  | controller pods' priorityClassName                                                                                                                           | `""`                             |
| `controller.topologySpreadConstraints`          | Topology Spread Constraints for pod assignment spread across your cluster among failure-domains. Evaluated as a template                                     | `[]`                             |
| `controller.schedulerName`                      | Name of the k8s scheduler (other than default) for controller pods                                                                                           | `""`                             |
| `controller.terminationGracePeriodSeconds`      | Seconds Redmine pod needs to terminate gracefully                                                                                                            | `""`                             |
| `controller.lifecycleHooks`                     | for the controller container(s) to automate configuration before or after startup                                                                            | `{}`                             |
| `controller.extraEnvVars`                       | Array with extra environment variables to add to controller nodes                                                                                            | `[]`                             |
| `controller.extraVolumes`                       | Extra Volumes for controller pods                                                                                                                            | `[]`                             |
| `controller.extraVolumeMounts`                  | Optionally specify extra list of additional volumeMounts for the controller container(s)                                                                     | `[]`                             |
| `controller.sidecars`                           | Add additional sidecar containers to the controller pod(s)                                                                                                   | `[]`                             |
| `controller.initContainers`                     | Add additional init containers to the controller pod(s)                                                                                                      | `[]`                             |

### Node Parameters

| Name                                             | Description                                                                                                                                                          | Value                                   |
| ------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------- |
| `node.image.CSINodeDriverRegistrar.registry`     | csi-node-driver-registrar image registry                                                                                                                             | `registry.k8s.io`                       |
| `node.image.CSINodeDriverRegistrar.repository`   | csi-node-driver-registrar image repository                                                                                                                           | `sig-storage/csi-node-driver-registrar` |
| `node.image.CSINodeDriverRegistrar.tag`          | csi-node-driver-registrar image tag (immutable tags are recommended)                                                                                                 | `v2.7.0`                                |
| `node.image.CSINodeDriverRegistrar.digest`       | csi-node-driver-registrar image digest in the way sha256:aa.... Please note this parameter, if set, will override the tag image tag (immutable tags are recommended) | `""`                                    |
| `node.image.CSINodeDriverRegistrar.pullPolicy`   | csi-node-driver-registrar image pull policy                                                                                                                          | `IfNotPresent`                          |
| `node.image.CSINodeDriverRegistrar.pullSecrets`  | csi-node-driver-registrar image pull secrets                                                                                                                         | `[]`                                    |
| `node.image.livenessProbe.registry`              | liveness-probe image registry                                                                                                                                        | `registry.k8s.io`                       |
| `node.image.livenessProbe.repository`            | liveness-probe image repository                                                                                                                                      | `sig-storage/livenessprobe`             |
| `node.image.livenessProbe.tag`                   | liveness-probe image tag (immutable tags are recommended)                                                                                                            | `v2.9.0`                                |
| `node.image.livenessProbe.digest`                | liveness-probe image digest in the way sha256:aa.... Please note this parameter, if set, will override the tag image tag (immutable tags are recommended)            | `""`                                    |
| `node.image.livenessProbe.pullPolicy`            | liveness-probe image pull policy                                                                                                                                     | `IfNotPresent`                          |
| `node.image.livenessProbe.pullSecrets`           | liveness-probe image pull secrets                                                                                                                                    | `[]`                                    |
| `node.image.hcloudCSIDriver.registry`            | hcloud-csi-driver image registry                                                                                                                                     | `docker.io`                             |
| `node.image.hcloudCSIDriver.repository`          | hcloud-csi-driver image repository                                                                                                                                   | `hetznercloud/hcloud-csi-driver`        |
| `node.image.hcloudCSIDriver.tag`                 | hcloud-csi-driver image tag (immutable tags are recommended)                                                                                                         | `2.2.0`                                 |
| `node.image.hcloudCSIDriver.digest`              | hcloud-csi-driver image digest in the way sha256:aa.... Please note this parameter, if set, will override the tag image tag (immutable tags are recommended)         | `""`                                    |
| `node.image.hcloudCSIDriver.pullPolicy`          | hcloud-csi-driver image pull policy                                                                                                                                  | `IfNotPresent`                          |
| `node.image.hcloudCSIDriver.pullSecrets`         | hcloud-csi-driver image pull secrets                                                                                                                                 | `[]`                                    |
| `node.containerPorts.metrics`                    | node Metrics container port                                                                                                                                          | `9189`                                  |
| `node.containerPorts.healthz`                    | node Health container port                                                                                                                                           | `9808`                                  |
| `node.service.ports.metrics`                     | node service Metrics port                                                                                                                                            | `9189`                                  |
| `node.service.annotations`                       | Additional custom annotations for node service                                                                                                                       | `{}`                                    |
| `node.livenessProbe.enabled`                     | Enable livenessProbe on node containers                                                                                                                              | `true`                                  |
| `node.livenessProbe.initialDelaySeconds`         | Initial delay seconds for livenessProbe                                                                                                                              | `10`                                    |
| `node.livenessProbe.periodSeconds`               | Period seconds for livenessProbe                                                                                                                                     | `2`                                     |
| `node.livenessProbe.timeoutSeconds`              | Timeout seconds for livenessProbe                                                                                                                                    | `3`                                     |
| `node.livenessProbe.failureThreshold`            | Failure threshold for livenessProbe                                                                                                                                  | `5`                                     |
| `node.livenessProbe.successThreshold`            | Success threshold for livenessProbe                                                                                                                                  | `1`                                     |
| `node.customLivenessProbe`                       | Custom livenessProbe that overrides the default one                                                                                                                  | `{}`                                    |
| `node.customReadinessProbe`                      | Custom readinessProbe that overrides the default one                                                                                                                 | `{}`                                    |
| `node.customStartupProbe`                        | Custom startupProbe that overrides the default one                                                                                                                   | `{}`                                    |
| `node.resources.CSINodeDriverRegistrar.limits`   | The resources limits for the CSINodeDriverRegistrar containers                                                                                                       | `{}`                                    |
| `node.resources.CSINodeDriverRegistrar.requests` | The requested resources for the CSINodeDriverRegistrar containers                                                                                                    | `{}`                                    |
| `node.resources.livenessProbe.limits`            | The resources limits for the livenessProbe containers                                                                                                                | `{}`                                    |
| `node.resources.livenessProbe.requests`          | The requested resources for the livenessProbe containers                                                                                                             | `{}`                                    |
| `node.resources.hcloudCSIDriver.limits`          | The resources limits for the hcloudCSIDriver containers                                                                                                              | `{}`                                    |
| `node.resources.hcloudCSIDriver.requests`        | The requested resources for the hcloudCSIDriver containers                                                                                                           | `{}`                                    |
| `node.podSecurityContext.enabled`                | Enabled node pods' Security Context                                                                                                                                  | `true`                                  |
| `node.podSecurityContext.fsGroup`                | Set node pod's Security Context fsGroup                                                                                                                              | `1001`                                  |
| `node.hostNetwork`                               | Enables the hostNetwork                                                                                                                                              | `false`                                 |
| `node.podLabels`                                 | Extra labels for node pods                                                                                                                                           | `{}`                                    |
| `node.podAnnotations`                            | Annotations for node pods                                                                                                                                            | `{}`                                    |
| `node.affinity`                                  | Affinity for node pods assignment                                                                                                                                    | `{}`                                    |
| `node.nodeSelector`                              | Node labels for node pods assignment                                                                                                                                 | `{}`                                    |
| `node.tolerations`                               | Tolerations for node pods assignment                                                                                                                                 | `{}`                                    |
| `node.updateStrategy.type`                       | node statefulset strategy type                                                                                                                                       | `RollingUpdate`                         |
| `node.priorityClassName`                         | node pods' priorityClassName                                                                                                                                         | `""`                                    |
| `node.schedulerName`                             | Name of the k8s scheduler (other than default) for node pods                                                                                                         | `""`                                    |
| `node.terminationGracePeriodSeconds`             | Seconds Redmine pod needs to terminate gracefully                                                                                                                    | `""`                                    |
| `node.lifecycleHooks`                            | for the node container(s) to automate configuration before or after startup                                                                                          | `{}`                                    |
| `node.extraVolumes`                              | Extra Volumes for controller pods                                                                                                                                    | `[]`                                    |
| `node.extraVolumeMounts`                         | Optionally specify extra list of additional volumeMounts for the node container(s)                                                                                   | `[]`                                    |
| `node.sidecars`                                  | Add additional sidecar containers to the node pod(s)                                                                                                                 | `[]`                                    |
| `node.initContainers`                            | Add additional init containers to the node pod(s)                                                                                                                    | `[]`                                    |

### Other Parameters

| Name                                          | Description                                                                                            | Value   |
| --------------------------------------------- | ------------------------------------------------------------------------------------------------------ | ------- |
| `serviceAccount.create`                       | Specifies whether a ServiceAccount should be created                                                   | `true`  |
| `serviceAccount.name`                         | The name of the ServiceAccount to use.                                                                 | `""`    |
| `serviceAccount.annotations`                  | Additional Service Account annotations (evaluated as a template)                                       | `{}`    |
| `serviceAccount.automountServiceAccountToken` | Automount service account token for the server service account                                         | `true`  |
| `metrics.enabled`                             | Enable the export of Prometheus metrics                                                                | `false` |
| `metrics.serviceMonitor.enabled`              | if `true`, creates a Prometheus Operator ServiceMonitor (also requires `metrics.enabled` to be `true`) | `false` |
| `metrics.serviceMonitor.namespace`            | Namespace in which Prometheus is running                                                               | `""`    |
| `metrics.serviceMonitor.annotations`          | Additional custom annotations for the ServiceMonitor                                                   | `{}`    |
| `metrics.serviceMonitor.labels`               | Extra labels for the ServiceMonitor                                                                    | `{}`    |
| `metrics.serviceMonitor.jobLabel`             | The name of the label on the target service to use as the job name in Prometheus                       | `""`    |
| `metrics.serviceMonitor.honorLabels`          | honorLabels chooses the metric's labels on collisions with target labels                               | `false` |
| `metrics.serviceMonitor.interval`             | Interval at which metrics should be scraped.                                                           | `""`    |
| `metrics.serviceMonitor.scrapeTimeout`        | Timeout after which the scrape is ended                                                                | `""`    |
| `metrics.serviceMonitor.metricRelabelings`    | Specify additional relabeling of metrics                                                               | `[]`    |
| `metrics.serviceMonitor.relabelings`          | Specify general relabeling                                                                             | `[]`    |
| `metrics.serviceMonitor.selector`             | Prometheus instance selector labels                                                                    | `{}`    |
| `storageClasses`                              | Creates one or more storageClasses                                                                     | `{}`    |



## Additional environment variables

In case you want to add extra environment variables (useful for advanced operations like custom init scripts), you can use the `extraEnvVars` property.

```yaml
controller:
  extraEnvVars:
    - name: LOG_LEVEL
      value: debug
```

## Pod affinity

This chart allows you to set your custom affinity using the `affinity` parameter. Find more information about Pod affinity in the [kubernetes documentation](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity).


