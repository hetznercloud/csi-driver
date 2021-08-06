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
	"strings"
	"sync"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"
	"golang.org/x/crypto/ssh"
)

type hcloudK8sSetup struct {
	Hcloud         *hcloud.Client
	HcloudToken    string
	K8sVersion     string
	TestIdentifier string
	privKey        string
	server         *hcloud.Server
	sshKey         *hcloud.SSHKey
}

type cloudInitTmpl struct {
	K8sVersion  string
	HcloudToken string
}

func (s *hcloudK8sSetup) PrepareTestEnv(ctx context.Context, additionalSSHKeys []*hcloud.SSHKey) error {
	const op = "hcloudK8sSetup/PrepareTestEnv"
	userData, err := s.getCloudInitConfig()
	if err != nil {
		return fmt.Errorf("%s getCloudInitConfig: %s", op, err)
	}

	err = s.getSSHKey(ctx)
	if err != nil {
		return fmt.Errorf("%s getSSHKey: %s", op, err)
	}

	sshKeys := []*hcloud.SSHKey{s.sshKey}
	for _, additionalSSHKey := range additionalSSHKeys {
		sshKeys = append(sshKeys, additionalSSHKey)
	}

	res, _, err := s.Hcloud.Server.Create(ctx, hcloud.ServerCreateOpts{
		Name:       fmt.Sprintf("srv-%s", s.TestIdentifier),
		ServerType: &hcloud.ServerType{Name: "cpx21"},
		Image:      &hcloud.Image{Name: "ubuntu-20.04"},
		SSHKeys:    sshKeys,
		UserData:   userData,
		Labels:     map[string]string{"K8sVersion": s.K8sVersion, "test": s.TestIdentifier},
	})
	if err != nil {
		return fmt.Errorf("%s Hcloud.Server.Create: %s", op, err)
	}

	_, errCh := s.Hcloud.Action.WatchProgress(ctx, res.Action)
	if err := <-errCh; err != nil {
		return fmt.Errorf("%s WatchProgress Action %s: %s", op, res.Action.Command, err)
	}

	for _, nextAction := range res.NextActions {
		_, errCh := s.Hcloud.Action.WatchProgress(ctx, nextAction)
		if err := <-errCh; err != nil {
			return fmt.Errorf("%s WatchProgress NextAction %s: %s", op, nextAction.Command, err)
		}
	}
	s.server = res.Server
	fmt.Printf("%s Waiting for server to be sshable:", op)
	for {
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:22", s.server.PublicNet.IPv4.IP.String()))
		if err != nil {
			fmt.Print(".")
			time.Sleep(1 * time.Second)
			continue
		}
		_ = conn.Close()
		fmt.Print("Connection successful\n")
		break
	}
	err = s.waitForCloudInit()
	if err != nil {
		return err
	}

	err = s.transferCSIDockerImage()
	if err != nil {
		return fmt.Errorf("%s: %s", op, err)
	}

	fmt.Printf("%s Load Image:\n", op)
	err = RunCommandOnServer(s.privKey, s.server, fmt.Sprintf("docker load --input ci-hcloud-csi-driver.tar"))
	if err != nil {
		return fmt.Errorf("%s:  Load image%s", op, err)
	}

	return nil
}
func (s *hcloudK8sSetup) RunE2ETests() error {
	const op = "hcloudK8sSetup/RunE2ETests"
	err := s.prepareCSIDriverDeploymentFile()
	if err != nil {
		return fmt.Errorf("%s: %s", op, err)
	}

	fmt.Printf("%s: Apply csi driver deployment\n", op)
	err = RunCommandOnServer(s.privKey, s.server, fmt.Sprintf("KUBECONFIG=/root/.kube/config kubectl apply -f csi-driver.yml"))
	if err != nil {
		return fmt.Errorf("%s Deploy CSI Driver: %s", op, err)
	}

	fmt.Printf("%s: Read test-driver.yml configuration file\n", op)
	testDriverFile, err := ioutil.ReadFile("templates/testdrivers/1.18.yml")
	if err != nil {
		return fmt.Errorf("%s read testdriverfile file: %s %v", op, "templates/testdrivers/1.18.yml", err)
	}

	fmt.Printf("%s: Transfer test-driver.yml configuration file\n", op)
	err = RunCommandOnServer(s.privKey, s.server, fmt.Sprintf("echo '%s' >> test-driver.yml", testDriverFile))
	if err != nil {
		return fmt.Errorf("%s send testdriverfile file: %s %v", op, "templates/testdrivers/1.18.yml", err)
	}

	fmt.Printf("%s: Execute parallel e2e.test\n", op)
	err = RunCommandOnServer(s.privKey, s.server, fmt.Sprintf("KUBECONFIG=/root/.kube/config ./ginkgo -p -v -focus='External.Storage' -skip='\\[Feature:|\\[Disruptive\\]|\\[Serial\\]' ./e2e.test -- -storage.testdriver=test-driver.yml"))
	if err != nil {
		return fmt.Errorf("%s run e2e tests: %s", op, err)
	}

	fmt.Printf("%s: Execute serial e2e.test\n", op)
	err = RunCommandOnServer(s.privKey, s.server, fmt.Sprintf("KUBECONFIG=/root/.kube/config ./ginkgo -v -focus='External.Storage.*(\\[Feature:|\\[Serial\\])' ./e2e.test -- -storage.testdriver=test-driver.yml"))
	if err != nil {
		return fmt.Errorf("%s run e2e tests: %s", op, err)
	}
	return nil
}

func (s *hcloudK8sSetup) prepareCSIDriverDeploymentFile() error {
	const op = "hcloudK8sSetup/prepareCSIDriverDeploymentFile"
	fmt.Printf("%s: Read master deployment filee\n", op)
	deploymentFile, err := ioutil.ReadFile("../deploy/kubernetes/hcloud-csi-master.yml")
	if err != nil {
		return fmt.Errorf("%s: read csi driver deployment file %s: %v", op, "../deploy/kubernetes/hcloud-csi-master.yml", err)
	}

	fmt.Printf("%s: Prepare deployment file and transfer it\n", op)
	deploymentFile = []byte(strings.ReplaceAll(string(deploymentFile), "hetznercloud/hcloud-csi-driver:latest", fmt.Sprintf("hcloud-csi-driver:ci_%s", s.TestIdentifier)))
	deploymentFile = []byte(strings.ReplaceAll(string(deploymentFile), " imagePullPolicy: Always", " imagePullPolicy: IfNotPresent"))

	err = RunCommandOnServer(s.privKey, s.server, fmt.Sprintf("echo '%s' >> csi-driver.yml", deploymentFile))
	if err != nil {
		return fmt.Errorf("%s: Prepare deployment file and transfer it: %s", op, err)
	}
	return nil
}

func (s *hcloudK8sSetup) transferCSIDockerImage() error {
	const op = "hcloudK8sSetup/transferCSIDockerImage"
	fmt.Printf("%s: Transfer docker image\n", op)
	err := WithSSHSession(s.privKey, s.server.PublicNet.IPv4.IP.String(), func(session *ssh.Session) error {
		file, err := os.Open("ci-hcloud-csi-driver.tar")
		if err != nil {
			return fmt.Errorf("%s read ci-hcloud-csi-driver.tar: %s", op, err)
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

func (s *hcloudK8sSetup) waitForCloudInit() error {
	const op = "hcloudK8sSetup/PrepareTestEnv"
	fmt.Printf("%s: Wait for cloud-init\n", op)
	err := RunCommandOnServer(s.privKey, s.server, fmt.Sprintf("cloud-init status --wait > /dev/null"))
	if err != nil {
		return fmt.Errorf("%s: Wait for cloud-init: %s", op, err)
	}
	return nil
}
func (s *hcloudK8sSetup) TearDown(ctx context.Context) error {
	const op = "hcloudK8sSetup/TearDown"
	_, err := s.Hcloud.Server.Delete(ctx, s.server)
	if err != nil {
		return fmt.Errorf("%s Hcloud.Server.Delete: %s", op, err)
	}
	s.server = nil
	_, err = s.Hcloud.SSHKey.Delete(ctx, s.sshKey)
	if err != nil {
		return fmt.Errorf("%s Hcloud.SSHKey.Delete: %s", err, err)
	}
	s.sshKey = nil
	return nil
}

func (s *hcloudK8sSetup) getCloudInitConfig() (string, error) {
	const op = "hcloudK8sSetup/getCloudInitConfig"
	str, err := ioutil.ReadFile("templates/cloudinit.txt.tpl")
	if err != nil {
		return "", fmt.Errorf("%s: read template file %s: %v", "templates/cloudinit.txt.tpl", op, err)
	}
	tmpl, err := template.New("cloud_init").Parse(string(str))
	if err != nil {
		return "", fmt.Errorf("%s: parsing template file %s: %v", "templates/cloudinit.txt.tpl", op, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cloudInitTmpl{K8sVersion: s.K8sVersion, HcloudToken: s.HcloudToken}); err != nil {
		return "", fmt.Errorf("%s: execute template: %v", op, err)
	}
	return buf.String(), nil
}

func (s *hcloudK8sSetup) getSSHKey(ctx context.Context) error {
	const op = "hcloudK8sSetup/getSSHKey"
	pubKey, privKey, err := MakeSSHKeyPair()
	if err != nil {
		return err
	}
	sshKey, _, err := s.Hcloud.SSHKey.Create(ctx, hcloud.SSHKeyCreateOpts{
		Name:      fmt.Sprintf("s-%s", s.TestIdentifier),
		PublicKey: pubKey,
		Labels:    map[string]string{"K8sVersion": s.K8sVersion, "test": s.TestIdentifier},
	})
	if err != nil {
		return fmt.Errorf("%s: creating ssh key: %v", op, err)
	}
	s.privKey = privKey
	s.sshKey = sshKey
	return nil
}

func MakeSSHKeyPair() (string, string, error) {
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
