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

func getKernelVersion() (KernelVersion, error) {
	var utsname syscall.Utsname
	if err := syscall.Uname(&utsname); err != nil {
		return KernelVersion{}, err
	}

	data := make([]byte, 65)
	for i := range 65 {
		if utsname.Release[i] == 0 {
			break
		}
		data[i] = byte(utsname.Release[i])
	}

	releaseString := strings.Trim(string(data), "\x00")
	parts := strings.Split(releaseString, "-")
	mmpParts := strings.Split(parts[0], ".") // Major.Minor.Patch

	versions := make([]int, 3)
	for i, part := range mmpParts {
		version, err := strconv.Atoi(part)
		if err != nil {
			return KernelVersion{}, err
		}
		versions[i] = version
	}

	return KernelVersion{
		major: versions[0],
		minor: versions[1],
		patch: versions[2],
	}, nil
}

func (k *KernelVersion) IsNewerThan(b KernelVersion) bool {
	if k.major > b.major {
		return true
	}
	if k.minor > b.minor {
		return true
	}
	if k.patch > b.patch {
		return true
	}
	return false
}

func GetXFSConfigPath() string {
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
		return supportedVersions[j].IsNewerThan(supportedVersions[i])
	})

	current, err := getKernelVersion()
	if err != nil {
		return ""
	}

	for _, supported := range supportedVersions {
		if current == supported || current.IsNewerThan(supported) {
			configName := fmt.Sprintf("lts_%d.%d.conf", supported.major, supported.minor)
			return path.Join(XFSDefaultConfigsLocation, configName)
		}
	}

	return ""
}
