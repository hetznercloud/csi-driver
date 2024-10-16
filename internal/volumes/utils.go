package volumes

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"k8s.io/apimachinery/pkg/util/version"
)

const (
	XFSDefaultConfigsLocation = "/usr/share/xfsprogs/mkfs/"
	XFSConfigMatchPattern     = "lts_[0-9]*.[0-9]*.conf"
	XFSGlobMatchPattern       = XFSDefaultConfigsLocation + XFSConfigMatchPattern
)

func GetKernelVersion() (*version.Version, error) {
	var utsname syscall.Utsname
	if err := syscall.Uname(&utsname); err != nil {
		return nil, err
	}

	data := make([]byte, 65)
	for i := range 65 {
		if utsname.Release[i] == 0 {
			break
		}
		data[i] = byte(utsname.Release[i])
	}

	versionString := strings.Trim(string(data), "\x00")

	return version.ParseSemantic(versionString)
}

func GetXFSConfigPath(current *version.Version) (string, error) {
	filepaths, err := filepath.Glob(XFSGlobMatchPattern)
	if err != nil {
		return "", err
	}

	filenames := make([]string, 0, len(filepaths))
	for _, path := range filepaths {
		filename := strings.TrimPrefix(path, XFSDefaultConfigsLocation)
		filenames = append(filenames, filename)
	}

	supportedVersions := make([]*version.Version, 0, len(filepaths))
	for _, filename := range filenames {
		versionString := strings.TrimSuffix(strings.TrimPrefix(filename, "lts_"), ".conf")
		versionString = fmt.Sprintf("%s.0", versionString)
		kernelVersionXFS, err := version.ParseSemantic(versionString)
		if err != nil {
			continue
		}
		supportedVersions = append(supportedVersions, kernelVersionXFS)
	}

	sort.Slice(supportedVersions, func(i, j int) bool {
		return supportedVersions[i].GreaterThan(supportedVersions[j])
	})

	for _, supported := range supportedVersions {
		if current.AtLeast(supported) || current.GreaterThan(supported) {
			configName := fmt.Sprintf("lts_%d.%d.conf", supported.Major(), supported.Minor())
			return path.Join(XFSDefaultConfigsLocation, configName), nil
		}
	}

	return "", fmt.Errorf("no suitable mkfs.xfs config found for version: %s; tested versions: %v", current.String(), supportedVersions)
}
