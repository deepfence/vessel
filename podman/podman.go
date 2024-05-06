package podman

import (
	"bufio"
	"bytes"
	"errors"
	"os/exec"
	"strings"

	"github.com/deepfence/vessel/utils"
	"github.com/sirupsen/logrus"
)

// New instantiates a new Podman runtime object
func New(endpoint string) *Podman {
	return &Podman{
		socketPath: endpoint,
	}
}

// GetSocket is socket getter
func (d Podman) GetSocket() string {
	return d.socketPath
}

// ExtractImage creates the tarball out of image and extracts it
func (d Podman) ExtractImage(imageID, imageName, path string) error {
	var stderr bytes.Buffer
	save := exec.Command("podman", "--remote", "--url", d.socketPath, "save", imageID)
	save.Stderr = &stderr
	extract := exec.Command("tar", "xf", "-", "--warning=none", "-C"+path)
	extract.Stderr = &stderr
	pipe, err := extract.StdinPipe()
	if err != nil {
		return err
	}
	save.Stdout = pipe

	err = extract.Start()
	if err != nil {
		return errors.New(stderr.String())
	}
	err = save.Run()
	if err != nil {
		return errors.New(stderr.String())
	}
	err = pipe.Close()
	if err != nil {
		return err
	}
	err = extract.Wait()
	if err != nil {
		return errors.New(stderr.String())
	}
	return nil
}

// GetImageID returns the image id
func (d Podman) GetImageID(imageName string) ([]byte, error) {
	return exec.Command("podman", "--remote", "--url", d.socketPath, "images", "-q", "--no-trunc", imageName).Output()
}

// Save just saves image using -o flag
func (d Podman) Save(imageName, outputParam string) ([]byte, error) {
	return exec.Command("podman", "--remote", "--url", d.socketPath, "save", imageName, "-o", outputParam).Output()
}

// ExtractFileSystem Extract the file system from tar of an image by creating a temporary dormant container instance
func (d Podman) ExtractFileSystem(imageTarPath string, outputTarPath string, imageName string) error {
	imageMsg, err := utils.RunCommand(exec.Command("podman", "--remote", "--url", d.socketPath, "load", "-i", imageTarPath), "podman load: "+imageTarPath)
	if err != nil {
		return err
	}
	var imageId = ""
	scanner := bufio.NewScanner(imageMsg)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Loaded image") {
			splits := strings.SplitAfterN(line, ":", 2)
			if len(splits) > 1 {
				imageId = strings.TrimSpace(splits[1])
				break
			}
		}
	}
	if imageId == "" {
		return errors.New("image not found from podman load with output: " + imageMsg.String())
	}
	containerOutput, err := utils.RunCommand(exec.Command("podman", "--remote", "--url", d.socketPath, "create", imageId), "podman create: "+imageId)
	if err != nil {
		return err
	}
	containerId := strings.TrimSpace(containerOutput.String())
	_, err = utils.RunCommand(exec.Command("podman", "--remote", "--url", d.socketPath, "export", strings.TrimSpace(containerId), "-o", outputTarPath), "podman export: "+string(containerId))
	if err != nil {
		return err
	}
	_, err = utils.RunCommand(exec.Command("podman", "--remote", "--url", d.socketPath, "container", "rm", containerId), "delete container:"+containerId)
	if err != nil {
		logrus.Warn(err.Error())
	}
	_, err = utils.RunCommand(exec.Command("podman", "--remote", "--url", d.socketPath, "image", "rm", imageId), "delete image:"+imageId)
	if err != nil {
		logrus.Warn(err.Error())
	}
	return nil
}

// ExtractFileSystemContainer Extract the file system of an existing container to tar
func (d Podman) ExtractFileSystemContainer(containerId string, namespace string, outputTarPath string) error {
	cmd := exec.Command("podman", "--remote", "--url", d.socketPath, "export", strings.TrimSpace(containerId), "-o", outputTarPath)
	_, err := utils.RunCommand(cmd, "podman export: "+string(containerId))
	if err != nil {
		return err
	}
	return nil
}

// ExtractFileSystemContainer Extract the file system of an existing container to tar
func (d Podman) GetFileSystemPathsForContainer(containerId string, namespace string) ([]byte, error) {
	return exec.Command("podman", "--remote", "--url", d.socketPath, "inspect", strings.TrimSpace(containerId), "|", "jq", "-r", "'map([.Name, .GraphDriver.Data.MergedDir]) | .[] | \"\\(.[0])\t\\(.[1])\"'").Output()
}
