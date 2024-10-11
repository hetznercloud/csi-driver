package volumes

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

const XFSDefaultConfigsLocation = "/usr/share/xfsprogs/mkfs"

type KernelVersion struct {
	major int
	minor int
	patch int
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
		major: versions[0],
		minor: versions[1],
		patch: versions[2],
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
	if k.major != b.major {
		return k.major > b.major
	}

	if k.minor != b.minor {
		return k.minor > b.minor
	}

	return k.patch > b.patch
}

func GetXFSConfigPath(current *KernelVersion) string {
	files, err := os.ReadDir(XFSDefaultConfigsLocation)
	if err != nil {
		return ""
	}

	files = slices.DeleteFunc(files, func(file fs.DirEntry) bool {
		return !strings.HasPrefix(file.Name(), "lts_")
	})

	supportedVersions := make([]KernelVersion, 0, len(files))
	for _, file := range files {
		versionString := strings.TrimSuffix(strings.TrimPrefix(file.Name(), "lts_"), ".conf")
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
			major: major,
			minor: minor,
		}

		supportedVersions = append(supportedVersions, kernelVersionXFS)
	}

	sort.Slice(supportedVersions, func(i, j int) bool {
		return supportedVersions[i].IsNewerThan(&supportedVersions[j])
	})

	for _, supported := range supportedVersions {
		if *current == supported || current.IsNewerThan(&supported) {
			configName := fmt.Sprintf("lts_%d.%d.conf", supported.major, supported.minor)
			return path.Join(XFSDefaultConfigsLocation, configName)
		}
	}

	return ""
}
