# Container Storage Interface driver for Hetzner Cloud

[![GitHub Actions status](https://github.com/hetznercloud/csi-driver/workflows/Run%20tests/badge.svg)](https://github.com/hetznercloud/csi-driver/actions)

This is a [Container Storage Interface](https://github.com/container-storage-interface/spec) driver for Hetzner Cloud
enabling you to use ReadWriteOnce Volumes within Kubernetes. Please note that this driver **requires Kubernetes 1.13 or newer**.

## Getting Started

1. Create an API token in the [Hetzner Cloud Console](https://console.hetzner.cloud/).

2. Create a secret containing the token:

   ```
   # secret.yml
   apiVersion: v1
   kind: Secret
   metadata:
     name: hcloud-csi
     namespace: kube-system
   stringData:
     token: YOURTOKEN
   ```

   and apply it:
   ```
   kubectl apply -f <secret.yml>
   ```

3. Deploy the CSI driver and wait until everything is up and running:

    Have a look at our [Version Matrix](README.md#versioning-policy) to pick the correct deployment file.
   ```
   kubectl apply -f https://raw.githubusercontent.com/hetznercloud/csi-driver/v1.5.1/deploy/kubernetes/hcloud-csi.yml
   ```

4. To verify everything is working, create a persistent volume claim and a pod
   which uses that volume:

   ```
   apiVersion: v1
   kind: PersistentVolumeClaim
   metadata:
     name: csi-pvc
   spec:
     accessModes:
     - ReadWriteOnce
     resources:
       requests:
         storage: 10Gi
     storageClassName: hcloud-volumes
   ---
   kind: Pod
   apiVersion: v1
   metadata:
     name: my-csi-app
   spec:
     containers:
       - name: my-frontend
         image: busybox
         volumeMounts:
         - mountPath: "/data"
           name: my-csi-volume
         command: [ "sleep", "1000000" ]
     volumes:
       - name: my-csi-volume
         persistentVolumeClaim:
           claimName: csi-pvc
   ```

   Once the pod is ready, exec a shell and check that your volume is mounted at `/data`.

   ```
   kubectl exec -it my-csi-app -- /bin/sh
   ```

## Integration with Root Servers

Root servers can be part of the cluster, but the CSI plugin doesn't work there. Taint the root server as follows to skip that node for the daemonset.

```bash
kubectl taint node <node name> instance.hetzner.cloud/is-root-server:true
```

## Versioning policy

We aim to support the latest three versions of Kubernetes. After a new
Kubernetes version has been released we will stop supporting the oldest
previously supported version. This does not necessarily mean that the
CSI driver does not still work with this version. However, it means that
we do not test that version anymore. Additionally we will not fix bugs
related only to an unsupported version.

| Kubernetes | CSI Driver    | Deployment File                                                                                    |
| ---------- | -------------:| --------------------------------------------------------------------------------------------------:|
| 1.21       | master        | https://raw.githubusercontent.com/hetznercloud/csi-driver/master/deploy/kubernetes/hcloud-csi.yml  |
| 1.20       | master        | https://raw.githubusercontent.com/hetznercloud/csi-driver/master/deploy/kubernetes/hcloud-csi.yml  |
| 1.19       | 1.5.1, master | https://raw.githubusercontent.com/hetznercloud/csi-driver/v1.5.1/deploy/kubernetes/hcloud-csi.yml  |

## E2E Tests

The Hetzner Cloud CSI Driver was tested against the official k8s e2e
tests for a specific version. You can run the tests with the following
commands. Keep in mind, that these tests run on real cloud servers and
will create volumes that will be billed.

**Development/Testing**

For local development, you will need the following tools installed:

 * Docker
 * Golang 1.16
 * [Skaffold](https://skaffold.dev/)
 * [k3sup](https://github.com/alexellis/k3sup#readme)
 * [hcloud CLI](https://github.com/hetznercloud/cli#readme)

You will also need to set a `HCLOUD_TOKEN` in your shell session:

```sh
 $ export HCLOUD_TOKEN=<token>
```

You can quickly bring up a dev cluster test environment in Hetzner Cloud.

```sh
  $ eval $(INSTANCES=3 hack/dev-up.sh)
  # In about a minute, you should have a 3 node cluster of CPX11 instances (cost is around 2 cents per hour)
  $ kubectl get nodes
  # Now let's run the app.
  $ SKAFFOLD_DEFAULT_REPO=my_dockerhub_username skaffold dev
  # In a minute or two, the project should be built into an image, deployed into the test cluster.
  # Logs will now be tailing out to your shell.
  # If you make changes to the project, the image will be rebuilt and pushed to the cluster, restarting pods as needed.
  # You can even debug the containers running remotely in The Cloud(tm) using standard Golang delve.
  ^C
  $ skaffold debug
  # The logs will indicate which debug ports are available.
  # IMPORTANT! The created servers are not automatically cleaned up. You must remember to delete everything yourself:
  $ hack/dev-down.sh
```

### A note about `SKAFFOLD_DEFAULT_REPO`

When you use Skaffold to deploy the driver to a remote cluster in Hetzner Cloud, you need somewhere to host the images. The default image repository is owned by Hetzner, and thus cannot be used for local development purposes. Instead, you can point Skaffold at your own Docker Hub, ghcr.io, Quay.io, or whatever. The Skaffold docks talk more about [Image Repository Handling](https://skaffold.dev/docs/environment/image-registries/) in gory detail, if you need more information.

Please see the [Skaffold Documentation](https://skaffold.dev/docs/) for more information on the things you can do with Skaffold.

### Running end-to-end tests

Note, these tests will create and detach a *lot* of volumes. You will likely run into API request limits if you run this too frequently.
The tests take 10-20 minutes.


```sh
hack/run-e2e-tests.sh
```

## License

MIT license
