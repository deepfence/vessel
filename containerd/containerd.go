package containerd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/deepfence/vessel/constants"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/containerd/containerd"
	containerdApi "github.com/containerd/containerd"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/images/archive"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
)

// New instantiates a new Containerd runtime object
func New(host string) *Containerd {
	return &Containerd{
		socketPath: host,
		namespaces: getNamespaces(host),
	}
}

func getNamespaces(host string) []string {
	clientd, err := containerd.New(strings.Replace(host, "unix://", "", 1))
	if err != nil {
		return nil
	}
	defer clientd.Close()
	namespace_store := clientd.NamespaceService()

	list, err := namespace_store.List(context.Background())
	if err != nil {
		return nil
	}
	return list
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
	save := exec.Command("nerdctl", "save", imageName, "--address", c.socketPath)
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
	return exec.Command("nerdctl", "images", "-q", "--no-trunc", imageName, "--address", c.socketPath).Output()
}

// Save just saves image using -o flag
func (c Containerd) Save(imageName, outputParam string) ([]byte, error) {
	nerrors := []error{}
	for _, ns := range c.namespaces {
		res, err := exec.Command("nerdctl", "-n", ns, "save", "--address", c.socketPath, "-o", outputParam, imageName).Output()
		if err == nil {
			return res, nil
		}
		nerrors = append(nerrors, fmt.Errorf("namespace: %v, err: %v\n", ns, err))
	}
	return nil, fmt.Errorf("Save failed. errors:\n%v", nerrors)
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
func (c Containerd) ExtractFileSystem(imageTarPath string, outputTarPath string, imageName string, socketPath string) error {
	// create a new client connected to the default socket path for containerd
	client, err := containerdApi.New(strings.Replace(socketPath, "unix://", "", 1))
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
	imgs, err := client.Import(ctx, reader,
		containerdApi.WithSkipDigestRef(func(name string) bool { return name != "" }),
		containerdApi.WithDigestRef(archive.DigestTranslator(imageName)))
	if err != nil {
		fmt.Println("Error while Importing image")
		return err
	}
	if len(imgs) == 0 {
		fmt.Printf("No images imported, imageTarPath: %s, outputTarPath: %s, imageName: %s \n", imageTarPath, outputTarPath, imageName)
		return errors.New("image not imported from: " + imageTarPath)
	}
	image, err := client.GetImage(ctx, imgs[0].Name)
	if err != nil {
		fmt.Println("Error while getting image from client")
		return err
	}
	rand.Seed(time.Now().UnixNano())
	containerName := "temp" + fmt.Sprint(rand.Intn(9999))
	err = image.Unpack(ctx, "")
	if err != nil {
		fmt.Println("Error while unpacking image")
		return err
	}
	container, err := client.NewContainer(
		ctx,
		containerName,
		containerdApi.WithImage(image),
		containerdApi.WithNewSnapshot("temp"+fmt.Sprint(rand.Intn(9999)), image),
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
		fmt.Println("Error while creating temp target dir", target, err.Error())
		return err
	}
	_, err = exec.Command("bash", "-c", fmt.Sprintf("mount -t %s %s %s -o %s\n", mounts[0].Type, mounts[0].Source, target, strings.Join(mounts[0].Options, ","))).Output()
	if err != nil {
		fmt.Println("Error while mounting image on temp target dir")
		return err
	}
	_, err = exec.Command("tar", "-czvf", outputTarPath, "-C", target, ".").Output()
	if err != nil {
		fmt.Println("Error while packing tar")
		return err
	}
	exec.Command("umount", target).Output()
	exec.Command("rm", "-rf", target).Output()
	container.Delete(ctx, containerdApi.WithSnapshotCleanup)
	client.ImageService().Delete(ctx, imgs[0].Name, images.SynchronousDelete())
	return nil
}

// ExtractFileSystemContainer Extract the file system of an existing container to tar
func (c Containerd) ExtractFileSystemContainer(containerId string, namespace string, outputTarPath string, socketPath string) error {
	// create a new client connected to the default socket path for containerd
	client, err := containerdApi.New(strings.Replace(socketPath, "unix://", "", 1))
	if err != nil {
		return err
	}
	defer client.Close()
	// create a new context with namespace
	if len(namespace) == 0 {
		namespace = constants.CONTAINERD_K8S_NS
	}
	ctx := namespaces.WithNamespace(context.Background(), namespace)
	container, err := client.LoadContainer(ctx, containerId)
	if err != nil {
		fmt.Println("Error while getting container")
		return err
	}
	info, _ := container.Info(ctx)
	snapshotter := client.SnapshotService(info.Snapshotter)
	mounts, err := snapshotter.Mounts(ctx, info.SnapshotKey)
	target := strings.Replace(outputTarPath, ".tar", "", 1) + containerId
	_, err = exec.Command("mkdir", target).Output()
	if err != nil {
		fmt.Println("Error while creating temp target dir", target,  err.Error())
		return err
	}
	_, err = exec.Command("bash", "-c", fmt.Sprintf("mount -t %s %s %s -o %s\n", mounts[0].Type, mounts[0].Source, target, strings.Join(mounts[0].Options, ","))).Output()
	if err != nil {
		fmt.Println("Error while mounting image on temp target dir")
		return err
	}
	_, err = exec.Command("tar", "-czvf", outputTarPath, "-C", target, ".").Output()
	if err != nil {
		fmt.Println("Error while packing tar")
		return err
	}
	exec.Command("umount", target).Output()
	return nil
}
