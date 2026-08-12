//go:build integration

package volumes

// SetDeviceIdentityPaths points the device identity check at fixture directories.
//
// The integration tests publish files used as fake devices. Those have neither a by-id
// symlink, which udev maintains on a node, nor a vpd_pg80, which the kernel exposes for a
// SCSI disk and which can not be created because sysfs is not writable. The tests create
// both under a temporary directory and call this, so that publishing takes the same path
// it takes on a node instead of one that skips the check.
//
// This file is only built with the `integration` build tag, so the paths of the shipped
// driver can not be redirected.
func SetDeviceIdentityPaths(byIDPrefix, sysClassBlock string) {
	volumeDevicePrefix = byIDPrefix
	sysClassBlockPath = sysClassBlock
}
