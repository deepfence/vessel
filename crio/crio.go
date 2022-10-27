package crio

import (
	"errors"
	"os/exec"

	"github.com/sirupsen/logrus"
)

// New instantiates a new CRIO runtime object
func New(host string) *CRIO {
	return &CRIO{
		socketPath: host,
	}
}

func (c CRIO) GetSocket() string {
	return c.socketPath
}

func (c CRIO) ExtractImage(imageID, imageName, path string) error {
	cmd := exec.Command("podman", "save", "--events-backend", "file",
		"--format", "docker-dir", "--output", path, imageName)
	logrus.Infof("extract image command: %s", cmd.String())
	if _, err := cmd.Output(); err != nil {
		return err
	}
	return nil
}

func (c CRIO) GetImageID(imageName string) ([]byte, error) {
	cmd := exec.Command("podman", "inspect", imageName,
		"--type", "image", "--format", "{{ .ID }}")
	logrus.Infof("get imageID command: %s", cmd.String())
	return cmd.Output()
}

func (c CRIO) Save(imageName, outputParam string) ([]byte, error) {
	cmd := exec.Command("podman", "save", "--events-backend", "file",
		"--format", "docker-archive", "--output", outputParam, imageName)
	logrus.Infof("save image command: %s", cmd.String())
	return cmd.Output()
}

func (c CRIO) ExtractFileSystem(imageTarPath string, outputTarPath string,
	imageName string, socketPath string) error {
	return errors.New("function not implemented for cri-o")
}

func (c CRIO) ExtractFileSystemContainer(containerId string, namespace string,
	outputTarPath string, socketPath string) error {

	// inspect doesnot accept runtime endpoint option
	_, _ = exec.Command(
		"crictl",
		"config",
		"--set", "runtime-endpoint="+c.socketPath).Output()
	// get root path
	cmd := exec.Command(
		"crictl",
		"inspect",
		"--output", "go-template",
		"--template", "\"{{ .info.runtimeSpec.root.path }}\"", containerId)
	logrus.Infof("contaier root path command: %s", cmd.String())
	rootpath, err := cmd.Output()
	if err != nil {
		logrus.Errorf("failed to get container root path error %s", err)
		return err
	}
	logrus.Infof("containerId: %s rootPath: %s", containerId, rootpath)

	if len(rootpath) < 1 {
		logrus.Errorf("container root path is empty for containerID %s", containerId)
		return errors.New("container root path is empty")
	}

	cmd = exec.Command("tar", "-czvf", outputTarPath, "-C", string(rootpath), ".")
	logrus.Infof("tar command: %s", cmd.String())
	if _, err := cmd.Output(); err != nil {
		logrus.Errorf("error while packing tar containerId: %s file: %s path: %s error: %s",
			containerId, outputTarPath, rootpath, err)
		return err
	}

	return nil
}
