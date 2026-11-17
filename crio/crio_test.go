package crio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// installCrictlShim puts a crictl on PATH that answers `config` quietly and
// prints the given body for `inspect`, so the real ExtractFileSystemContainer
// can be driven without a cri-o socket.
func installCrictlShim(t *testing.T, inspectBody string) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\ncase \"$1\" in\n  config) exit 0 ;;\n  inspect) printf '%s' '" + inspectBody + "' ; exit 0 ;;\nesac\nexit 1\n"
	path := filepath.Join(dir, "crictl")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("writing crictl shim: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// crictl renders the go-template inside literal quotes, so a container whose
// root path is empty comes back as `""`. The emptiness check has to run on the
// trimmed value, because that is the one handed to tar.
func TestExtractFileSystemContainerRejectsEmptyRootPath(t *testing.T) {
	installCrictlShim(t, `""`+"\n")

	out := filepath.Join(t.TempDir(), "fs.tar")
	err := New("unix:///var/run/crio/crio.sock").ExtractFileSystemContainer("abc123", "", out)

	if err == nil {
		t.Fatalf("expected an error for an empty root path, got nil (output tar: %s)", out)
	}
	if !strings.Contains(err.Error(), "container root path is empty") {
		t.Fatalf("expected the empty-root-path error, got: %v", err)
	}
}

// Positive control: a real directory must still be archived, so a failure above
// is the guard and not a broken harness.
func TestExtractFileSystemContainerArchivesRealRootPath(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "marker.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatalf("seeding root path: %v", err)
	}
	installCrictlShim(t, `"`+src+`"`+"\n")

	out := filepath.Join(t.TempDir(), "fs.tar")
	if err := New("unix:///var/run/crio/crio.sock").ExtractFileSystemContainer("abc123", "", out); err != nil {
		t.Fatalf("expected the real root path to be archived, got: %v", err)
	}
	fi, err := os.Stat(out)
	if err != nil {
		t.Fatalf("expected an output tar: %v", err)
	}
	if fi.Size() == 0 {
		t.Fatalf("expected a non-empty output tar")
	}
}
