package containerd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	containerdApi "github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/deepfence/vessel/constants"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

// New instantiates a new Containerd runtime object
func New() *Containerd {
	return &Containerd{
		socketPath: "unix:///run/containerd/containerd.sock",
	}
}

// GetSocket is socket getter
func (c Containerd) GetSocket() string {
	return c.socketPath
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

// ExtractFileSystem Extract the file system from tar of an image by creating a temporary dormant container instance
func (c Containerd) ExtractFileSystem(imageTarPath string, outputTarPath string, imageName string) error {
	// create a new client connected to the default socket path for containerd
	client, err := containerdApi.New(constants.CONTAINERD_SOCKET_ADDRESS)
	if err != nil {
		return err
	}
	defer client.Close()
	// create a new context with an "temp" namespace
	ctx := namespaces.WithNamespace(context.Background(), "temp")
	reader, err := os.Open(imageTarPath)
	if err != nil {
		fmt.Println("Error while opening image")
		return err
	}
	imgs, err := client.Import(ctx, reader)
	if err != nil {
		fmt.Println("Error while Importing image")
		return err
	}
	image, err := client.GetImage(ctx, imgs[0].Name)
	if err != nil {
		fmt.Println("Error while getting image from client")
		return err
	}
	rand.Seed(time.Now().UnixNano())
	containerName := imageName + fmt.Sprint(rand.Intn(9999))
	err = image.Unpack(ctx, "")
	if err != nil {
		fmt.Println("Error while unpacking image")
		return err
	}
	container, err := client.NewContainer(
		ctx,
		containerName,
		containerdApi.WithImage(image),
		containerdApi.WithNewSnapshot(imageName + fmt.Sprint(rand.Intn(9999)), image),
		containerdApi.WithNewSpec(oci.WithImageConfig(image)),
	)
	if err != nil {
		fmt.Println("Error while creating container")
		return err
	}
	info, _ := container.Info(ctx)
	snapshotter := client.SnapshotService(info.Snapshotter)
	mounts, err := snapshotter.Mounts(ctx, info.SnapshotKey)
	target := strings.Replace(outputTarPath, ".tar", "", 1) + containerName
	_, err = exec.Command("mkdir", target).Output()
	if err != nil {
		fmt.Println("Error while creating temp target dir")
		return err
	}
	_, err = exec.Command("bash", "-c",fmt.Sprintf("mount -t %s %s %s -o %s\n", mounts[0].Type, mounts[0].Source, target, strings.Join(mounts[0].Options, ","))).Output()
	if err != nil {
		fmt.Println("Error while mounting image on temp target dir")
		return err
	}
	_, err = exec.Command("tar", "-czvf", outputTarPath, "-C", target, ".").Output()
	if err != nil {
		fmt.Println("Error while packing tar")
		return err
	}
	defer container.Delete(ctx, containerdApi.WithSnapshotCleanup)
	exec.Command("umount", target).Output()
	exec.Command("rm", "-rf", target).Output()
	return nil
}
