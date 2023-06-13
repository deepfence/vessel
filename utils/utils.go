package utils

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"

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
