package e2etests

import (
	"fmt"
	"os"
	"testing"
)

var testCluster TestCluster

func TestMain(m *testing.M) {
	if err := testCluster.Start(); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	rc := m.Run()

	if err := testCluster.Stop(rc > 0); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	os.Exit(rc)
}

func TestOfficialTestsuite(t *testing.T) {
	// The e2e tests are a bit flaky, and at the moment in ~1/3 of the runs a test fails, causing the whole pipeline to
	// fail. As ,the e2e tests take 15-20 minutes each, this is quite annoying. By setting -flakeAttempts=2, the pipeline
	// will immediately retry any failed tests.
	t.Run("parallel tests", func(t *testing.T) {
		err := RunCommandVisibleOnServer(testCluster.setup.privKey, testCluster.setup.MainNode, "KUBECONFIG=/root/.kube/config ./ginkgo -nodes=6 -flakeAttempts=2 -v -focus='External.Storage' -skip='\\[Feature:|\\[Disruptive\\]|\\[Serial\\]' ./e2e.test -- -storage.testdriver=test-driver.yml")
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("serial tests", func(t *testing.T) {
		// Tests tagged as "Feature:SELinuxMountReadWriteOncePod" were added in
		// Kubernetes v1.26, and fail for us because we do not support the
		// SINGLE_NODE_MULTI_WRITER Capability (equivalent to ReadWriteOncePod
		// Volume Access Mode in Kubernetes).
		// This feature is being tracked in https://github.com/hetznercloud/csi-driver/issues/327
		// and we should add the tests once we have implemented the capability.
		err := RunCommandVisibleOnServer(testCluster.setup.privKey, testCluster.setup.MainNode, "KUBECONFIG=/root/.kube/config ./ginkgo -flakeAttempts=2 -v -focus='External.Storage.*(\\[Feature:|\\[Serial\\])' -skip='\\[Feature:SELinuxMountReadWriteOncePod\\]' ./e2e.test -- -storage.testdriver=test-driver.yml")
		if err != nil {
			t.Error(err)
		}
	})
}
