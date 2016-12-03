package net

import (
	"io"
	"os"
	"sync"
	"time"

	"code.cloudfoundry.org/cli/cf/formatters"
	"code.cloudfoundry.org/cli/cf/terminal"
)

type ProgressReader struct {
	ioReadSeeker   io.ReadSeeker
	bytesRead      int64
	total          int64
	quit           chan bool
	ui             terminal.UI
	outputInterval time.Duration
	mutex          sync.RWMutex
}

func NewProgressReader(readSeeker io.ReadSeeker, ui terminal.UI, outputInterval time.Duration) *ProgressReader {
	return &ProgressReader{
		ioReadSeeker:   readSeeker,
		ui:             ui,
		outputInterval: outputInterval,
		mutex:          sync.RWMutex{},
	}
}

func (progressReader *ProgressReader) Read(p []byte) (int, error) {
	if progressReader.ioReadSeeker == nil {
		return 0, os.ErrInvalid
	}

	n, err := progressReader.ioReadSeeker.Read(p)

	if progressReader.total > int64(0) {
		if n > 0 {
			if progressReader.quit == nil {
				progressReader.quit = make(chan bool)
				go progressReader.printProgress(progressReader.quit)
			}

			progressReader.mutex.Lock()
			progressReader.bytesRead += int64(n)
			progressReader.mutex.Unlock()

			if progressReader.total == progressReader.bytesRead {
				progressReader.quit <- true
				return n, err
			}
		}
	}

	return n, err
}

func (progressReader *ProgressReader) Seek(offset int64, whence int) (int64, error) {
	return progressReader.ioReadSeeker.Seek(offset, whence)
}

func (progressReader *ProgressReader) printProgress(quit chan bool) {
	timer := time.NewTicker(progressReader.outputInterval)

	for {
		select {
		case <-quit:
			//The spaces are there to ensure we overwrite the entire line
			//before using the terminal printer to output Done Uploading
			progressReader.ui.PrintCapturingNoOutput("\r                             ")
			progressReader.ui.Say("\rDone uploading")
			return
		case <-timer.C:
			progressReader.mutex.RLock()
			progressReader.ui.PrintCapturingNoOutput("\r%s uploaded...", formatters.ByteSize(progressReader.bytesRead))
			progressReader.mutex.RUnlock()
		}
	}
}

func (progressReader *ProgressReader) SetTotalSize(size int64) {
	progressReader.total = size
}
