package e2etests

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"
)

var rng *rand.Rand

func init() {
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
}
func TestCSIDriver(t *testing.T) {
	fmt.Println("Starting CSI Driver Testsuite")
	isUsingGithubActions := os.Getenv("GITHUB_ACTIONS")
	isUsingGitlabCI := os.Getenv("CI_JOB_ID")
	testIdentifier := ""
	if isUsingGithubActions == "true" {
		testIdentifier = fmt.Sprintf("gh-%s-%d", os.Getenv("GITHUB_RUN_ID"), rng.Int())
		fmt.Println("Running in Github Action")
	}
	if isUsingGitlabCI != "" {
		testIdentifier = fmt.Sprintf("gl-%s", isUsingGitlabCI)
		fmt.Println("Running in Gitlab CI")
	}
	if testIdentifier == "" {
		testIdentifier = fmt.Sprintf("local-%d", rand.Int())
		fmt.Println("Running locally")
	}

	k8sVersion := os.Getenv("K8S_VERSION")
	if k8sVersion == "" {
		k8sVersion = "1.18.9"
	}
	token := os.Getenv("HCLOUD_TOKEN")
	if len(token) != 64 {
		t.Fatalf("No valid HCLOUD_TOKEN found")
	}

	var additionalSSHKeys []*hcloud.SSHKey

	opts := []hcloud.ClientOption{
		hcloud.WithToken(token),
		hcloud.WithApplication("hcloud-csi-driver-testsuite", "1.0"),
	}
	hcloudClient := hcloud.NewClient(opts...)
	additionalSSHKeysIdOrName := os.Getenv("USE_SSH_KEYS")
	if additionalSSHKeysIdOrName != "" {
		idsOrNames := strings.Split(additionalSSHKeysIdOrName, ",")
		for _, idOrName := range idsOrNames {
			additionalSSHKey, _, err := hcloudClient.SSHKey.Get(context.Background(), idOrName)
			if err != nil {
				t.Fatal(err)
			}
			additionalSSHKeys = append(additionalSSHKeys, additionalSSHKey)
		}

	}

	fmt.Printf("Test against k8s %s\n", k8sVersion)

	fmt.Println("Building csi driver image")
	cmd := exec.Command("docker", "build", "-t", fmt.Sprintf("hcloud-csi-driver:ci_%s", testIdentifier), "../")
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("Saving csi driver image to disk")
	cmd = exec.Command("docker", "save", "--output", "ci-hcloud-csi-driver.tar", fmt.Sprintf("hcloud-csi-driver:ci_%s", testIdentifier))
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		t.Fatal(err)
	}

	setup := hcloudK8sSetup{Hcloud: hcloudClient, K8sVersion: k8sVersion, TestIdentifier: testIdentifier, HcloudToken: token}
	fmt.Println("Setting up test env")

	err = setup.PrepareTestEnv(context.Background(), additionalSSHKeys)
	if err != nil {
		t.Fatalf("%s", err)
	}

	fmt.Println("Run tests")
	err = setup.RunE2ETests()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("Tear Down")
	err = setup.TearDown(context.Background())
	if err != nil {
		t.Fatal(err)
	}

}
