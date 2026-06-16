# Setting up a cluster and creating your first Volume

In this tutorial, you will learn how to set up a lightweight Kubernetes cluster on **Hetzner Cloud** using **k3s**, install the **Hetzner Cloud csi-driver**, and create a Deployment with a PersistentVolumeClaim.

**What you’ll achieve:**

- A running Kubernetes cluster with nodes in Hetzner Cloud
- A running Deployment, which writes data to a Hetzner Cloud Volume

**Prerequisites:**

- Basic knowledge of Kubernetes concepts
- Installed command-line tools:
  - [hcloud CLI](https://github.com/hetznercloud/cli)
  - [k3sup](https://github.com/alexellis/k3sup)
  - [kubectl](https://kubernetes.io/docs/tasks/tools/)
  - [Helm](https://helm.sh/docs/intro/install/)
  - [jq](https://jqlang.org/)

---

## 1. Set up Hetzner Cloud Resources

### 1.1. Create SSH Key

Lets create and upload a SSH key, which is used by k3sup to access our servers during the Kubernetes installation process.

```bash
ssh-keygen -t ed25519 -f ./hcloud-k3s
hcloud ssh-key create --name k3s-key --public-key-from-file ./hcloud-k3s.pub
```

### 1.2. Create Control Plane and Worker Nodes

Our cluster will consist of a single control plane with a single worker. These servers will be located in Helsinki, use Ubuntu as a base image and use the server-type cx23.

```bash
hcloud server create --name tutorial-control-plane \
  --type cx23 \
  --location hel1 \
  --image ubuntu-24.04 \
  --ssh-key k3s-key

hcloud server create --name tutorial-worker \
  --type cx23 \
  --location hel1 \
  --image ubuntu-24.04 \
  --ssh-key k3s-key
```

Use `hcloud server list` to check that the servers are running and note their public IP addresses.

---

## 2. Deploy k3s Cluster

### 2.1. Install k3s on Control Plane

```bash
k3sup install \
  --ip=<CONTROL_PLANE_PUBLIC_IP> \
  --ssh-key ./hcloud-k3s \
  --local-path=./kubeconfig \
  --k3s-extra-args="\
    --kubelet-arg=cloud-provider=external \
    --disable-cloud-controller \
    --disable-network-policy \
    --disable=traefik \
    --disable=servicelb \
    --disable=local-storage \
    --node-ip='<CONTROL_PLANE_PUBLIC_IP>'"
```

- `cloud-provider=external` prepares the cluster for an external cloud controller. In our case the hcloud-cloud-controller-manager.
- `--disable-network-policy`, `--disable=traefik` and `--disable=servicelb` removes k3s builtin components, which would collide with the products we deploy in this tutorial.
- `--disable=local-storage` removes the local-path StorageClass that k3s ships with, so that the `hcloud-volumes` StorageClass we install later becomes the cluster's default.

### 2.2. Join Worker Node

```bash
k3sup join \
  --ip=<WORKER_PUBLIC_IP> \
  --server-ip=<CONTROL_PLANE_PUBLIC_IP> \
  --user=root \
  --ssh-key ./hcloud-k3s \
  --k3s-extra-args="\
    --kubelet-arg=cloud-provider=external \
    --node-ip='<WORKER_PUBLIC_IP>'"
```

Set the kubeconfig file:

```bash
export KUBECONFIG=./kubeconfig
kubectl get nodes -o wide
```

---

## 3. Install Hetzner Cloud csi-driver

### 3.1. Create Hetzner Cloud API Token Secret

```bash
kubectl -n kube-system create secret generic hcloud --from-literal=token=<YOUR_HCLOUD_API_TOKEN>
```

### 3.2. Install csi-driver via Helm

```bash
helm repo add hcloud https://charts.hetzner.cloud
helm repo update
helm install hcloud-csi hcloud/hcloud-csi -n kube-system
```

---

## 4. Initialize the nodes with the cloud-controller-manager

When we created the cluster, we passed `--kubelet-arg=cloud-provider=external` and `--disable-cloud-controller`. This tells the kubelet that an _external_ cloud controller is responsible for initializing the nodes. Until that controller runs, every node carries the taint `node.cloudprovider.kubernetes.io/uninitialized=true:NoSchedule`, which prevents regular workloads — including the `hcloud-csi-controller` we just installed — from being scheduled.

You can see this for yourself:

```bash
kubectl -n kube-system get pods -l app=hcloud-csi-controller
# The controller Pod stays Pending until the nodes are initialized.
```

Install the `hcloud-cloud-controller-manager`, which reuses the `hcloud` secret and Helm repository we set up earlier:

```bash
helm install hcloud-ccm hcloud/hcloud-cloud-controller-manager -n kube-system
```

Once it is running, it removes the taint, sets the `providerID` and adds topology labels (such as `topology.kubernetes.io/region`) to each node.

```bash
kubectl -n kube-system rollout status deployment/hcloud-cloud-controller-manager
kubectl get nodes -o wide
```

All nodes should now be `Ready`, and the `hcloud-csi-controller` Pod should have moved to `Running`.

---

## 5. Create your first PersistentVolumeClaim

The csi-driver Helm chart created a default StorageClass named `hcloud-volumes`:

```bash
kubectl get storageclass
```

```
NAME                       PROVISIONER         RECLAIMPOLICY   VOLUMEBINDINGMODE      ALLOWVOLUMEEXPANSION   AGE
hcloud-volumes (default)   csi.hetzner.cloud   Delete          WaitForFirstConsumer   true                   2m
```

Two properties of this StorageClass matter for the rest of the tutorial:

- **`volumeBindingMode: WaitForFirstConsumer`** — the volume is only created once a Pod that uses the claim is scheduled. A freshly created PVC therefore stays `Pending` until then. This makes sure the volume is created in the same location as the Pod that will use it.
- **`reclaimPolicy: Delete`** — when you delete the PVC, the underlying Hetzner Cloud Volume is deleted as well.

Create a file called `pvc-app.yaml` containing a PersistentVolumeClaim and a Deployment that mounts it:

```yaml
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
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-csi-app
spec:
  replicas: 1
  # A Hetzner Cloud Volume (ReadWriteOnce) can only be attached to one node at
  # a time, so we replace the old Pod before starting a new one.
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app: my-csi-app
  template:
    metadata:
      labels:
        app: my-csi-app
    spec:
      containers:
        - name: my-frontend
          image: busybox
          command: ["sleep", "infinity"]
          volumeMounts:
            - name: my-csi-volume
              mountPath: /data
      volumes:
        - name: my-csi-volume
          persistentVolumeClaim:
            claimName: csi-pvc
```

The minimum size for a Hetzner Cloud Volume is 10 GiB, so we request `10Gi`.

Apply the manifest:

```bash
kubectl apply -f pvc-app.yaml
```

---

## 6. Verify the volume

Watch the PVC move from `Pending` to `Bound`. This happens as soon as the Deployment's Pod is scheduled and the csi-driver has provisioned the volume:

```bash
kubectl get pvc csi-pvc -w
```

```
NAME      STATUS    VOLUME        CAPACITY   ACCESS MODES   STORAGECLASS     AGE
csi-pvc   Pending                                          hcloud-volumes   5s
csi-pvc   Bound     pvc-7b1f...   10Gi       RWO            hcloud-volumes   20s
```

Confirm that the Pod is running:

```bash
kubectl get pods -l app=my-csi-app
```

You can also see the newly created volume in Hetzner Cloud. Its name starts with `pvc-` and matches the volume bound above:

```bash
hcloud volume list
```

---

## 7. Write data and confirm it persists

Write a file into the mounted volume:

```bash
kubectl exec deploy/my-csi-app -- sh -c 'echo "Hello from Hetzner Cloud Volumes" > /data/hello.txt'
kubectl exec deploy/my-csi-app -- cat /data/hello.txt
```

To prove the data lives on the volume and not inside the Pod, delete the Pod. The Deployment recreates it, the same volume is re-attached, and the file is still there:

```bash
kubectl delete pod -l app=my-csi-app
kubectl rollout status deployment/my-csi-app
kubectl exec deploy/my-csi-app -- cat /data/hello.txt
```

You should see `Hello from Hetzner Cloud Volumes` again. 🎉 You now have a Deployment writing to a persistent Hetzner Cloud Volume.

---

## 8. Clean up

Delete the workload and the claim. Because the StorageClass uses `reclaimPolicy: Delete`, removing the PVC also deletes the underlying Hetzner Cloud Volume:

```bash
kubectl delete -f pvc-app.yaml
```

Verify the volume is gone:

```bash
hcloud volume list
```

Finally, remove the servers and the SSH key created at the start of the tutorial:

```bash
hcloud server delete tutorial-control-plane tutorial-worker
hcloud ssh-key delete k3s-key
rm -f hcloud-k3s hcloud-k3s.pub kubeconfig
```
