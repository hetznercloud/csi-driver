package integration

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

const dockerExecutable = "docker"

func DockerBuild(imageName string, dir string) (string, error) {
	dockerArgs := []string{"build"}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		dockerArgs = []string{"buildx", "build", "--platform=linux/amd64", "--load"}
	}
	dockerArgs = append(dockerArgs, "-t", imageName, dir)
	return runCmd(dockerExecutable, dockerArgs...)
}

func DockerSave(imageName string, output string) (string, error) {
	dockerArgs := []string{"save", "--output", output, imageName}
	return runCmd(dockerExecutable, dockerArgs...)
}

func DockerRun(imageName string, envs []string, argv []string, privileged bool) (string, error) {
	dockerArgs := []string{"run", "--rm"}
	for _, env := range envs {
		dockerArgs = append(dockerArgs, "-e", env)
	}
	if privileged {
		dockerArgs = append(dockerArgs, "--privileged")
	}
	dockerArgs = append(dockerArgs, imageName)
	dockerArgs = append(dockerArgs, argv...)
	return runCmd(dockerExecutable, dockerArgs...)
}

func runCmd(name string, args ...string) (string, error) {
	return runCmdWithStdin("", name, args...)
}

func runCmdWithStdin(stdin string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	outputBytes, err := cmd.CombinedOutput()
	output := string(outputBytes)
	if err != nil {
		return output, fmt.Errorf("run command %s failed: %w\n", strings.Join(append([]string{name}, args...), " "), err)
	}
	return output, nil
}
