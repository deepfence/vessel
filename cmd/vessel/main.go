package main

import (
	"fmt"
	"log"
	"os"

	"github.com/deepfence/vessel"
	"github.com/joho/godotenv"
)

var activeRuntime string
var ContainerRuntimeInterface vessel.Runtime
var SockPath string
var ContainerRuntime string

func init() {
	var err error
	// Auto-detect underlying container runtime
	ContainerRuntime, SockPath, err = vessel.AutoDetectRuntime()
	if err != nil {
		panic(fmt.Sprint(err))
	}
	activeRuntime = ContainerRuntime
	fmt.Printf("%s detected\n", ContainerRuntime)

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

	err = godotenv.Write(envars, "./.env")

	return err
}

func main() {
	if activeRuntime != "" {
		envars := map[string]string{
			"CONTAINER_RUNTIME": ContainerRuntime,
			"CRI_ENDPOINT":      SockPath,
		}
		setDotEnvVariable(envars)
	}
}
