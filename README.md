# Container Storage Interface driver for Hetzner Cloud

[![Build Status](https://travis-ci.com/hetznercloud/csi-driver.svg?branch=master)](https://travis-ci.com/hetznercloud/csi-driver)

This is a [Container Storage Interface](https://github.com/container-storage-interface/spec) driver for Hetzner Cloud
enabling you to use Volumes within Kubernetes. Please note that this driver **requires Kubernetes 1.13 or newer**.

## Getting Started

1. Make sure that the following feature gates are enabled in your cluster (kubelet and kube-apiserver):

   ```
   --feature-gates=CSINodeInfo=true,CSIDriverRegistry=true
   ```

   Also make sure that privileged pods are allowed in your cluster (kubelet and kube-apiserver):

   ```
   --allow-privileged=true
   ```

   See [Enabling features](https://kubernetes-csi.github.io/docs/Setup.html#enabling-features) and
   [Enable privileged Pods](https://kubernetes-csi.github.io/docs/Setup.html#enable-privileged-pods)
   in the Kubernetes CSI documentation.

2. Create the `CSINodeInfo` resource:

   ```
   kubectl create -f https://raw.githubusercontent.com/kubernetes/csi-api/master/pkg/crd/manifests/csinodeinfo.yaml
   ```

   See [Enabling CSINodeInfo](https://kubernetes-csi.github.io/docs/Setup.html#enabling-csinodeinfo)
   in the Kubernetes CSI documentation.

3. Create an API token in the [Hetzner Cloud Console](https://console.hetzner.cloud/).

4. Create a secret containing the token:

   ```
   apiVersion: v1
   kind: Secret
   metadata:
     name: hcloud-csi
     namespace: kube-system
   stringData:
     token: YOURTOKEN
   ```

5. Deploy the CSI driver and wait until everything is up and running:

   ```
   kubectl apply -f https://raw.githubusercontent.com/hetznercloud/csi-driver/master/deploy/kubernetes/hcloud-csi.yml
   ```

6. To verify everything is working, create a persistent volume claim and a pod
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

## License

MIT license
