package e2etests

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/hetznercloud/csi-driver/integrationtests"
	"github.com/hetznercloud/hcloud-go/hcloud"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

var rng *rand.Rand

func init() {
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
}

type TestCluster struct {
	KeepOnFailure bool
	setup         *hcloudK8sSetup
	k8sClient     *kubernetes.Clientset
	started       bool

	mu sync.Mutex
}

func (tc *TestCluster) initialize() error {
	const op = "e2tests/TestCluster.initialize"

	if tc.started {
		return nil
	}

	fmt.Printf("%s: Starting Testsuite\n", op)

	isUsingGithubActions := os.Getenv("GITHUB_ACTIONS")
	isUsingGitlabCI := os.Getenv("CI_JOB_ID")
	testIdentifier := ""
	if isUsingGithubActions == "true" {
		testIdentifier = fmt.Sprintf("gh-%s-%d", os.Getenv("GITHUB_RUN_ID"), rng.Int())
		fmt.Printf("%s: Running in Github Action\n", op)
	}
	if isUsingGitlabCI != "" {
		testIdentifier = fmt.Sprintf("gl-%s", isUsingGitlabCI)
		fmt.Printf("%s: Running in Gitlab CI\n", op)
	}
	if testIdentifier == "" {
		testIdentifier = fmt.Sprintf("local-%d", rng.Int())
		fmt.Printf("%s: Running local\n", op)
	}

	k8sVersion := os.Getenv("K8S_VERSION")
	if k8sVersion == "" {
		k8sVersion = "k8s-1.18.9"
	}

	k8sVersionsDetails := strings.Split(k8sVersion, "-")
	if len(k8sVersionsDetails) != 2 {
		return fmt.Errorf("%s: invalid k8s version: %v should be format <distribution>-<version>", op, k8sVersion)
	}

	token := os.Getenv("HCLOUD_TOKEN")
	if len(token) != 64 {
		return fmt.Errorf("%s: No valid HCLOUD_TOKEN found", op)
	}
	tc.KeepOnFailure = os.Getenv("KEEP_SERVER_ON_FAILURE") == "yes"

	var additionalSSHKeys []*hcloud.SSHKey

	opts := []hcloud.ClientOption{
		hcloud.WithToken(token),
		hcloud.WithApplication("hcloud-ccm-testsuite", "1.0"),
	}
	hcloudClient := hcloud.NewClient(opts...)
	additionalSSHKeysIDOrName := os.Getenv("USE_SSH_KEYS")
	if additionalSSHKeysIDOrName != "" {
		idsOrNames := strings.Split(additionalSSHKeysIDOrName, ",")
		for _, idOrName := range idsOrNames {
			additionalSSHKey, _, err := hcloudClient.SSHKey.Get(context.Background(), idOrName)
			if err != nil {
				return fmt.Errorf("%s: %s", op, err)
			}
			additionalSSHKeys = append(additionalSSHKeys, additionalSSHKey)
		}
	}

	fmt.Printf("%s: Test against %s\n", op, k8sVersion)

	imageName := os.Getenv("CSI_IMAGE_NAME")
	buildImage := false
	if imageName == "" {
		imageName = fmt.Sprintf("hcloud-csi:ci_%s", testIdentifier)
		buildImage = true
	}
	if buildImage {
		fmt.Printf("%s: Building image\n", op)
		if _, err := integrationtests.DockerBuild(imageName, "../"); err != nil {
			return fmt.Errorf("%s: %v", op, err)
		}
	}

	fmt.Printf("%s: Saving image to disk\n", op)
	if _, err := integrationtests.DockerSave(imageName, "ci-hcloud-csi-driver.tar"); err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}

	tc.setup = &hcloudK8sSetup{
		Hcloud:          hcloudClient,
		K8sDistribution: K8sDistribution(k8sVersionsDetails[0]),
		K8sVersion:      k8sVersionsDetails[1],
		TestIdentifier:  testIdentifier,
		ImageName:       imageName,
		HcloudToken:     token,
		KeepOnFailure:   tc.KeepOnFailure,
	}
	fmt.Printf("%s: Setting up test env\n", op)

	err := tc.setup.PrepareTestEnv(context.Background(), additionalSSHKeys)
	if err != nil {
		return fmt.Errorf("%s: %s", op, err)
	}

	kubeconfigPath, err := tc.setup.PrepareK8s()
	if err != nil {
		return fmt.Errorf("%s: %s", op, err)
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return fmt.Errorf("%s: clientcmd.BuildConfigFromFlags: %s", op, err)
	}

	tc.k8sClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("%s: kubernetes.NewForConfig: %s", op, err)
	}

	tc.started = true
	return nil
}

func (tc *TestCluster) Start() error {
	const op = "e2etests/TestCluster.Start"

	tc.mu.Lock()
	defer tc.mu.Unlock()

	if err := tc.initialize(); err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}
	if err := tc.ensureNodesReady(); err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}
	if err := tc.ensurePodsReady(); err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}
	return nil
}

func (tc *TestCluster) Stop(testFailed bool) error {
	const op = "e2etests/TestCluster.Stop"

	tc.mu.Lock()
	defer tc.mu.Unlock()

	if !tc.started {
		return nil
	}

	if err := tc.setup.TearDown(testFailed); err != nil {
		fmt.Printf("%s: Tear Down: %s", op, err)
	}
	return nil
}

func (tc *TestCluster) ensureNodesReady() error {
	const op = "e2etests/ensureNodesReady"

	err := wait.Poll(1*time.Second, 5*time.Minute, func() (bool, error) {
		var totalNodes = len(tc.setup.WorkerNodes) + 1 // Number Worker Nodes + 1 Cluster Node
		var readyNodes int
		nodes, err := tc.k8sClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		for _, node := range nodes.Items {
			for _, cond := range node.Status.Conditions {
				if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
					readyNodes++
				}
			}
		}
		pendingNodes := totalNodes - readyNodes
		fmt.Printf("Waiting for %d/%d nodes\n", pendingNodes, totalNodes)
		return pendingNodes == 0, err
	})

	if err != nil {
		return fmt.Errorf("%s: %s", op, err)
	}
	return nil
}

func (tc *TestCluster) ensurePodsReady() error {
	const op = "e2etests/ensurePodsReady"

	err := wait.Poll(1*time.Second, 10*time.Minute, func() (bool, error) {
		pods, err := tc.k8sClient.CoreV1().Pods("kube-system").List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		totalPods := len(pods.Items)

		var readyPods int
		for _, pod := range pods.Items {
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					readyPods++
				}
			}
		}

		pendingPods := totalPods - readyPods
		fmt.Printf("Waiting for %d/%d pods\n", pendingPods, totalPods)
		return pendingPods == 0, err
	})

	if err != nil {
		return fmt.Errorf("%s: %s", op, err)
	}
	return nil
}
