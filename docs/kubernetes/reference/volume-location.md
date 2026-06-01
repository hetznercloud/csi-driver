# Volume Location

During the initialization of the CSI controller, the default location for all volumes is determined based on the following prioritized methods (evaluated in order from 1 to 4). However, when `volumeBindingMode: WaitForFirstConsumer` is used, the volume's location is determined by the node where the Pod is scheduled, and the default location is not applicable. For more details, refer to the official [Kubernetes documentation](https://kubernetes.io/docs/concepts/storage/storage-classes/#volume-binding-mode).

1. The location is explicitly set using the `HCLOUD_VOLUME_DEFAULT_LOCATION` variable.
2. The location is derived by querying a server specified by the `HCLOUD_SERVER_ID` variable.
3. If neither of the above is set, the `KUBE_NODE_NAME` environment variable defaults to the name of the node where the CSI controller is scheduled. This node name is then used to query the Hetzner API for a matching server and its location.
4. As a final fallback, the [Hetzner metadata service](https://docs.hetzner.cloud/reference/cloud#server-metadata) is queried to obtain the server ID, which is then used to fetch the location from the Hetzner API.
