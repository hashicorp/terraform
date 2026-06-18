// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build race

package configload

import (
	"fmt"
	"sync"
	"testing"
)

func TestLoaderSourcesConcurrentWithParserWrite(t *testing.T) {
	loader, cleanup := NewLoaderForTests(t)
	defer cleanup()

	const iterations = 2000

	start := make(chan struct{})
	var ready sync.WaitGroup
	ready.Add(2)

	var workers sync.WaitGroup
	workers.Add(2)

	go func() {
		defer workers.Done()
		ready.Done()
		<-start

		for i := 0; i < iterations; i++ {
			_ = loader.Sources()
		}
	}()

	go func() {
		defer workers.Done()
		ready.Done()
		<-start

		for i := 0; i < iterations; i++ {
			loader.Parser().ForceFileSource(
				fmt.Sprintf("testdata/%04d.tf", i),
				[]byte("resource \"test\" \"example\" {}\n"),
			)
		}
	}()

	ready.Wait()
	close(start)
	workers.Wait()
}
