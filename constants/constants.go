package constants

import (
	"os"
	"time"
)

const (
	UnixProtocol                  = "unix"
	Timeout                       = 10 * time.Second
	CONTAINERD_K8S_NS             = "k8s.io"
	CONTAINERD                    = "containerd"
	DOCKER                        = "docker"
	CONTAINERD_SOCKET_ADDRESS     = "/run/containerd/containerd.sock"
	K3S_CONTAINERD_SOCKET_ADDRESS = "/run/k3s/containerd/containerd.sock"
	DOCKER_SOCKET_ADDRESS         = "/var/run/docker.sock"
	CONTAINERD_SOCKET_URI         = "unix://" + CONTAINERD_SOCKET_ADDRESS
	DOCKER_SOCKET_URI             = "unix://" + DOCKER_SOCKET_ADDRESS
	K3S_CONTAINERD_SOCKET_URI     = "unix://" + K3S_CONTAINERD_SOCKET_ADDRESS
)

var SupportedRuntimes = map[string][]string{
	DOCKER:     {DOCKER_SOCKET_URI},
	CONTAINERD: {CONTAINERD_SOCKET_URI, K3S_CONTAINERD_SOCKET_URI},
}

func init() {
	dockerSockerPath := os.Getenv("DOCKER_SOCKET_PATH")
	if dockerSockerPath != "" {
		SupportedRuntimes[DOCKER] = append(SupportedRuntimes[DOCKER], "unix://"+dockerSockerPath)
	}
	containerdSockerPath := os.Getenv("CONTAINERD_SOCKET_PATH")
	if containerdSockerPath != "" {
		SupportedRuntimes[CONTAINERD] = append(SupportedRuntimes[CONTAINERD], "unix://"+containerdSockerPath)
	}
}
