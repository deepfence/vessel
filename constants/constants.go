package constants

import "time"

const (
	UnixProtocol      = "unix"
	Timeout           = 10 * time.Second
	CONTAINERD_K8S_NS = "k8s.io"
	CONTAINERD        = "containerd"
	DOCKER            = "docker"
)

var SupportedRuntimes = map[string]string{
	"unix:///var/run/docker.sock": DOCKER,
	"unix:///run/containerd/containerd.sock": CONTAINERD,
}
