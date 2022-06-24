package e2etests

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"golang.org/x/crypto/ssh"
)

type K8sDistribution string

const (
	K8sDistributionK8s K8sDistribution = "k8s"
	K8sDistributionK3s K8sDistribution = "k3s"
)

var instanceType = "cpx21"

type hcloudK8sSetup struct {
	Hcloud          *hcloud.Client
	HcloudToken     string
	K8sVersion      string
	K8sDistribution K8sDistribution
	TestIdentifier  string
	ImageName       string
	KeepOnFailure   bool
	MainNode        *hcloud.Server
	WorkerNodes     []*hcloud.Server
	privKey         string
	sshKey          *hcloud.SSHKey
	clusterJoinCMD  string
	testLabels      map[string]string
}

type cloudInitTmpl struct {
	K8sVersion      string
	HcloudToken     string
	IsClusterServer bool
	JoinCMD         string
}

// PrepareTestEnv setups a test environment for the CSI Driver
// This includes the creation of a SSH Key, a "Cluster Node" and a defined amount of Worker nodes
// The servers will be created with a Cloud Init UserData
// The template can be found under e2etests/templates/cloudinit_<k8s-distribution>.txt.tpl
func (s *hcloudK8sSetup) PrepareTestEnv(ctx context.Context, additionalSSHKeys []*hcloud.SSHKey) error {
	const op = "hcloudK8sSetup/PrepareTestEnv"

	s.testLabels = map[string]string{"K8sDistribution": string(s.K8sDistribution), "K8sVersion": strings.ReplaceAll(s.K8sVersion, "+", ""), "test": s.TestIdentifier}
	err := s.getSSHKey(ctx)
	if err != nil {
		return fmt.Errorf("%s getSSHKey: %s", op, err)
	}

	srv, err := s.createClusterServer(ctx, "cluster-node", instanceType, additionalSSHKeys)
	if err != nil {
		return fmt.Errorf("%s: create cluster node: %v", op, err)
	}
	s.MainNode = srv

	s.waitUntilSSHable(s.MainNode)

	err = s.waitForCloudInit(s.MainNode)
	if err != nil {
		return err
	}

	joinCmd, err := s.getJoinCmd()
	if err != nil {
		return err
	}
	s.clusterJoinCMD = joinCmd

	err = s.transferDockerImage(s.MainNode)
	if err != nil {
		return fmt.Errorf("%s: %s", op, err)
	}

	fmt.Printf("[cluster-node] %s Load Image:\n", op)
	transferCmd := "ctr -n=k8s.io image import ci-hcloud-csi-driver.tar"
	err = RunCommandOnServer(s.privKey, s.MainNode, transferCmd)
	if err != nil {
		return fmt.Errorf("%s: Load image %s", op, err)
	}

	var workers = 3 // Change this value if you want to have more workers for the test
	var wg sync.WaitGroup
	for worker := 1; worker <= workers; worker++ {
		wg.Add(1)
		go s.createClusterWorker(ctx, additionalSSHKeys, &wg, worker)
	}
	wg.Wait()
	return nil
}

func (s *hcloudK8sSetup) createClusterWorker(ctx context.Context, additionalSSHKeys []*hcloud.SSHKey, wg *sync.WaitGroup, worker int) {
	const op = "hcloudK8sSetup/createClusterWorker"
	defer wg.Done()

	workerName := fmt.Sprintf("cluster-worker-%d", worker)
	fmt.Printf("[%s] %s Create worker node:\n", workerName, op)

	userData, err := s.getCloudInitConfig(false)
	if err != nil {
		fmt.Printf("[%s] %s getCloudInitConfig: %s", workerName, op, err)
		return
	}
	srv, err := s.createServer(ctx, workerName, instanceType, additionalSSHKeys, err, userData)
	if err != nil {
		fmt.Printf("[%s] %s createServer: %s", workerName, op, err)
		return
	}
	s.WorkerNodes = append(s.WorkerNodes, srv)

	s.waitUntilSSHable(srv)

	err = s.waitForCloudInit(srv)
	if err != nil {
		fmt.Printf("[%s] %s: wait for cloud init on worker: %v", srv.Name, op, err)
		return
	}

	err = s.transferDockerImage(srv)
	if err != nil {
		fmt.Printf("[%s] %s: transfer image on worker: %v", srv.Name, op, err)
		return
	}

	fmt.Printf("[%s] %s Load Image\n", srv.Name, op)

	transferCmd := "ctr -n=k8s.io image import ci-hcloud-csi-driver.tar"

	err = RunCommandOnServer(s.privKey, srv, transferCmd)
	if err != nil {
		fmt.Printf("[%s] %s: load image on worker: %v", srv.Name, op, err)
		return
	}
}

func (s *hcloudK8sSetup) waitUntilSSHable(server *hcloud.Server) {
	const op = "hcloudK8sSetup/PrepareTestEnv"
	fmt.Printf("[%s] %s: Waiting for server to be sshable:\n", server.Name, op)
	for {
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:22", server.PublicNet.IPv4.IP.String()))
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		_ = conn.Close()
		fmt.Printf("[%s] %s: SSH Connection successful\n", server.Name, op)
		break
	}
}

func (s *hcloudK8sSetup) createClusterServer(ctx context.Context, name, typ string, additionalSSHKeys []*hcloud.SSHKey) (*hcloud.Server, error) {
	const op = "e2etest/createClusterServer"

	userData, err := s.getCloudInitConfig(true)
	if err != nil {
		return nil, fmt.Errorf("%s getCloudInitConfig: %s", op, err)
	}
	srv, err := s.createServer(ctx, name, typ, additionalSSHKeys, err, userData)
	if err != nil {
		return nil, fmt.Errorf("%s createServer: %s", op, err)
	}
	return srv, nil
}

func (s *hcloudK8sSetup) createServer(ctx context.Context, name string, typ string, additionalSSHKeys []*hcloud.SSHKey, err error, userData string) (*hcloud.Server, error) {
	const op = "e2etest/createServer"
	sshKeys := []*hcloud.SSHKey{s.sshKey}
	for _, additionalSSHKey := range additionalSSHKeys {
		sshKeys = append(sshKeys, additionalSSHKey)
	}

	res, _, err := s.Hcloud.Server.Create(ctx, hcloud.ServerCreateOpts{
		Name:       fmt.Sprintf("%s-%s", name, s.TestIdentifier),
		ServerType: &hcloud.ServerType{Name: typ},
		Image:      &hcloud.Image{Name: "ubuntu-20.04"},
		SSHKeys:    sshKeys,
		UserData:   userData,
		Labels:     s.testLabels,
	})
	if err != nil {
		return nil, fmt.Errorf("%s Hcloud.Server.Create: %s", op, err)
	}

	_, errCh := s.Hcloud.Action.WatchProgress(ctx, res.Action)
	if err := <-errCh; err != nil {
		return nil, fmt.Errorf("%s WatchProgress Action %s: %s", op, res.Action.Command, err)
	}

	for _, nextAction := range res.NextActions {
		_, errCh := s.Hcloud.Action.WatchProgress(ctx, nextAction)
		if err := <-errCh; err != nil {
			return nil, fmt.Errorf("%s WatchProgress NextAction %s: %s", op, nextAction.Command, err)
		}
	}
	srv, _, err := s.Hcloud.Server.GetByID(ctx, res.Server.ID)
	if err != nil {
		return nil, fmt.Errorf("%s Hcloud.Server.GetByID: %s", op, err)
	}
	return srv, nil
}

// PrepareK8s patches an existing kubernetes cluster with the correct
// CSI Driver version from this test run.
// This should only run on the cluster main node
func (s *hcloudK8sSetup) PrepareK8s() (string, error) {
	const op = "hcloudK8sSetup/PrepareK8s"

	err := s.prepareCSIDriverDeploymentFile()
	if err != nil {
		return "", fmt.Errorf("%s: %s", op, err)
	}

	fmt.Printf("[%s] %s: Apply csi-driver deployment\n", s.MainNode.Name, op)
	err = RunCommandOnServer(s.privKey, s.MainNode, "KUBECONFIG=/root/.kube/config kubectl apply -f csi-driver.yml")
	if err != nil {
		return "", fmt.Errorf("%s Deploy csi: %s", op, err)
	}

	patch := `{"spec":{"template":{"spec":{"containers":[{"name":"hcloud-csi-driver","env":[{"name":"LOG_LEVEL","value":"debug"}]}]}}}}`
	fmt.Printf("[%s] %s: Patch deployment for debug logging\n", s.MainNode.Name, op)
	err = RunCommandOnServer(s.privKey, s.MainNode, fmt.Sprintf("KUBECONFIG=/root/.kube/config kubectl patch deployment hcloud-csi-controller -n kube-system --patch '%s'", patch))
	if err != nil {
		return "", fmt.Errorf("%s Patch Deployment: %s", op, err)
	}
	err = RunCommandOnServer(s.privKey, s.MainNode, fmt.Sprintf("KUBECONFIG=/root/.kube/config kubectl patch daemonset hcloud-csi-node -n kube-system --patch '%s'", patch))
	if err != nil {
		return "", fmt.Errorf("%s Patch DaemonSet: %s", op, err)
	}

	fmt.Printf("[%s] %s: Ensure Server is not labeled as master\n", s.MainNode.Name, op)
	err = RunCommandOnServer(s.privKey, s.MainNode, "KUBECONFIG=/root/.kube/config kubectl label nodes --all node-role.kubernetes.io/master-")
	if err != nil {
		return "", fmt.Errorf("%s Ensure Server is not labeled as master: %s", op, err)
	}

	fmt.Printf("[%s] %s: Read test-driver.yml configuration file\n", s.MainNode.Name, op)
	testDriverFile, err := ioutil.ReadFile("templates/testdrivers/1.18.yml")
	if err != nil {
		return "", fmt.Errorf("%s read testdriverfile file: %s %v", op, "templates/testdrivers/1.18.yml", err)
	}

	fmt.Printf("[%s] %s: Transfer test-driver.yml configuration file\n", s.MainNode.Name, op)
	err = RunCommandOnServer(s.privKey, s.MainNode, fmt.Sprintf("echo '%s' >> test-driver.yml", testDriverFile))
	if err != nil {
		return "", fmt.Errorf("%s send testdriverfile file: %s %v", op, "templates/testdrivers/1.18.yml", err)
	}
	fmt.Printf("[%s] %s: Download kubeconfig\n", s.MainNode.Name, op)
	err = scp("ssh_key", fmt.Sprintf("root@%s:/root/.kube/config", s.MainNode.PublicNet.IPv4.IP.String()), "kubeconfig")
	if err != nil {
		return "", fmt.Errorf("%s download kubeconfig: %s", op, err)
	}

	fmt.Printf("[%s] %s: Ensure correct server is set\n", s.MainNode.Name, op)
	kubeconfigBefore, err := ioutil.ReadFile("kubeconfig")
	if err != nil {
		return "", fmt.Errorf("%s reading kubeconfig: %s", op, err)
	}
	kubeconfigAfterwards := strings.Replace(string(kubeconfigBefore), "127.0.0.1", s.MainNode.PublicNet.IPv4.IP.String(), -1)
	err = ioutil.WriteFile("kubeconfig", []byte(kubeconfigAfterwards), 0)
	if err != nil {
		return "", fmt.Errorf("%s writing kubeconfig: %s", op, err)
	}
	return "kubeconfig", nil
}

func scp(identityFile, src, dest string) error {
	const op = "e2etests/scp"

	err := runCmd(
		"/usr/bin/scp",
		[]string{
			"-F", "/dev/null", // ignore $HOME/.ssh/config
			"-i", identityFile,
			"-o", "IdentitiesOnly=yes", // only use the identities passed on the command line
			"-o", "UserKnownHostsFile=/dev/null",
			"-o", "StrictHostKeyChecking=no",
			src,
			dest,
		},
		nil,
	)
	if err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}
	return nil
}

func runCmd(name string, argv []string, env []string) error {
	cmd := exec.Command(name, argv...)
	if os.Getenv("TEST_DEBUG_MODE") != "" {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run cmd: %s %s: %v", name, strings.Join(argv, " "), err)
	}
	return nil
}

// prepareCSIDriverDeploymentFile patches the Cloud Controller Deployment file
// It replaces the used image and the pull policy to always use the local image
// from this test run
func (s *hcloudK8sSetup) prepareCSIDriverDeploymentFile() error {
	const op = "hcloudK8sSetup/prepareCSIDriverDeploymentFile"
	fmt.Printf("[%s] %s: Read master deployment file\n", s.MainNode.Name, op)
	deploymentFile, err := ioutil.ReadFile("../deploy/kubernetes/hcloud-csi.yml")
	if err != nil {
		return fmt.Errorf("%s: read csi driver deployment file %s: %v", op, "../deploy/kubernetes/hcloud-csi.yml", err)
	}

	fmt.Printf("[%s] %s: Prepare deployment file and transfer it\n", s.MainNode.Name, op)
	deploymentFile = []byte(strings.ReplaceAll(string(deploymentFile), "hetznercloud/hcloud-csi-driver:latest", fmt.Sprintf("hcloud-csi:ci_%s", s.TestIdentifier)))
	deploymentFile = []byte(strings.ReplaceAll(string(deploymentFile), " imagePullPolicy: Always", " imagePullPolicy: IfNotPresent"))

	err = RunCommandOnServer(s.privKey, s.MainNode, fmt.Sprintf("echo '%s' >> csi-driver.yml", deploymentFile))
	if err != nil {
		return fmt.Errorf("%s: Prepare deployment file and transfer it: %s", op, err)
	}
	return nil
}

// transferDockerImage transfers the local build docker image tar via SCP
func (s *hcloudK8sSetup) transferDockerImage(server *hcloud.Server) error {
	const op = "hcloudK8sSetup/transferDockerImage"
	fmt.Printf("[%s] %s: Transfer docker image\n", server.Name, op)
	err := WithSSHSession(s.privKey, server.PublicNet.IPv4.IP.String(), func(session *ssh.Session) error {
		file, err := os.Open("ci-hcloud-csi-driver.tar")
		if err != nil {
			return fmt.Errorf("%s read ci-hcloud-ccm.tar: %s", op, err)
		}
		defer file.Close()
		stat, err := file.Stat()
		if err != nil {
			return fmt.Errorf("%s file.Stat: %s", op, err)
		}
		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			hostIn, _ := session.StdinPipe()
			defer hostIn.Close()
			fmt.Fprintf(hostIn, "C0664 %d %s\n", stat.Size(), "ci-hcloud-csi-driver.tar")
			io.Copy(hostIn, file)
			fmt.Fprint(hostIn, "\x00")
			wg.Done()
		}()

		err = session.Run("/usr/bin/scp -t /root")
		if err != nil {
			return fmt.Errorf("%s copy via scp: %s", op, err)
		}
		wg.Wait()
		return err
	})
	return err
}

// waitForCloudInit waits on cloud init on the server.
// when cloud init is ready we can assume that the server
// and the plain k8s installation is ready
func (s *hcloudK8sSetup) waitForCloudInit(server *hcloud.Server) error {
	const op = "hcloudK8sSetup/PrepareTestEnv"
	fmt.Printf("[%s] %s: Wait for cloud-init\n", server.Name, op)
	err := RunCommandOnServer(s.privKey, server, fmt.Sprintf("cloud-init status --wait > /dev/null"))
	if err != nil {
		return fmt.Errorf("[%s] %s: Wait for cloud-init: %s", server.Name, op, err)
	}
	return nil
}

// waitForCloudInit waits on cloud init on the server.
// when cloud init is ready we can assume that the server
// and the plain k8s installation is ready
func (s *hcloudK8sSetup) getJoinCmd() (string, error) {
	const op = "hcloudK8sSetup/getJoinCmd"
	fmt.Printf("[%s] %s: Download join cmd\n", s.MainNode.Name, op)
	err := scp("ssh_key", fmt.Sprintf("root@%s:/root/join.txt", s.MainNode.PublicNet.IPv4.IP.String()), "join.txt")
	if err != nil {
		return "", fmt.Errorf("[%s] %s download join cmd: %s", s.MainNode.Name, op, err)
	}
	cmd, err := ioutil.ReadFile("join.txt")
	if err != nil {
		return "", fmt.Errorf("[%s] %s reading join cmd file: %s", s.MainNode.Name, op, err)
	}
	return string(cmd), nil
}

// TearDown deletes all created resources within the Hetzner Cloud
// there is no need to "shutdown" the k8s cluster before
// so we just delete all created resources
func (s *hcloudK8sSetup) TearDown(testFailed bool) error {
	const op = "hcloudK8sSetup/TearDown"

	if s.KeepOnFailure && testFailed {
		fmt.Println("Skipping tear-down for further analysis.")
		fmt.Println("Please clean-up afterwards ;-)")
		return nil
	}

	ctx := context.Background()
	for _, wn := range s.WorkerNodes {
		_, err := s.Hcloud.Server.Delete(ctx, wn)
		if err != nil {
			return fmt.Errorf("[%s] %s Hcloud.Server.Delete: %s", wn.Name, op, err)
		}
	}
	_, err := s.Hcloud.Server.Delete(ctx, s.MainNode)
	if err != nil {
		return fmt.Errorf("[cluster-node] %s Hcloud.Server.Delete: %s", op, err)
	}
	s.MainNode = nil

	_, err = s.Hcloud.SSHKey.Delete(ctx, s.sshKey)
	if err != nil {
		return fmt.Errorf("%s Hcloud.SSHKey.Delete: %s", err, err)
	}
	s.sshKey = nil
	return nil
}

// getCloudInitConfig returns the generated cloud init configuration
func (s *hcloudK8sSetup) getCloudInitConfig(isClusterServer bool) (string, error) {
	const op = "hcloudK8sSetup/getCloudInitConfig"

	str, err := ioutil.ReadFile(fmt.Sprintf("templates/cloudinit_%s.txt.tpl", s.K8sDistribution))
	if err != nil {
		return "", fmt.Errorf("%s: read template file %s: %v", "templates/cloudinit.txt.tpl", op, err)
	}
	tmpl, err := template.New("cloud_init").Parse(string(str))
	if err != nil {
		return "", fmt.Errorf("%s: parsing template file %s: %v", "templates/cloudinit.txt.tpl", op, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cloudInitTmpl{K8sVersion: s.K8sVersion, HcloudToken: s.HcloudToken, IsClusterServer: isClusterServer, JoinCMD: s.clusterJoinCMD}); err != nil {
		return "", fmt.Errorf("%s: execute template: %v", op, err)
	}
	return buf.String(), nil
}

//getSSHKey create and get the Hetzner Cloud SSH Key for the test
func (s *hcloudK8sSetup) getSSHKey(ctx context.Context) error {
	const op = "hcloudK8sSetup/getSSHKey"
	pubKey, privKey, err := makeSSHKeyPair()
	if err != nil {
		return err
	}

	sshKey, _, err := s.Hcloud.SSHKey.Create(ctx, hcloud.SSHKeyCreateOpts{
		Name:      fmt.Sprintf("s-%s", s.TestIdentifier),
		PublicKey: pubKey,
		Labels:    s.testLabels,
	})
	if err != nil {
		return fmt.Errorf("%s: creating ssh key: %v", op, err)
	}
	s.privKey = privKey
	s.sshKey = sshKey
	err = ioutil.WriteFile("ssh_key", []byte(s.privKey), 0600)
	if err != nil {
		return fmt.Errorf("%s: writing ssh key private key: %v", op, err)
	}
	return nil
}

// makeSSHKeyPair generate a SSH key pair
func makeSSHKeyPair() (string, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return "", "", err
	}

	// generate and write private key as PEM
	var privKeyBuf strings.Builder

	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	if err := pem.Encode(&privKeyBuf, privateKeyPEM); err != nil {
		return "", "", err
	}

	// generate and write public key
	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", err
	}

	var pubKeyBuf strings.Builder
	pubKeyBuf.Write(ssh.MarshalAuthorizedKey(pub))

	return pubKeyBuf.String(), privKeyBuf.String(), nil
}
func RunCommandOnServer(privKey string, server *hcloud.Server, command string) error {
	return WithSSHSession(privKey, server.PublicNet.IPv4.IP.String(), func(session *ssh.Session) error {
		if ok := os.Getenv("TEST_DEBUG_MODE"); ok != "" {
			session.Stdout = os.Stdout
		}
		return session.Run(command)
	})
}
func RunCommandVisibleOnServer(privKey string, server *hcloud.Server, command string) error {
	return WithSSHSession(privKey, server.PublicNet.IPv4.IP.String(), func(session *ssh.Session) error {
		session.Stdout = os.Stdout
		return session.Run(command)
	})
}

func WithSSHSession(privKey string, host string, fn func(*ssh.Session) error) error {
	signer, err := ssh.ParsePrivateKey([]byte(privKey))
	if err != nil {
		return err
	}

	client, err := ssh.Dial("tcp", net.JoinHostPort(host, "22"), &ssh.ClientConfig{
		User:            "root",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         1 * time.Second,
	})
	if err != nil {
		return err
	}

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	return fn(session)
}
