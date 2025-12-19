package utils

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/rs/zerolog/log"
)

func CheckTarGzFileValid(tarFilePath string) bool {
	file, err := os.Open(tarFilePath)
	if err != nil {
		return false
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return false
	}
	defer gzipReader.Close()
	tr := tar.NewReader(gzipReader)
	_, err = tr.Next()
	if err != nil {
		if err == io.EOF {
			return true
		}
		log.Error().Err(err).Msg("tar validation error")
		return false
	}
	return true
}

func CheckTarFileValid(tarFilePath string) bool {
	file, err := os.Open(tarFilePath)
	if err != nil {
		return false
	}
	defer file.Close()
	tr := tar.NewReader(file)
	_, err = tr.Next()
	if err != nil {
		if err == io.EOF {
			return true
		}
		log.Error().Err(err).Msg("tar validation error")
		return false
	}
	return true
}

// RunCommand operation is prepended to error message in case of error: optional
func RunCommand(cmd *exec.Cmd, operation string) (*bytes.Buffer, error) {
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	errorOnRun := cmd.Run()
	if errorOnRun != nil {
		log.Error().Msgf("cmd: %s", cmd.String())
		log.Error().Err(errorOnRun).Msg("command execution failed")
		return nil, errors.New(operation + fmt.Sprint(errorOnRun) + ": " + stderr.String())
	}
	return &out, nil
}
