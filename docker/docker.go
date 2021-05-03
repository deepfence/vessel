package docker

import (
	"bytes"
	"errors"
	"os/exec"
)

// New instantiates a new Docker runtime object
func New() *Docker {
	return &Docker{
		socketPath: "unix:///var/run/docker.sock",
	}
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
