# Container Storage Interface driver for Hetzner Cloud

[![GitHub Actions status](https://github.com/hetznercloud/csi-driver/workflows/Run%20tests/badge.svg)](https://github.com/hetznercloud/csi-driver/actions)

This is a [Container Storage Interface](https://github.com/container-storage-interface/spec) driver for Hetzner Cloud
enabling you to use Volumes within Kubernetes. Please note that this driver **requires Kubernetes 1.13 or newer**.

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

    Have a look at our [Version Matrix](README.md#version-matrix) to pick the correct deployment file.
   ```
   kubectl apply -f https://raw.githubusercontent.com/hetznercloud/csi-driver/v1.3.2/deploy/kubernetes/hcloud-csi.yml
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

## Version Matrix

| Kubernetes    | CSI Driver   | Deployment File |
| ------------- | -----:| ------------------------------------------------------------------------------------------------------:|
| 1.16-1.18     | 1.3.2 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v1.3.2/deploy/kubernetes/hcloud-csi.yml      |
| 1.14-1.15     | 1.1.5 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v1.1.5/deploy/kubernetes/hcloud-csi.yml      |
| 1.13          | 1.1.5 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v1.1.5/deploy/kubernetes/hcloud-csi-1.13.yml |

## License

MIT license
