package main

import (
	"github.com/deepfence/vessel"
	"github.com/rs/zerolog/log"
)

func main() {
	// check if image exists
	imageName := "nginx:latest"
	runtime, err := vessel.NewRuntime()
	if err != nil {
		log.Error().Err(err).Msg("failed to create runtime")
		return
	}

	if runtime.ImageExists(imageName) {
		log.Info().Msgf("Image %s exists", imageName)
	} else {
		log.Info().Msgf("Image %s does not exist", imageName)
	}
}
