package volumes

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

const cryptsetupExecutable = "cryptsetup"

type CryptSetup struct {
	logger *slog.Logger
}

func NewCryptSetup(logger *slog.Logger) *CryptSetup {
	return &CryptSetup{logger: logger}
}

func (cs *CryptSetup) Status(ctx context.Context, luksDeviceName string) (LUKSDeviceStatus, error) {
	output, code, err := cryptsetup(ctx, "status", luksDeviceName)
	if err != nil {
		if code == 4 {
			return LUKSDeviceStatus{Active: false}, nil
		}
		return LUKSDeviceStatus{Active: false}, fmt.Errorf("unable to check LUKS device %s status: %w", luksDeviceName, err)
	}

	return NewLUKSDeviceStatus(output), nil
}

func (cs *CryptSetup) Format(ctx context.Context, devicePath string, passphrase string) error {
	cs.logger.Info(
		"formatting LUKS device",
		"devicePath", devicePath,
	)
	output, _, err := cryptsetupWithStdin(ctx, passphrase, "luksFormat", "--type", "luks1", devicePath)
	if err != nil {
		return fmt.Errorf("unable to format device %s with LUKS: %s", devicePath, output)
	}
	return nil
}

func (cs *CryptSetup) Open(ctx context.Context, devicePath string, luksDeviceName string, passphrase string) error {
	status, err := cs.Status(ctx, luksDeviceName)
	if err != nil {
		return err
	}
	if status.Active {
		if sameDevice(status.Device, devicePath) {
			cs.logger.Debug(
				"LUKS device already active",
				"devicePath", devicePath,
				"luksDeviceName", luksDeviceName,
			)
			return nil
		}

		// A volume detached without being unpublished leaves behind a mapper whose backing
		// device is gone. Reusing it would format and mount a dead device.
		cs.logger.Warn(
			"LUKS device is not backed by the expected device, closing it",
			"devicePath", devicePath,
			"luksDeviceName", luksDeviceName,
			"backingDevice", status.Device,
		)
		if err := cs.Close(ctx, luksDeviceName); err != nil {
			return fmt.Errorf("unable to close stale LUKS device %s: %w", luksDeviceName, err)
		}
	}

	cs.logger.Info(
		"opening LUKS device",
		"devicePath", devicePath,
		"luksDeviceName", luksDeviceName,
	)
	output, _, err := cryptsetupWithStdin(ctx, passphrase, "luksOpen", "--allow-discards", devicePath, luksDeviceName)
	if err != nil {
		return fmt.Errorf("unable to open LUKS device %s: %s", devicePath, output)
	}
	return nil
}

func (cs *CryptSetup) Close(ctx context.Context, luksDeviceName string) error {
	status, err := cs.Status(ctx, luksDeviceName)
	if err != nil {
		return err
	}
	if !status.Active {
		return nil
	}
	cs.logger.Info(
		"closing LUKS device",
		"luksDeviceName", luksDeviceName,
	)
	output, _, err := cryptsetup(ctx, "luksClose", luksDeviceName)
	if err != nil {
		return fmt.Errorf("unable to close LUKS device %s: %s", luksDeviceName, output)
	}
	return nil
}

func (cs *CryptSetup) Resize(ctx context.Context, luksDeviceName string) error {
	cs.logger.Info(
		"resizing LUKS device",
		"luksDeviceName", luksDeviceName,
	)
	output, _, err := cryptsetup(ctx, "resize", luksDeviceName)
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

func cryptsetup(ctx context.Context, args ...string) (string, int, error) {
	return cryptsetupWithStdin(ctx, "", args...)
}

func cryptsetupWithStdin(ctx context.Context, stdin string, args ...string) (string, int, error) {
	cmd := exec.CommandContext(ctx, cryptsetupExecutable, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	outputBytes, err := cmd.CombinedOutput()
	output := string(outputBytes)
	if err != nil {
		exitError := &exec.ExitError{}
		if !errors.As(err, &exitError) {
			return output, 0, err
		}
		return output, exitError.ExitCode(), fmt.Errorf("%w\n%s", exitError, output)
	}
	return output, 0, nil
}

type LUKSDeviceStatus struct {
	Active bool
	Device string
}

func NewLUKSDeviceStatus(output string) LUKSDeviceStatus {
	// An inactive device is reported through the exit code, not the output.
	status := LUKSDeviceStatus{Active: true}
	for line := range strings.SplitSeq(output, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if ok && key == "device" {
			status.Device = strings.TrimSpace(value)
		}
	}
	return status
}

func sameDevice(a string, b string) bool {
	infoA, err := os.Stat(a)
	if err != nil || infoA.Mode()&os.ModeDevice == 0 {
		return false
	}
	infoB, err := os.Stat(b)
	if err != nil || infoB.Mode()&os.ModeDevice == 0 {
		return false
	}
	return infoA.Sys().(*syscall.Stat_t).Rdev == infoB.Sys().(*syscall.Stat_t).Rdev
}
