package volumes

import (
	"fmt"
	"log/slog"

	"golang.org/x/sys/unix"

	"github.com/hetznercloud/csi-driver/internal/utils"
)

// StatsService get statistics about mounted volumes.
type StatsService interface {
	ByteFilesystemStats(volumePath string) (totalBytes int64, availableBytes int64, usedBytes int64, err error)
	INodeFilesystemStats(volumePath string) (total int64, used int64, free int64, err error)
}

// LinuxStatsService mounts volumes on a Linux system.
type LinuxStatsService struct {
	logger *slog.Logger
}

func NewLinuxStatsService(logger *slog.Logger) *LinuxStatsService {
	return &LinuxStatsService{
		logger: logger,
	}
}

func (l *LinuxStatsService) ByteFilesystemStats(volumePath string) (totalBytes int64, availableBytes int64, usedBytes int64, err error) {
	statfs := &unix.Statfs_t{}
	err = unix.Statfs(volumePath, statfs)
	if err != nil {
		return
	}

	// golang.org/x/sys/unix returns a 32-bit integer on 32-bit systems (ARMv6)
	// ensure it is converted to int64
	bsize := int64(statfs.Bsize)

	bavail, err := utils.UInt64ToInt64(statfs.Bavail)
	if err != nil {
		err = fmt.Errorf("error converting available blocks: %w", err)
		return
	}

	blocks, err := utils.UInt64ToInt64(statfs.Blocks)
	if err != nil {
		err = fmt.Errorf("error converting blocks: %w", err)
		return
	}

	bfree, err := utils.UInt64ToInt64(statfs.Bfree)
	if err != nil {
		err = fmt.Errorf("error converting free blocks: %w", err)
		return
	}

	availableBytes = bavail * bsize
	usedBytes = (blocks - bfree) * bsize
	totalBytes = blocks * bsize

	return
}

func (l *LinuxStatsService) INodeFilesystemStats(volumePath string) (total int64, used int64, free int64, err error) {
	statfs := &unix.Statfs_t{}
	err = unix.Statfs(volumePath, statfs)
	if err != nil {
		return
	}

	total, err = utils.UInt64ToInt64(statfs.Files)
	if err != nil {
		err = fmt.Errorf("error converting total inodes: %w", err)
		return
	}

	free, err = utils.UInt64ToInt64(statfs.Ffree)
	if err != nil {
		err = fmt.Errorf("error converting free inodes: %w", err)
		return
	}

	used = total - free

	return
}
