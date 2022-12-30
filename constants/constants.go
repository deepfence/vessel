package constants

import (
	"os"
	"time"
)

const (
	UnixProtocol                  = "unix"
	Timeout                       = 8 * time.Second
	CONTAINERD_K8S_NS             = "k8s.io"
	CONTAINERD                    = "containerd"
	DOCKER                        = "docker"
	CRIO                          = "crio"
	CONTAINERD_SOCKET_ADDRESS     = "/run/containerd/containerd.sock"
	K3S_CONTAINERD_SOCKET_ADDRESS = "/run/k3s/containerd/containerd.sock"
	DOCKER_SOCKET_ADDRESS         = "/var/run/docker.sock"
	CRIO_SOCKET_ADDRESS           = "/var/run/crio/crio.sock"
	CONTAINERD_SOCKET_URI         = "unix://" + CONTAINERD_SOCKET_ADDRESS
	DOCKER_SOCKET_URI             = "unix://" + DOCKER_SOCKET_ADDRESS
	K3S_CONTAINERD_SOCKET_URI     = "unix://" + K3S_CONTAINERD_SOCKET_ADDRESS
	CRIO_SOCKET_URI               = "unix://" + CRIO_SOCKET_ADDRESS
)

var SupportedRuntimes = map[string][]string{
	DOCKER:     {DOCKER_SOCKET_URI},
	CONTAINERD: {CONTAINERD_SOCKET_URI, K3S_CONTAINERD_SOCKET_URI},
	CRIO:       {CRIO_SOCKET_URI},
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
	crioSockerPath := os.Getenv("CRIO_SOCKET_PATH")
	if crioSockerPath != "" {
		SupportedRuntimes[CRIO] = append(SupportedRuntimes[CRIO], "unix://"+crioSockerPath)
	}
}
