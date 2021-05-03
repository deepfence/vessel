package vessel

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os/exec"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// GetAddressAndDialer returns the address parsed from the given endpoint and a context dialer.
func GetAddressAndDialer(endpoint string) (string, func(ctx context.Context, addr string) (net.Conn, error), error) {
	protocol, addr, err := parseEndpointWithFallbackProtocol(endpoint, unixProtocol)
	if err != nil {
		return "", nil, err
	}
	if protocol != unixProtocol {
		return "", nil, fmt.Errorf("only support unix socket endpoint")
	}

	return addr, dial, nil
}

func dial(ctx context.Context, addr string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, unixProtocol, addr)
}

func parseEndpointWithFallbackProtocol(endpoint string, fallbackProtocol string) (protocol string, addr string, err error) {
	if protocol, addr, err = parseEndpoint(endpoint); err != nil && protocol == "" {
		fallbackEndpoint := fallbackProtocol + "://" + endpoint
		protocol, addr, err = parseEndpoint(fallbackEndpoint)
		if err == nil {
			logrus.Warningf("Using %q as endpoint is deprecated, please consider using full url format %q.", endpoint, fallbackEndpoint)
		}
	}
	return
}

func parseEndpoint(endpoint string) (string, string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", "", err
	}

	switch u.Scheme {
	case "tcp":
		return "tcp", u.Host, nil

	case "unix":
		return "unix", u.Path, nil

	case "":
		return "", "", fmt.Errorf("using %q as endpoint is deprecated, please consider using full url format", endpoint)

	default:
		return u.Scheme, "", fmt.Errorf("protocol %q not supported", u.Scheme)
	}
}

func getContainerRuntime(endPoints map[string]string) (string, error) {
	if endPoints == nil || len(endPoints) == 0 {
		return "", fmt.Errorf("endpoint is not set")
	}
	endPointsLen := len(endPoints)
	indx := 0
	var runtime string
	for r, endPoint := range endPoints {
		logrus.Infof("connect using endpoint '%s' with '%s' timeout", endPoint, Timeout)
		addr, dialer, err := GetAddressAndDialer(endPoint)
		if err != nil {
			if indx == endPointsLen-1 {
				return "", err
			}
			logrus.Error(err)
			continue
		}

		if r == "docker" {
			_, err = net.Dial(unixProtocol, addr)
			if err != nil {
				errMsg := errors.Wrapf(err, "connect endpoint '%s', make sure you are running as root and the endpoint has been started", endPoint, addr)
				if indx == endPointsLen-1 {
					return "", errMsg
				}
				logrus.Error(errMsg)
			} else {
				running, err := isDockerRunning()
				if err != nil {
					return "", err
				}

				if !running {
					errMsg := errors.Wrapf(err, "connect endpoint '%s', docker is not the runtime", endPoint, addr)
					if indx == endPointsLen-1 {
						return "", errMsg
					}
					logrus.Error(errMsg)
				} else {
					logrus.Infof("connected successfully using endpoint: %s", endPoint)
					runtime = r
					break
				}
			}
		} else {
			_, err = grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(Timeout), grpc.WithContextDialer(dialer))
			if err != nil {
				errMsg := errors.Wrapf(err, "connect endpoint '%s', make sure you are running as root and the endpoint has been started", endPoint, addr)
				if indx == endPointsLen-1 {
					return "", errMsg
				}
				logrus.Error(errMsg)
				logrus.Infof("trying next socket...")
			} else {
				running, err := isContainerdRunning()
				if err != nil {
					return "", err
				}

				if !running {
					errMsg := errors.Wrapf(err, "connect endpoint '%s', containerd is not the runtime", endPoint, addr)
					if indx == endPointsLen-1 {
						return "", errMsg
					}
					logrus.Error(errMsg)
				} else {
					logrus.Infof("connected successfully using endpoint: %s", endPoint)
					runtime = r
					break
				}

			}
		}

	}
	return runtime, nil
}

// AutoDetectRuntime auto detects the underlying container runtime like docker, containerd
func AutoDetectRuntime() (string, error) {
	logrus.Info("trying to auto-detect container runtime...")
	runtime, err := getContainerRuntime(supportedRuntimes)
	if err != nil {
		return "", err
	}
	return runtime, nil
}

func isDockerRunning() (bool, error) {
	cli, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return false, errors.Wrapf(err, " :error creating docker clientset")
	}

	if len(containers) > 0 {
		for _, container := range containers {
			fmt.Printf("Container name: %s \n", container.Image)
		}
		return true, nil
	}

	return false, nil
}

// getHostName returns the container id
func getHostName() ([]byte, error) {
	return exec.Command("hostname").Output()
}

func isContainerdRunning() (bool, error) {
	clientd, err := containerd.New("/run/containerd/containerd.sock")
	defer clientd.Close()

	// create a context for k8s with containerd namespace
	// TODO: using k8s ns, to support containerd standalone
	// make this configurable or autodetect
	k8s := namespaces.WithNamespace(context.Background(), CONTAINERD_K8S_NS)

	containers, err := clientd.Containers(k8s)
	if err != nil {
		return false, errors.Wrapf(err, " :error creating docker clientset")
	}

	if len(containers) > 0 {
		return true, nil
	}
	return false, nil
}
