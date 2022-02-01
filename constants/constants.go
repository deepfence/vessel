package constants

import "time"

const (
	UnixProtocol                  = "unix"
	Timeout                       = 10 * time.Second
	CONTAINERD_K8S_NS             = "k8s.io"
	CONTAINERD                    = "containerd"
	K3S_CONTAINERD                = "k3s_containerd"
	DOCKER                        = "docker"
	CONTAINERD_SOCKET_ADDRESS     = "/run/containerd/containerd.sock"
	K3S_CONTAINERD_SOCKET_ADDRESS = "/run/k3s/containerd/containerd.sock"
	DOCKER_SOCKET_ADDRESS         = "/var/run/docker.sock"
	CONTAINERD_SOCKET_IRI         = "unix://" + CONTAINERD_SOCKET_ADDRESS
	DOCKER_SOCKET_IRI             = "unix://" + DOCKER_SOCKET_ADDRESS
	K3S_CONTAINERD_SOCKET_IRI     = "unix://" + K3S_CONTAINERD_SOCKET_ADDRESS
)

var SupportedRuntimes = map[string]string{
	DOCKER:         DOCKER_SOCKET_IRI,
	CONTAINERD:     CONTAINERD_SOCKET_IRI,
	K3S_CONTAINERD: K3S_CONTAINERD_SOCKET_IRI,
}
