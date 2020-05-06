package volumes

import (
	"github.com/go-kit/kit/log"
	"golang.org/x/sys/unix"
)

// StatsService get statistics about mounted volumes.
type StatsService interface {
	ByteFilesystemStats(volumePath string) (totalBytes int64, availableBytes int64, usedBytes int64, err error)
	INodeFilesystemStats(volumePath string) (total int64, used int64, free int64, err error)
}

// LinuxStatsService mounts volumes on a Linux system.
type LinuxStatsService struct {
	logger log.Logger
}

func NewLinuxStatsService(logger log.Logger) *LinuxStatsService {
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
	availableBytes = int64(statfs.Bavail) * int64(statfs.Bsize)
	usedBytes = (int64(statfs.Blocks) - int64(statfs.Bfree)) * int64(statfs.Bsize)
	totalBytes = int64(statfs.Blocks) * int64(statfs.Bsize)
	return
}

func (l *LinuxStatsService) INodeFilesystemStats(volumePath string) (total int64, used int64, free int64, err error) {
	statfs := &unix.Statfs_t{}
	err = unix.Statfs(volumePath, statfs)
	if err != nil {
		return
	}
	total = int64(statfs.Files)
	free = int64(statfs.Ffree)
	used = total - free
	return
}
