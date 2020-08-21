package iochan

import (
	"bufio"
	"io"
)

// DelimReader takes an io.Reader and produces the contents of the reader
// on the returned channel. The contents on the channel will be returned
// on boundaries specified by the delim parameter, and will include this
// delimiter.
//
// If an error occurs while reading from the reader, the reading will end.
//
// In the case of an EOF or error, the channel will be closed.
//
// This must only be called once for any individual reader. The behavior is
// unknown and will be unexpected if this is called multiple times with the
// same reader.
func DelimReader(r io.Reader, delim byte) <-chan string {
	ch := make(chan string)

	go func() {
		buf := bufio.NewReader(r)

		for {
			line, err := buf.ReadString(delim)
			if line != "" {
				ch <- line
			}

			if err != nil {
				break
			}
		}

		close(ch)
	}()

	return ch
}
