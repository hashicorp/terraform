package main

import (
	"io"
	"sync"
)

type synchronizedWriter struct {
	io.Writer
	mutex *sync.Mutex
}

// synchronizedWriters takes a set of writers and returns wrappers that ensure
// that only one write can be outstanding at a time across the whole set.
func synchronizedWriters(targets ...io.Writer) []io.Writer {
	mutex := &sync.Mutex{}
	ret := make([]io.Writer, len(targets))
	for i, target := range targets {
		ret[i] = &synchronizedWriter{
			Writer: target,
			mutex:  mutex,
		}
	}
	return ret
}

func (w *synchronizedWriter) Write(p []byte) (int, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.Writer.Write(p)
}
