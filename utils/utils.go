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

	"github.com/sirupsen/logrus"
)

func CheckTarFileValid(tarFilePath string) bool {
	file, err := os.Open(tarFilePath)
	if err != nil {
		return false
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return false
	}
	tr := tar.NewReader(gzipReader)
	_, err = tr.Next()
	if err != nil {
		if err == io.EOF {
			return true
		}
		logrus.Error(err)
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
		logrus.Errorf("cmd: %s", cmd.String())
		logrus.Error(errorOnRun)
		return nil, errors.New(operation + fmt.Sprint(errorOnRun) + ": " + stderr.String())
	}
	return &out, nil
}
