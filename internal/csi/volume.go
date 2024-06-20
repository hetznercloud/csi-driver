package csi

// Volume represents a volume in the CSI driver domain.
type Volume struct {
	ID          int64
	Name        string
	Size        int // GB
	Location    string
	LinuxDevice string
	Server      *Server
}

func (v Volume) SizeBytes() int64 {
	return int64(v.Size) * 1024 * 1024 * 1024
}
