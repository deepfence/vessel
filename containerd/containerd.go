package containerd

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path"
)

// New instantiates a new Containerd runtime object
func New() *Containerd {
	return &Containerd{
		socketPath: "unix:///run/containerd/containerd.sock",
	}
}

// ExtractImage will create the tarball from the containerd image, extracts into dir
// and then skopeo is used to migrate oci layers using the dir to docker v1 layer spec format tar
// and again extracts back to dir
// example:
// skopeo copy oci:///home/ubuntu/img/docker/threatmapper_containerd-dir \
// docker-archive:/home/ubuntu/img/docker/threatmapper_containerd.tar
func (c Containerd) ExtractImage(imageID, imageName, path string) error {
	var stderr bytes.Buffer
	save := exec.Command("/usr/local/bin/nerdctl", "save", imageName)
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

	err = migrateOCIToDockerV1(path, imageID, "")
	if err != nil {
		return err
	}
	return nil
}

// GetImageID returns the image id
func (c Containerd) GetImageID(imageName string) ([]byte, error) {
	return exec.Command("/usr/local/bin/nerdctl", "images", "-q", "--no-trunc", imageName).Output()
}

// Save just saves image using -o flag
func (c Containerd) Save(imageName, outputParam string) ([]byte, error) {
	return exec.Command("/usr/local/bin/nerdctl", "-n", "k8s.io", "save", "-o", outputParam, imageName).Output()
}

// migrateOCIToDockerV1 migrates OCI image to Docker v1 image tarball
func migrateOCIToDockerV1(path, imageID, tarFilePath string) error {
	if tarFilePath == "" {
		tarFilePath = path + imageID + ".tar"
	}
	sourceDir := "oci://" + path
	destinationTar := "docker-archive:" + tarFilePath
	var stderr bytes.Buffer

	// skopeo will convert oci dir into docker v1 tarball
	skopeoCopy := exec.Command("/usr/bin/skopeo", "copy", sourceDir, destinationTar)
	skopeoCopy.Stderr = &stderr
	err := skopeoCopy.Run()
	if err != nil {
		return fmt.Errorf("failed to migrate OCI to Docker image: %v", stderr)
	}

	// untar the docker archive
	tarxf := exec.Command("tar", "xf", tarFilePath, "--warning=none", "-C"+path)
	tarxf.Stderr = &stderr
	err = tarxf.Run()
	if err != nil {
		return fmt.Errorf("failed to migrate OCI to Docker image: %v", err)
	}

	// delete docker tar, not required
	removeTar := exec.Command("rm", tarFilePath)
	removeTar.Stderr = &stderr
	err = removeTar.Run()
	if err != nil {
		return fmt.Errorf("failed to delete generated docker-archive: %v", err)
	}

	return nil
}

// fileuploader specific
func MigrateOCITarToDockerV1Tar(dir, tarName string) error {
	fmt.Println("migrating image ...")
	var stderr bytes.Buffer
	tarPath := path.Join(dir, tarName)
	_, err := exec.Command("tar", "xf", tarPath, "--warning=none", "-C"+dir).Output()
	if err != nil {
		return fmt.Errorf("failed to migrate OCI to Docker image: failed to untar: %v", err)
	}

	// delete docker tar, not required
	removeTar := exec.Command("rm", tarPath)
	removeTar.Stderr = &stderr
	err = removeTar.Run()
	if err != nil {
		return fmt.Errorf("failed to delete generated docker-archive: %v", err)
	}

	//migrate now
	sourceDir := "oci://" + dir
	destinationTar := "docker-archive:" + tarPath
	// var stderr bytes.Buffer

	// skopeo will convert oci dir into docker v1 tarball
	skopeoCopy := exec.Command("/usr/bin/skopeo", "copy", sourceDir, destinationTar)
	skopeoCopy.Stderr = &stderr
	err = skopeoCopy.Run()
	if err != nil {
		return fmt.Errorf("failed to migrate OCI to Docker image: %v", stderr)
	}
	return nil

}
