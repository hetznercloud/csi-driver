# Container Storage Interface driver for Hetzner Cloud

[![GitHub Actions status](https://github.com/hetznercloud/csi-driver/workflows/Run%20tests/badge.svg)](https://github.com/hetznercloud/csi-driver/actions)

This is a [Container Storage Interface](https://github.com/container-storage-interface/spec) driver for Hetzner Cloud
enabling you to use ReadWriteOnce Volumes within Kubernetes. Please note that this driver **requires Kubernetes 1.19 or newer**.

## Getting Started

### :stop_sign: There is a known bug in `v2.0.0`, if you are using this version, please check out the issue [#330](https://github.com/hetznercloud/csi-driver/issues/333) to learn about the impact. :stop_sign: 

1. Create a read+write API token in the [Hetzner Cloud Console](https://console.hetzner.cloud/).

2. Create a secret containing the token:

   **(v2.x):**
   ```
   # secret.yml
   apiVersion: v1
   kind: Secret
   metadata:
     name: hcloud
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
   kubectl apply -f https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.1.0/deploy/kubernetes/hcloud-csi.yml
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

5. To add encryption with LUKS you have to create a dedicate secret containing an encryption passphrase and duplicate the default `hcloud-volumes` storage class with added parameters referencing this secret:

   ```
   apiVersion: v1
   kind: Secret
   metadata:
     name: encryption-secret
     namespace: kube-system
   stringData:
     encryption-passphrase: foobar

   --- 

   apiVersion: storage.k8s.io/v1
   kind: StorageClass
   metadata:
     name: hcloud-volumes-encrypted
   provisioner: csi.hetzner.cloud
   reclaimPolicy: Delete
   volumeBindingMode: WaitForFirstConsumer
   allowVolumeExpansion: true
   parameters:
     csi.storage.k8s.io/node-publish-secret-name: encryption
     csi.storage.k8s.io/node-publish-secret-namespace: default
   ```

## Upgrading

To upgrade the csi-driver version, you just need to apply the new manifests to your cluster.

In case of a new major version, there might be manual steps that you need to follow to upgrade the csi-driver. See the following section for a list of major updates and their required steps.

### From v1 to v2

There are two breaking changes between v1.6 and v2.0 that require user intervention. Please take care to follow these steps, as otherwise the update might fail.

**Before the rollout**:

1. The secret containing the API token was renamed from `hcloud-csi` to `hcloud`. This change was made so both the cloud-controller-manager and the csi-driver can use the same secret. Check that you have a secret `hcloud` in the namespace `kube-system`, and that the secret contains the API token, as described in the section [Getting Started](#getting-started):

   ```shell
   $ kubectl get secret -n kube-system hcloud
   ```

2. We added a new field to our `CSIDriver` resource to support [CSI volume fsGroup policy management](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#configure-volume-permission-and-ownership-change-policy-for-pods). This change requires a replacement of the `CSIDriver` object. You need to manually delete the old object:

   ```shell
   $ kubectl delete csidriver csi.hetzner.cloud
   ```

   The new `CSIDriver` will be installed when you apply the new manifests.

3. Stop the old pods to make sure that only everything is replaced in order and no incompatible pods are running side-by-side:

   ```shell
   $ kubectl delete statefulset -n kube-system hcloud-csi-controller
   $ kubectl delete daemonset -n kube-system hcloud-csi-node
   ```

**Rollout the new manifest**:

```shell
$ kubectl apply -f https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.1.0/deploy/kubernetes/hcloud-csi.yml
```

**After the rollout**:

1. Delete the now unused secret `hcloud-csi` in the namespace `kube-system`:

   ```shell
   $ kubectl delete secret -n kube-system hcloud-csi
   ```

2. Remove old resources that have been replaced:

   ```shell
   $ kubectl delete clusterrolebinding hcloud-csi
   $ kubectl delete clusterrole hcloud-csi
   $ kubectl delete serviceaccount -n kube-system hcloud-csi
   ```

## Integration with Root Servers

Root servers can be part of the cluster, but the CSI plugin doesn't work there. Taint the root server as follows to skip that node for the daemonset.

```bash
kubectl label nodes <node name> instance.hetzner.cloud/is-root-server=true
```

## Versioning policy

We aim to support the latest three versions of Kubernetes. After a new
Kubernetes version has been released we will stop supporting the oldest
previously supported version. This does not necessarily mean that the
CSI driver does not still work with this version. However, it means that
we do not test that version anymore. Additionally, we will not fix bugs
related only to an unsupported version.

| Kubernetes | CSI Driver |                                                                                   Deployment File |
| ---------- | ---------: | ------------------------------------------------------------------------------------------------: |
| 1.25       |      2.1.0 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.1.0/deploy/kubernetes/hcloud-csi.yml |
| 1.24       |      2.1.0 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.1.0/deploy/kubernetes/hcloud-csi.yml |
| 1.23       |      2.1.0 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.1.0/deploy/kubernetes/hcloud-csi.yml |
| 1.22       |      1.6.0 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v1.6.0/deploy/kubernetes/hcloud-csi.yml |
| 1.21       |      1.6.0 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v1.6.0/deploy/kubernetes/hcloud-csi.yml |
| 1.20       |      1.6.0 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v1.6.0/deploy/kubernetes/hcloud-csi.yml |

## Integration Tests

**Requirements: Docker**

The core operations like publishing and resizing can be tested locally with Docker.

```bash
go test $(go list ./... | grep integrationtests) -v
```

## E2E Tests

The Hetzner Cloud CSI Driver was tested against the official k8s e2e
tests for a specific version. You can run the tests with the following
commands. Keep in mind, that these tests run on real cloud servers and
will create volumes that will be billed.

**Test Server Setup**:

1x CPX21 (Ubuntu 18.04)

**Requirements: Docker and Go 1.17**

1. Configure your environment correctly
   ```bash
   export HCLOUD_TOKEN=<specifiy a project token>
   export K8S_VERSION=1.21.0 # The specific (latest) version is needed here
   export USE_SSH_KEYS=key1,key2 # Name or IDs of your SSH Keys within the Hetzner Cloud, the servers will be accessable with that keys
   ```
2. Run the tests
   ```bash
   go test $(go list ./... | grep e2etests) -v -timeout 60m
   ```

The tests will now run, this will take a while (~30 min).

**If the tests fail, make sure to clean up the project with the Hetzner Cloud Console or the hcloud cli.**

## Local test setup
This repository provides [skaffold](https://skaffold.dev/) to easily deploy / debug this driver on demand

### Requirements
1. Install [hcloud-cli](https://github.com/hetznercloud/cli)
2. Install [k3sup](https://github.com/alexellis/k3sup)
3. Install [cilium](https://github.com/cilium/cilium-cli)
4. Install [docker](https://www.docker.com/)

You will also need to set a `HCLOUD_TOKEN` in your shell session
### Manual Installation guide
1. Create an SSH key

Assuming you already have created an ssh key via `ssh-keygen`
```
hcloud ssh-key create --name ssh-key-csi-test --public-key-from-file ~/.ssh/id_rsa.pub 
```

2. Create a server
```
hcloud server create --name csi-test-server --image ubuntu-20.04 --ssh-key ssh-key-csi-test --type cx11 
```

3. Setup k3s on this server
```
k3sup install --ip $(hcloud server ip csi-test-server) --local-path=/tmp/kubeconfig --cluster --k3s-channel=v1.23 --k3s-extra-args='--no-flannel --no-deploy=servicelb --no-deploy=traefik --disable-cloud-controller --disable-network-policy --kubelet-arg=cloud-provider=external'
```
- The kubeconfig will be created under `/tmp/kubeconfig`
- Kubernetes version can be configured via `--k3s-channel`

4. Switch your kubeconfig to the test cluster
```
export KUBECONFIG=/tmp/kubeconfig
```

5. Install cilium + test your cluster
```
cilium install
```

6. Add your secret to the cluster
```
kubectl -n kube-system create secret generic hcloud --from-literal="token=$HCLOUD_TOKEN"
```

7. Install hcloud-cloud-controller-manager + test your cluster
```
kubectl apply -f  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm.yaml
kubectl config set-context default
kubectl get node -o wide
```

8. Deploy your CSI driver
```
SKAFFOLD_DEFAULT_REPO=naokiii skaffold dev
```
- `docker login` required
- Skaffold is using your own dockerhub repo to push the CSI image.

On code change, skaffold will repack the image & deploy it to your test cluster again. Also, it is printing all logs from csi components.

## License

MIT license
