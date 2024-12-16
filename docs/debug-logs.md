# Debug Logs

Debug logs can help with debugging issues in the CSI Driver. By default no debug logs are shown, they need to be activated. You can active the debug logs by adding a new environment variable to both csi-driver workloads.

The new variables as a patch:

```diff
diff --git a/deploy/kubernetes/controller/deployment.yaml b/deploy/kubernetes/controller/deployment.yaml
index 1c051dc..a64ea15 100644
--- a/deploy/kubernetes/controller/deployment.yaml
+++ b/deploy/kubernetes/controller/deployment.yaml
@@ -38,6 +38,10 @@ spec:
         imagePullPolicy: Always
         command: [/bin/hcloud-csi-driver-controller]
         env:
+        - name: LOG_LEVEL
+          value: debug
+        - name: HCLOUD_DEBUG
+          value: "true"
         - name: CSI_ENDPOINT
           value: unix:///run/csi/socket
         - name: METRICS_ENDPOINT
diff --git a/deploy/kubernetes/node/daemonset.yaml b/deploy/kubernetes/node/daemonset.yaml
index c8f8b57..335d451 100644
--- a/deploy/kubernetes/node/daemonset.yaml
+++ b/deploy/kubernetes/node/daemonset.yaml
@@ -45,6 +45,10 @@ spec:
         imagePullPolicy: Always
         command: [/bin/hcloud-csi-driver-node]
         env:
+        - name: LOG_LEVEL
+          value: debug
+        - name: HCLOUD_DEBUG
+          value: "true"
         - name: CSI_ENDPOINT
           value: unix:///run/csi/socket
         - name: METRICS_ENDPOINT
```

The new variables as a [strategic merge patch](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/strategic-merge-patch.md):

```yaml
kind: Deployment
apiVersion: apps/v1
metadata:
  name: hcloud-csi-controller
  namespace: kube-system
spec:
  template:
    spec:
      containers:
        - name: hcloud-csi-driver
          env:
            - name: LOG_LEVEL
              value: debug
            - name: HCLOUD_DEBUG
              value: "true"

---
kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: hcloud-csi-node
  namespace: kube-system
spec:
  template:
    spec:
      containers:
        - name: hcloud-csi-driver
          env:
            - name: LOG_LEVEL
              value: debug
            - name: HCLOUD_DEBUG
              value: "true"
```

Once the new pods with the environment variable have started, you should see messages like this in the logs. If you do not, check that you set the environment variables in the right location:

```
level=info ts=2022-12-15T08:59:08.512440485Z component=idempotent-volume-service msg="volume created" volume-id=12345678
level=info ts=2022-12-15T08:59:08.51246971Z component=driver-controller-service msg="created volume" volume-id=12345678 volume-name=pvc-9a56bc78-b626-47fd-8220-2337fb672350
level=debug ts=2022-12-15T08:59:08.512491741Z component=grpc-server msg="finished handling request" method=/csi.v1.Controller/CreateVolume err=null
level=debug ts=2022-12-15T08:59:09.263921267Z component=grpc-server msg="handling request" method=/csi.v1.Controller/ControllerPublishVolume req="volume_id:\"12345678\" node_id:\"98765432\" volume_capability:<mount:<fs_type:\"ext4\" > access_mode:<mode:SINGLE_NODE_WRITER > > volume_context:<key:\"storage.kubernetes.io/csiProvisionerIdentity\" value:\"1671094571396-8081-csi.hetzner.cloud\" > "
level=info ts=2022-12-15T08:59:09.264013881Z component=api-volume-service msg="attaching volume" volume-id=12345678 server-id=98765432
--- Request:
GET /v1/volumes/12345678 HTTP/1.1
Host: api.hetzner.cloud
User-Agent: csi-driver/2.1.0 hcloud-go/1.37.0
Authorization: REDACTED
Accept-Encoding: gzip



--- Response:
HTTP/2.0 200 OK
Content-Length: 603
Access-Control-Allow-Credentials: true
Access-Control-Allow-Origin: *
Content-Type: application/json
Date: Thu, 15 Dec 2022 08:59:09 GMT
Ratelimit-Limit: 5000
Ratelimit-Remaining: 4999
Ratelimit-Reset: 1671094749
Strict-Transport-Security: max-age=15724800; includeSubDomains
X-Correlation-Id: 9e887c60182c28ad

{
  "volume": {
    "id": 12345678,
    "created": "2022-12-15T08:59:02+00:00",
    "name": "pvc-9a56bc78-b626-47fd-8220-2337fb672350",
    "server": null,
    "location": {
      "id": 3,
      "name": "hel1",
      "description": "Helsinki DC Park 1",
      "country": "FI",
      "city": "Helsinki",
      "latitude": 60.169855,
      "longitude": 24.938379,
      "network_zone": "eu-central"
    },
    "size": 10,
    "linux_device": "/dev/disk/by-id/scsi-0HC_Volume_12345678",
    "protection": {
      "delete": false
    },
    "labels": {},
    "status": "available",
    "format": null
  }
}
```
