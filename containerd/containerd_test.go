package containerd

import (
	"strings"
	"testing"
)

func TestSanitizeFallbackMountOptions(t *testing.T) {
	// Options reported in ThreatMapper Issue #2432
	options := []string{
		"index=off",
		"workdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1443/work",
		"upperdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1443/fs",
		"lowerdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/303/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/302/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/301/fs",
	}

	mountedHostPath := "/fenced/mnt/host"
	mergedLower := SanitizeFallbackMountOptions(options, mountedHostPath)

	// 1. workdir must NOT be in mergedLower
	if strings.Contains(mergedLower, "work") {
		t.Fatalf("expected workdir to be completely omitted, but found: %s", mergedLower)
	}

	// 2. upperdir must be prefixed with mountedHostPath
	expectedUpper := "/fenced/mnt/host/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/1443/fs"
	if !strings.HasPrefix(mergedLower, expectedUpper+":") {
		t.Fatalf("expected upperdir as first layer with host prefix, got: %s", mergedLower)
	}

	// 3. lowerdir layers must be prefixed with mountedHostPath
	expectedLower := "/fenced/mnt/host/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/303/fs"
	if !strings.Contains(mergedLower, expectedLower) {
		t.Fatalf("expected lower layers to have host prefix, got: %s", mergedLower)
	}

	// 4. Test when only lowerdir is present (read-only container without upperdir)
	onlyLower := []string{
		"index=off",
		"lowerdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/100/fs",
	}
	mergedOnlyLower := SanitizeFallbackMountOptions(onlyLower, mountedHostPath)
	expectedOnlyLower := "/fenced/mnt/host/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/100/fs"
	if mergedOnlyLower != expectedOnlyLower {
		t.Fatalf("expected %s, got %s", expectedOnlyLower, mergedOnlyLower)
	}
}
