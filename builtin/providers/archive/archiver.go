package archive

import (
	"fmt"
	"os"
)

type Archiver interface {
	ArchiveContent(content []byte, infilename string) error
	ArchiveFile(infilename string) error
	ArchiveDir(indirname string) error
	ArchiveMultiple(content map[string][]byte) error
}

type ArchiverBuilder func(filepath string) Archiver

var archiverBuilders = map[string]ArchiverBuilder{
	"zip": NewZipArchiver,
}

func getArchiver(archiveType string, filepath string) Archiver {
	if builder, ok := archiverBuilders[archiveType]; ok {
		return builder(filepath)
	}
	return nil
}

func assertValidFile(infilename string) (os.FileInfo, error) {
	fi, err := os.Stat(infilename)
	if err != nil && os.IsNotExist(err) {
		return fi, fmt.Errorf("could not archive missing file: %s", infilename)
	}
	return fi, err
}

func assertValidDir(indirname string) (os.FileInfo, error) {
	fi, err := os.Stat(indirname)
	if err != nil {
		if os.IsNotExist(err) {
			return fi, fmt.Errorf("could not archive missing directory: %s", indirname)
		}
		return fi, err
	}
	if !fi.IsDir() {
		return fi, fmt.Errorf("could not archive directory that is a file: %s", indirname)
	}
	return fi, nil
}
