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
	t.Run("parallel tests", func(t *testing.T) {
		err := RunCommandVisibleOnServer(testCluster.setup.privKey, testCluster.setup.MainNode, fmt.Sprintf("KUBECONFIG=/root/.kube/config ./ginkgo -nodes=6 -v -focus='External.Storage' -skip='\\[Feature:|\\[Disruptive\\]|\\[Serial\\]' ./e2e.test -- -storage.testdriver=test-driver.yml"))
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("serial tests", func(t *testing.T) {
		err := RunCommandVisibleOnServer(testCluster.setup.privKey, testCluster.setup.MainNode, fmt.Sprintf("KUBECONFIG=/root/.kube/config ./ginkgo -v -focus='External.Storage.*(\\[Feature:|\\[Serial\\])' ./e2e.test -- -storage.testdriver=test-driver.yml"))
		if err != nil {
			t.Error(err)
		}
	})
}
