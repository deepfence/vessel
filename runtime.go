package vessel

// Runtime interface, interfaces all the container runtime methods
type Runtime interface {
	ExtractImage(imageID string, imageName string, path string) error
	GetImageID(imageName string) ([]byte, error)
	Save(imageName, outputParam string) ([]byte, error)
	GetSocket() string
	ExtractFileSystem(imageTarPath string, outputTarPath string, imageName string) error
	ExtractFileSystemContainer(containerId string, namespace string, outputTarPath string) error
}
