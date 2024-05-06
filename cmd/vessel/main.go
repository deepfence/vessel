package main

import (
	"os"

	"github.com/deepfence/vessel"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

var activeRuntime string
var activeEndpoint string

func init() {
	var err error
	// Auto-detect underlying container runtime
	activeRuntime, activeEndpoint, err = vessel.AutoDetectRuntime()
	if err != nil {
		logrus.Error(err)
		return
	}
	// create .env
	_, err = os.Create(".env")
	if err != nil {
		logrus.Error(err)
		return
	}
}

// use godot package to load/read the .env file and
// return the value of the key
func setDotEnvVariable(envars map[string]string) error {
	// load .env file
	err := godotenv.Load(".env")
	if err != nil {
		return err
	}
	return godotenv.Write(envars, "./.env")
}

func main() {
	if activeRuntime != "" {
		envVars := map[string]string{
			"CONTAINER_RUNTIME": activeRuntime,
			"CRI_ENDPOINT":      activeEndpoint,
		}
		err := setDotEnvVariable(envVars)
		if err != nil {
			logrus.Error(err)
		}
	}
}
