module github.com/hetznercloud/csi-driver

go 1.19

require (
	github.com/container-storage-interface/spec v1.7.0
	github.com/go-kit/kit v0.12.0
	github.com/golang/protobuf v1.5.3
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/hetznercloud/hcloud-go v1.40.0
	github.com/kubernetes-csi/csi-test/v4 v4.4.0
	github.com/prometheus/client_golang v1.14.0
	golang.org/x/sys v0.6.0
	google.golang.org/grpc v1.53.0
	k8s.io/mount-utils v0.27.3
	k8s.io/utils v0.0.0-20230505201702-9f6742963106
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/go-kit/log v0.2.0 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/pprof v0.0.0-20210720184732-4bb14d4b1be1 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/moby/sys/mountinfo v0.6.2 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/onsi/gomega v1.27.4 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/text v0.8.0 // indirect
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/klog/v2 v2.90.1 // indirect
)

replace k8s.io/api => k8s.io/api v0.27.3

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.27.3

replace k8s.io/apimachinery => k8s.io/apimachinery v0.27.3

replace k8s.io/apiserver => k8s.io/apiserver v0.27.3

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.27.3

replace k8s.io/client-go => k8s.io/client-go v0.27.3

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.27.3

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.27.3

replace k8s.io/code-generator => k8s.io/code-generator v0.27.3

replace k8s.io/component-base => k8s.io/component-base v0.27.3

replace k8s.io/component-helpers => k8s.io/component-helpers v0.24.0

replace k8s.io/controller-manager => k8s.io/controller-manager v0.27.3

replace k8s.io/cri-api => k8s.io/cri-api v0.27.3

replace k8s.io/mount-utils => k8s.io/mount-utils v0.27.3

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.27.3

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.27.3

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.27.3

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.27.3

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.27.3

replace k8s.io/kubelet => k8s.io/kubelet v0.27.3

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.27.3

replace k8s.io/metrics => k8s.io/metrics v0.27.3

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.27.3

replace k8s.io/kubectl => k8s.io/kubectl v0.27.3
