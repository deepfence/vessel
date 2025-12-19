package vessel

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"strings"
	"sync"

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	selfContainerd "github.com/deepfence/vessel/containerd"
	"github.com/deepfence/vessel/crio"
	"github.com/deepfence/vessel/docker"
	selfPodman "github.com/deepfence/vessel/podman"
	"github.com/deepfence/vessel/utils"
	"github.com/docker/docker/api/types/container"
	dockerClient "github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
}

// GetAddressAndDialer returns the address parsed from the given endpoint and a context dialer.
func GetAddressAndDialer(endpoint string) (string, func(ctx context.Context, addr string) (net.Conn, error), error) {
	protocol, addr, err := parseEndpointWithFallbackProtocol(endpoint, utils.UnixProtocol)
	if err != nil {
		return "", nil, err
	}
	if protocol != utils.UnixProtocol {
		return "", nil, fmt.Errorf("only support unix socket endpoint")
	}

	return addr, dial, nil
}

func dial(ctx context.Context, addr string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, utils.UnixProtocol, addr)
}

func parseEndpointWithFallbackProtocol(endpoint string, fallbackProtocol string) (protocol string, addr string, err error) {
	if protocol, addr, err = parseEndpoint(endpoint); err != nil && protocol == "" {
		fallbackEndpoint := fallbackProtocol + "://" + endpoint
		protocol, addr, err = parseEndpoint(fallbackEndpoint)
		if err == nil {
			log.Warn().Msgf("Using %q as endpoint is deprecated, please consider using full url format %q.", endpoint, fallbackEndpoint)
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

func checkDockerRuntime(endPoint string) (bool, error) {
	addr, _, err := GetAddressAndDialer(endPoint)
	if err != nil {
		return false, err
	}
	_, err = net.DialTimeout(utils.UnixProtocol, addr, utils.Timeout)
	if err != nil {
		return false, errors.New("could not connect to endpoint '" + endPoint + "'")
	}
	running, err := isDockerRunning(endPoint)
	if err != nil {
		return false, err
	}
	if !running {
		log.Debug().Msgf("no running containers found with endpoint %s", endPoint)
		return false, nil
	}
	return true, nil
}

func checkPodmanRuntime(endPoint string) (bool, error) {
	running, err := isPodmanRunning(endPoint)
	if err != nil {
		return false, err
	}
	if !running {
		log.Debug().Msgf("no running containers found with endpoint %s", endPoint)
		return false, nil
	}
	return true, nil
}

func checkContainerdRuntime(endPoint string) (bool, error) {
	addr, dialer, err := GetAddressAndDialer(endPoint)
	if err != nil {
		return false, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), utils.Timeout)
	defer cancel()
	_, err = grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock(), grpc.WithContextDialer(dialer))
	if err != nil {
		return false, errors.New("could not connect to endpoint '" + endPoint + "'")
	}
	running, err := isContainerdRunning(endPoint)
	if err != nil {
		return false, err
	}
	if !running {
		log.Debug().Msgf("no running containers found with endpoint %s", endPoint)
		return false, nil
	}
	return true, nil
}

func checkCrioRuntime(endPoint string) (bool, error) {
	addr, _, err := GetAddressAndDialer(endPoint)
	if err != nil {
		return false, err
	}
	_, err = net.DialTimeout(utils.UnixProtocol, addr, utils.Timeout)
	if err != nil {
		return false, errors.New("could not connect to endpoint '" + endPoint + "'")
	}
	running, err := isCRIORunning(endPoint)
	if err != nil {
		return false, err
	}
	if !running {
		log.Debug().Msgf("no running containers found with endpoint %s", endPoint)
		return false, nil
	}
	return true, nil
}

type containerRuntime struct {
	Runtime   string
	Endpoint  string
	Connected bool
}

// getContainerRuntime returns the underlying container runtime, and it's socket path
func getContainerRuntime() (string, string, error) {
	var wg sync.WaitGroup
	var detectedRuntimes []containerRuntime
	var connectedRuntimes []containerRuntime
	detectedRuntimeChannel := make(chan containerRuntime, 1)

	for runtime, endPoints := range utils.SupportedRuntimes {
		for _, endPoint := range endPoints {
			wg.Add(1)
			go func(runtime, endPoint string) {
				log.Debug().Msgf("trying to connect to endpoint '%s' with timeout '%s'", endPoint, utils.Timeout)
				var connected bool
				var err error
				switch runtime {
				case utils.DOCKER:
					connected, err = checkDockerRuntime(endPoint)
				case utils.CONTAINERD:
					connected, err = checkContainerdRuntime(endPoint)
				case utils.CRIO:
					connected, err = checkCrioRuntime(endPoint)
				case utils.PODMAN:
					connected, err = checkPodmanRuntime(endPoint)
				default:
					err = fmt.Errorf("unknown container runtime %s", runtime)
				}
				if err != nil {
					log.Debug().Msg(err.Error())
					wg.Done()
					return
				}
				detectedRuntimeChannel <- containerRuntime{Runtime: runtime, Endpoint: endPoint, Connected: connected}
				if connected {
					log.Info().Msgf("connected successfully to endpoint: %s", endPoint)
				}
			}(runtime, endPoint)
		}
	}

	go func() {
		for detectedRuntime := range detectedRuntimeChannel {
			detectedRuntimes = append(detectedRuntimes, detectedRuntime)
			if detectedRuntime.Connected {
				connectedRuntimes = append(connectedRuntimes, detectedRuntime)
			}
			wg.Done()
		}
	}()

	wg.Wait()
	if len(connectedRuntimes) == 0 {
		if len(detectedRuntimes) == 0 {
			return "", "", nil
		} else {
			log.Info().Msg("No running runtimes, selecting first detected runtime")
			return detectedRuntimes[0].Runtime, detectedRuntimes[0].Endpoint, nil
		}
	}
	return connectedRuntimes[0].Runtime, connectedRuntimes[0].Endpoint, nil
}

// AutoDetectRuntime auto detects the underlying container runtime like docker, containerd
func AutoDetectRuntime() (string, string, error) {
	runtime, endpoint, err := getContainerRuntime()
	if err != nil {
		return "", "", err
	}
	if runtime == "" {
		return "", "", errors.New("could not detect container runtime")
	}
	log.Info().Msgf("container runtime detected: %s\n", runtime)
	return runtime, endpoint, nil
}

func isDockerRunning(host string) (bool, error) {
	dockerCli, err := dockerClient.NewClientWithOpts(dockerClient.WithAPIVersionNegotiation(), dockerClient.WithHost(host), dockerClient.WithTimeout(utils.Timeout))
	if err != nil {
		return false, errors.Wrapf(err, " :error creating docker client")
	}
	defer dockerCli.Close()
	containers, err := dockerCli.ContainerList(context.Background(),
		container.ListOptions{
			All: true, Size: false,
		})
	if err != nil {
		return false, errors.Wrapf(err, " :error creating docker client")
	}

	return len(containers) > 0, nil
}

func isPodmanRunning(host string) (bool, error) {
	op, err := utils.RunCommand(exec.Command("podman", "--remote", "--url", host, "ps"), "podman ps:")
	if err != nil {
		log.Warn().Msg(err.Error())
		return false, err
	}
	return len(strings.Split(strings.TrimSpace(op.String()), "\n")) > 1, nil
}

func isContainerdRunning(host string) (bool, error) {
	clientd, err := client.New(strings.Replace(host, "unix://", "", 1))
	if err != nil {
		return false, errors.Wrapf(err, " :error creating containerd client")
	}
	defer clientd.Close()
	namespace_store := clientd.NamespaceService()

	list, err := namespace_store.List(context.Background())
	if err != nil {
		return false, errors.Wrapf(err, " :error creating containerd client")
	}
	for _, l := range list {

		namespace := namespaces.WithNamespace(context.Background(), l)

		containers, err := clientd.Containers(namespace)
		if err != nil {
			return false, errors.Wrapf(err, " :error creating containerd client")
		}

		if len(containers) > 0 {
			return true, nil
		}
	}

	return false, nil

}

func isCRIORunning(host string) (bool, error) {
	cmd := exec.Command("crictl", "--runtime-endpoint", host, "ps", "-q")
	log.Debug().Msgf("command: %s", cmd.String())
	output, err := cmd.Output()
	if err != nil {
		log.Error().Msgf("%s, error: %s", output, err)
		return false, err
	}
	return true, nil
}

// NewRuntime Auto detect and returns the runtime available for the current system
func NewRuntime() (Runtime, error) {

	runtime, endpoint, err := AutoDetectRuntime()
	if err != nil {
		return nil, err
	}

	if runtime == utils.DOCKER {
		return docker.New(endpoint), nil
	} else if runtime == utils.CONTAINERD {
		return selfContainerd.New(endpoint), nil
	} else if runtime == utils.CRIO {
		return crio.New(endpoint), nil
	} else if runtime == utils.PODMAN {
		return selfPodman.New(endpoint), nil
	}

	return nil, errors.New("Unknown runtime")
}
