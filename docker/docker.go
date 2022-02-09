package docker

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// New instantiates a new Docker runtime object
func New() *Docker {
	return &Docker{
		socketPath: "unix:///var/run/docker.sock",
	}
}

// GetSocket is socket getter
func (d Docker) GetSocket() string {
	return d.socketPath
}

// ExtractImage creates the tarball out of image and extracts it
func (d Docker) ExtractImage(imageID, imageName, path string) error {
	var stderr bytes.Buffer
	save := exec.Command("docker", "save", imageID)
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
func (d Docker) GetImageID(imageName string) ([]byte, error) {
	return exec.Command("docker", "images", "-q", "--no-trunc", imageName).Output()
}

// Save just saves image using -o flag
func (d Docker) Save(imageName, outputParam string) ([]byte, error) {
	return exec.Command("docker", "save", imageName, "-o", outputParam).Output()
}

// ExtractFileSystem Extract the file system from tar of an image by creating a temporary dormant container instance
func (d Docker) ExtractFileSystem(imageTarPath string, outputTarPath string, imageName string, socketPath string) error {
	imageMsg, err := runCommand(exec.Command("docker", "load", "-i", imageTarPath), "docker load: " + imageTarPath)
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
		return errors.New("image not found from docker load with output: " + string(imageMsg.Bytes()))
	}
	containerOutput, err := runCommand(exec.Command("docker", "create", imageId), "docker create: " + imageId)
	if err != nil {
		return err
	}
	containerId := strings.TrimSpace(string(containerOutput.Bytes()))
	_, err = runCommand(exec.Command("docker", "export", strings.TrimSpace(containerId), "-o", outputTarPath), "docker export: " + string(containerId))
	if err != nil {
		return err
	}
	_, err = runCommand(exec.Command("docker", "container", "rm", containerId), "delete container:" + containerId)
	if err != nil {
		fmt.Println(err.Error())
	}
	_, err = runCommand(exec.Command("docker", "image", "rm", imageId), "delete image:" + imageId)
	if err != nil {
		fmt.Println(err.Error())
	}
	return nil
}

// ExtractFileSystemContainer Extract the file system of an existing container to tar
func (d Docker) ExtractFileSystemContainer(containerId string, namespace string, outputTarPath string, socketPath string) error {
	_, err := runCommand(exec.Command("docker", "export", strings.TrimSpace(containerId), "-o", outputTarPath), "docker export: " + string(containerId))
	if err != nil {
		return err
	}
	return nil
}

// operation is prepended to error message in case of error: optional
func runCommand(cmd *exec.Cmd, operation string) (*bytes.Buffer, error) {
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	errorOnRun := cmd.Run()
	if errorOnRun != nil {
		return nil, errors.New(operation + fmt.Sprint(errorOnRun) + ": " + stderr.String())
	}
	return &out, nil
}