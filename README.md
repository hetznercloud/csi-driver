# Container Storage Interface driver for Hetzner Cloud

**IMPORTANT: This project is still in development and not ready production yet!**

## Getting Started

1. Create service accounts, cluster roles, and bindings:

       kubectl create -f deploy/kubernetes/attacher-rbac.yml
       kubectl create -f deploy/kubernetes/node-rbac.yml
       kubectl create -f deploy/kubernetes/provisioner-rbac.yml

2. Add your API token to `secret.yml` and create it:

       kubectl create -f deploy/kubernetes/secret.yml

3. Create the attacher, provisioner, and node registrar:

       kubectl create -f deploy/kubernetes/attacher.yml
       kubectl create -f deploy/kubernetes/node.yml
       kubectl create -f deploy/kubernetes/provisioner.yml

4. Create the storage class:

       kubectl create -f deploy/kubernetes/storageclass.yml

5. Make sure all pods are running and ready:

       kubectl get pods

6. To test everything is working, create a persistent volume claim:

       kubectl create -f - <<EOF
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
       EOF

7. Create a pod which uses that volume:

       kubectl create -f - <<EOF
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
       EOF

8. Once the pod is ready, exec a shell and check your volume is mounted at `/data`.

## License

MIT license
