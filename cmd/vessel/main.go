package main

import (
	"log"
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
	containerRuntime, endpoint, err := vessel.AutoDetectRuntime()
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
	activeRuntime = containerRuntime
	activeEndpoint = endpoint
	// create .env
	os.Create(".env")
}

// use godot package to load/read the .env file and
// return the value of the key
func setDotEnvVariable(envars map[string]string) error {
	// load .env file
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	return godotenv.Write(envars, "./.env")
}

func main() {
	if activeRuntime != "" {
		envVars := map[string]string{
			"CONTAINER_RUNTIME": activeRuntime,
			"CRI_ENDPOINT":      activeEndpoint,
		}
		setDotEnvVariable(envVars)
	}
}
