package constants

import "time"

const (
	UnixProtocol      = "unix"
	Timeout           = 2 * time.Second
	CONTAINERD_K8S_NS = "k8s.io"
	CONTAINERD        = "containerd"
	DOCKER            = "docker"
)

var SupportedRuntimes = map[string]string{
	"docker":     "unix:///var/run/docker.sock",
	"containerd": "unix:///run/containerd/containerd.sock",
}
