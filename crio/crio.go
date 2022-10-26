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

// GetSocket is socket getter
func (c CRIO) GetSocket() string {
	return c.socketPath
}

// ExtractImage will create the tarball from the CRIO image, extracts into dir
// and then skopeo is used to migrate oci layers using the dir to docker v1 layer spec format tar
// and again extracts back to dir
// example:
// skopeo copy oci:///home/ubuntu/img/docker/threatmapper_containerd-dir \
// docker-archive:/home/ubuntu/img/docker/threatmapper_containerd.tar
func (c CRIO) ExtractImage(imageID, imageName, path string) error {
	cmd := exec.Command("podman", "save", "--events-backend", "file",
		"--format", "docker-dir", "--output", path, imageName)
	logrus.Info(cmd.String())
	if _, err := cmd.Output(); err != nil {
		return err
	}
	return nil
}

// GetImageID returns the image id
func (c CRIO) GetImageID(imageName string) ([]byte, error) {
	cmd := exec.Command("podman", "inspect", imageName,
		"--type", "image", "--format", "{{ .ID }}")
	logrus.Info(cmd.String())
	return cmd.Output()
}

// Save just saves image using -o flag
func (c CRIO) Save(imageName, outputParam string) ([]byte, error) {
	cmd := exec.Command("podman", "save", "--events-backend", "file",
		"--format", "docker-archive", "--output", outputParam, imageName)
	logrus.Info(cmd.String())
	return cmd.Output()
}

// ExtractFileSystem Extract the file system from tar of an image by creating a temporary dormant container instance
func (c CRIO) ExtractFileSystem(imageTarPath string, outputTarPath string, imageName string, socketPath string) error {
	return errors.New("function not implemented for cri-o")
}

// ExtractFileSystemContainer Extract the file system of an existing container to tar
func (c CRIO) ExtractFileSystemContainer(containerId string, namespace string, outputTarPath string, socketPath string) error {
	cmd := exec.Command(
		"crictl",
		"inspect",
		"--runtime-endpoint", c.socketPath,
		"--output", "go-template ",
		"--template ", "{{ .info.runtimeSpec.root.path }}", containerId)
	logrus.Infof("command: %s", cmd.String())
	rootpath, err := cmd.Output()
	if err != nil {
		return err
	}
	logrus.Infof("rootPath: %s", rootpath)

	return errors.New("function not implemented for cri-o")
}
