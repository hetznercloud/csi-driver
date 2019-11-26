module hetzner.cloud/csi

go 1.13

require (
	github.com/container-storage-interface/spec v1.2.0
	github.com/go-kit/kit v0.8.0
	github.com/golang/protobuf v1.3.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/hetznercloud/hcloud-go v1.17.0
	github.com/kubernetes-csi/csi-test v2.4.0-rc1+incompatible
	github.com/prometheus/client_golang v1.1.0
	github.com/spf13/afero v1.2.0 // indirect
	golang.org/x/sys v0.0.0-20190801041406-cbf593c0f2f3
	google.golang.org/grpc v1.19.0
	k8s.io/apimachinery v0.0.0-20181215012845-4d029f033399 // indirect
	k8s.io/klog v0.1.0 // indirect
	k8s.io/kubernetes v1.14.0
	k8s.io/utils v0.0.0-20190221042446-c2654d5206da // indirect
)
