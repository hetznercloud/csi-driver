package volumes

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

const (
	XFSDefaultConfigsLocation = "/usr/share/xfsprogs/mkfs/"
	XFSConfigMatchPattern     = "lts_[0-9]*.[0-9]*.conf"
	XFSGlobMatchPattern       = XFSDefaultConfigsLocation + XFSConfigMatchPattern
)

type KernelVersion struct {
	Major int
	Minor int
	Patch int
}

func NewKernelVersion(major int, minor int, patch int) *KernelVersion {
	return &KernelVersion{
		Major: major,
		Minor: minor,
		Patch: patch,
	}
}

func ParseKernelVersion(versionString string) (*KernelVersion, error) {
	releaseString := strings.Trim(versionString, "\x00")
	parts := strings.Split(releaseString, "-")
	mmpParts := strings.Split(parts[0], ".") // Major.Minor.Patch

	versions := make([]int, 3)
	for i, part := range mmpParts {
		version, err := strconv.Atoi(part)
		if err != nil {
			return &KernelVersion{}, err
		}
		versions[i] = version
	}

	return &KernelVersion{
		Major: versions[0],
		Minor: versions[1],
		Patch: versions[2],
	}, nil
}

func getKernelVersion() (*KernelVersion, error) {
	var utsname syscall.Utsname
	if err := syscall.Uname(&utsname); err != nil {
		return &KernelVersion{}, err
	}

	data := make([]byte, 65)
	for i := range 65 {
		if utsname.Release[i] == 0 {
			break
		}
		data[i] = byte(utsname.Release[i])
	}

	return ParseKernelVersion(string(data))
}

func (k *KernelVersion) IsNewerThan(b *KernelVersion) bool {
	if k.Major != b.Major {
		return k.Major > b.Major
	}

	if k.Minor != b.Minor {
		return k.Minor > b.Minor
	}

	return k.Patch > b.Patch
}

func GetXFSConfigPath(current *KernelVersion) string {
	filepaths, err := filepath.Glob(XFSGlobMatchPattern)
	if err != nil {
		return ""
	}

	filenames := make([]string, 0, len(filepaths))
	for _, path := range filepaths {
		filename := strings.TrimPrefix(path, XFSDefaultConfigsLocation)
		filenames = append(filenames, filename)
	}

	supportedVersions := make([]KernelVersion, 0, len(filepaths))
	for _, filename := range filenames {
		versionString := strings.TrimSuffix(strings.TrimPrefix(filename, "lts_"), ".conf")
		parts := strings.Split(versionString, ".")
		major, err := strconv.Atoi(parts[0])
		if err != nil {
			return ""
		}
		minor, err := strconv.Atoi(parts[1])
		if err != nil {
			return ""
		}
		kernelVersionXFS := KernelVersion{
			Major: major,
			Minor: minor,
		}

		supportedVersions = append(supportedVersions, kernelVersionXFS)
	}

	sort.Slice(supportedVersions, func(i, j int) bool {
		return supportedVersions[i].IsNewerThan(&supportedVersions[j])
	})

	for _, supported := range supportedVersions {
		if *current == supported || current.IsNewerThan(&supported) {
			configName := fmt.Sprintf("lts_%d.%d.conf", supported.Major, supported.Minor)
			return path.Join(XFSDefaultConfigsLocation, configName)
		}
	}

	return ""
}
