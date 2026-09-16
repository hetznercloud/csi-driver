package csi

import "time"

// Snapshot represents a point-in-time block volume snapshot in the CSI driver domain.
type Snapshot struct {
	ID             int64
	Name           string
	SourceVolumeID int64
	Size           int // GB
	Location       string
	Created        time.Time
	Ready          bool
}

func (s Snapshot) SizeBytes() int64 {
	return int64(s.Size) * 1024 * 1024 * 1024
}
