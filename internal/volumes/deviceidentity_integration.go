//go:build integration

package volumes

func SetDeviceIdentityPaths(dev, sysClassBlock string) {
	devPath = dev
	sysClassBlockPath = sysClassBlock
}
