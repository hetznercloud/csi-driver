package csi

// Volume represents a volume in the CSI driver domain.
type Volume struct {
	ID          uint64
	Name        string
	Size        int // GB
	Location    string
	LinuxDevice string
	Server      *Server
}

func (v Volume) SizeBytes() int64 {
	return int64(v.Size) * 1024 * 1024 * 1024
}

// IsAttached returns true if the volume is not attached to
// a server.
func (v Volume) IsAttached() bool {
	return v.Server != nil
}

// IsMounted returns true if the volume has been mounted to a
// Linux server and has a mount path.
func (v Volume) IsMounted() bool {
	return v.LinuxDevice != ""
}
