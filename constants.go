package vessel

import "time"

const (
	unixProtocol      = "unix"
	Timeout           = 2 * time.Second
	CONTAINERD_K8S_NS = "k8s.io"
)

var supportedRuntimes = map[string]string{
	"docker":     "unix:///var/run/docker.sock",
	"containerd": "unix:///run/containerd/containerd.sock",
}
