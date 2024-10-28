package volumes

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

const cryptsetupExecuable = "cryptsetup"

type CryptSetup struct {
	logger *slog.Logger
}

func NewCryptSetup(logger *slog.Logger) *CryptSetup {
	return &CryptSetup{logger: logger}
}

func (cs *CryptSetup) IsActive(luksDeviceName string) (bool, error) {
	output, code, err := command(cryptsetupExecuable, "status", luksDeviceName)
	if err != nil {
		if code == 4 {
			return false, nil
		}
		return false, fmt.Errorf("unable to check LUKS device %s activity: %s", luksDeviceName, output)
	}
	return true, nil
}

func (cs *CryptSetup) Format(devicePath string, passphrase string) error {
	cs.logger.Info(
		"formatting LUKS device",
		"devicePath", devicePath,
	)
	output, _, err := commandWithStdin(passphrase, cryptsetupExecuable, "luksFormat", "--type", "luks1", devicePath)
	if err != nil {
		return fmt.Errorf("unable to format device %s with LUKS: %s", devicePath, output)
	}
	return nil
}

func (cs *CryptSetup) Open(devicePath string, luksDeviceName string, passphrase string) error {
	active, err := cs.IsActive(luksDeviceName)
	if err != nil {
		return err
	}
	if active {
		return nil
	}
	cs.logger.Info(
		"opening LUKS device",
		"devicePath", devicePath,
		"luksDeviceName", luksDeviceName,
	)
	output, _, err := commandWithStdin(passphrase, cryptsetupExecuable, "luksOpen", "--allow-discards", devicePath, luksDeviceName)
	if err != nil {
		return fmt.Errorf("unable to open LUKS device %s: %s", devicePath, output)
	}
	return nil
}

func (cs *CryptSetup) Close(luksDeviceName string) error {
	active, err := cs.IsActive(luksDeviceName)
	if err != nil {
		return err
	}
	if !active {
		return nil
	}
	cs.logger.Info(
		"closing LUKS device",
		"luksDeviceName", luksDeviceName,
	)
	output, _, err := command(cryptsetupExecuable, "luksClose", luksDeviceName)
	if err != nil {
		return fmt.Errorf("unable to close LUKS device %s: %s", luksDeviceName, output)
	}
	return nil
}

func (cs *CryptSetup) Resize(luksDeviceName string) error {
	cs.logger.Info(
		"resizing LUKS device",
		"luksDeviceName", luksDeviceName,
	)
	output, _, err := command(cryptsetupExecuable, "resize", luksDeviceName)
	if err != nil {
		return fmt.Errorf("unable to resize LUKS device %s: %s", luksDeviceName, output)
	}
	return nil
}

func GenerateLUKSDeviceName(devicePath string) string {
	segments := strings.Split(devicePath, "/")
	return segments[len(segments)-1]
}

func GenerateLUKSDevicePath(luksDeviceName string) string {
	return "/dev/mapper/" + luksDeviceName
}

func command(name string, args ...string) (string, int, error) {
	return commandWithStdin("", name, args...)
}

func commandWithStdin(stdin string, name string, args ...string) (string, int, error) {
	cmd := exec.Command(name, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	outputBytes, err := cmd.CombinedOutput()
	output := string(outputBytes)
	if err != nil {
		exitError, ok := err.(*exec.ExitError)
		if !ok {
			return output, 0, err
		}
		return output, exitError.ExitCode(), fmt.Errorf("%w\n%s", exitError, output)
	}
	return output, 0, nil
}
