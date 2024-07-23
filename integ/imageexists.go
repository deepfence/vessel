package main

import (
	"github.com/deepfence/vessel"
	"github.com/sirupsen/logrus"
)

func main() {
	// check if image exists
	imageName := "nginx:latest"
	runtime, err := vessel.NewRuntime()
	if err != nil {
		logrus.Error(err)
		return
	}

	if runtime.ImageExists(imageName) {
		logrus.Infof("Image %s exists", imageName)
	} else {
		logrus.Infof("Image %s does not exist", imageName)
	}
}
