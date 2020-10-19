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
   kubectl apply -f https://raw.githubusercontent.com/hetznercloud/csi-driver/v1.5.0/deploy/kubernetes/hcloud-csi.yml
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
| 1.17-1.19     | 1.5.0 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v1.5.0/deploy/kubernetes/hcloud-csi.yml      |
| 1.16          | 1.4.0 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v1.4.0/deploy/kubernetes/hcloud-csi.yml      |


## E2E Tests

The Hetzner Cloud CSI Driver was tested against the official k8s e2e tests for a specific version. You can run the tests with the following commands. Keep in mind, that these tests run on real cloud servers and will create volumes that will be billed. 

**Test Server Setup:** 
1x CPX21 (Ubuntu 18.04)

**Requirements: Docker and Go 1.15**
1. Configure your environment correctly
```bash
export HCLOUD_TOKEN=<specifiy a project token>
export K8S_VERSION=1.19.2 # The specific (latest) version is needed here
export USE_SSH_KEYS=key1,key2 # Name or IDs of your SSH Keys within the Hetzner Cloud, the servers will be accessable with that keys
```
2. Run the tests
```bash
go test $(go list ./... | grep e2etests) -v -timeout 60m
```
The tests will now run, this will take a while (~30 min).

***If the tests fail, make sure to clean up the project with the Hetzner Cloud Console or the hcloud cli.***
## License

MIT license
