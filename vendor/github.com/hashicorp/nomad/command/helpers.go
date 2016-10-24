package command

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	gg "github.com/hashicorp/go-getter"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/jobspec"
	"github.com/hashicorp/nomad/nomad/structs"

	"github.com/ryanuber/columnize"
)

// formatKV takes a set of strings and formats them into properly
// aligned k = v pairs using the columnize library.
func formatKV(in []string) string {
	columnConf := columnize.DefaultConfig()
	columnConf.Empty = "<none>"
	columnConf.Glue = " = "
	return columnize.Format(in, columnConf)
}

// formatList takes a set of strings and formats them into properly
// aligned output, replacing any blank fields with a placeholder
// for awk-ability.
func formatList(in []string) string {
	columnConf := columnize.DefaultConfig()
	columnConf.Empty = "<none>"
	return columnize.Format(in, columnConf)
}

// formatListWithSpaces takes a set of strings and formats them into properly
// aligned output. It should be used sparingly since it doesn't replace empty
// values and hence not awk/sed friendly
func formatListWithSpaces(in []string) string {
	columnConf := columnize.DefaultConfig()
	return columnize.Format(in, columnConf)
}

// Limits the length of the string.
func limit(s string, length int) string {
	if len(s) < length {
		return s
	}

	return s[:length]
}

// formatTime formats the time to string based on RFC822
func formatTime(t time.Time) string {
	return t.Format("01/02/06 15:04:05 MST")
}

// formatUnixNanoTime is a helper for formatting time for output.
func formatUnixNanoTime(nano int64) string {
	t := time.Unix(0, nano)
	return formatTime(t)
}

// formatTimeDifference takes two times and determines their duration difference
// truncating to a passed unit.
// E.g. formatTimeDifference(first=1m22s33ms, second=1m28s55ms, time.Second) -> 6s
func formatTimeDifference(first, second time.Time, d time.Duration) string {
	return second.Truncate(d).Sub(first.Truncate(d)).String()
}

// getLocalNodeID returns the node ID of the local Nomad Client and an error if
// it couldn't be determined or the Agent is not running in Client mode.
func getLocalNodeID(client *api.Client) (string, error) {
	info, err := client.Agent().Self()
	if err != nil {
		return "", fmt.Errorf("Error querying agent info: %s", err)
	}
	var stats map[string]interface{}
	stats, _ = info["stats"]
	clientStats, ok := stats["client"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("Nomad not running in client mode")
	}

	nodeID, ok := clientStats["node_id"].(string)
	if !ok {
		return "", fmt.Errorf("Failed to determine node ID")
	}

	return nodeID, nil
}

// evalFailureStatus returns whether the evaluation has failures and a string to
// display when presenting users with whether there are failures for the eval
func evalFailureStatus(eval *api.Evaluation) (string, bool) {
	if eval == nil {
		return "", false
	}

	hasFailures := len(eval.FailedTGAllocs) != 0
	text := strconv.FormatBool(hasFailures)
	if eval.Status == "blocked" {
		text = "N/A - In Progress"
	}

	return text, hasFailures
}

// LineLimitReader wraps another reader and provides `tail -n` like behavior.
// LineLimitReader buffers up to the searchLimit and returns `-n` number of
// lines. After those lines have been returned, LineLimitReader streams the
// underlying ReadCloser
type LineLimitReader struct {
	io.ReadCloser
	lines       int
	searchLimit int

	timeLimit time.Duration
	lastRead  time.Time

	buffer     *bytes.Buffer
	bufFiled   bool
	foundLines bool
}

// NewLineLimitReader takes the ReadCloser to wrap, the number of lines to find
// searching backwards in the first searchLimit bytes. timeLimit can optionally
// be specified by passing a non-zero duration. When set, the search for the
// last n lines is aborted if no data has been read in the duration. This
// can be used to flush what is had if no extra data is being received. When
// used, the underlying reader must not block forever and must periodically
// unblock even when no data has been read.
func NewLineLimitReader(r io.ReadCloser, lines, searchLimit int, timeLimit time.Duration) *LineLimitReader {
	return &LineLimitReader{
		ReadCloser:  r,
		searchLimit: searchLimit,
		timeLimit:   timeLimit,
		lines:       lines,
		buffer:      bytes.NewBuffer(make([]byte, 0, searchLimit)),
	}
}

func (l *LineLimitReader) Read(p []byte) (n int, err error) {
	// Fill up the buffer so we can find the correct number of lines.
	if !l.bufFiled {
		b := make([]byte, len(p))
		n, err := l.ReadCloser.Read(b)
		if n > 0 {
			if _, err := l.buffer.Write(b[:n]); err != nil {
				return 0, err
			}
		}

		if err != nil {
			if err != io.EOF {
				return 0, err
			}

			l.bufFiled = true
			goto READ
		}

		if l.buffer.Len() >= l.searchLimit {
			l.bufFiled = true
			goto READ
		}

		if l.timeLimit.Nanoseconds() > 0 {
			if l.lastRead.IsZero() {
				l.lastRead = time.Now()
				return 0, nil
			}

			now := time.Now()
			if n == 0 {
				// We hit the limit
				if l.lastRead.Add(l.timeLimit).Before(now) {
					l.bufFiled = true
					goto READ
				} else {
					return 0, nil
				}
			} else {
				l.lastRead = now
			}
		}

		return 0, nil
	}

READ:
	if l.bufFiled && l.buffer.Len() != 0 {
		b := l.buffer.Bytes()

		// Find the lines
		if !l.foundLines {
			found := 0
			i := len(b) - 1
			sep := byte('\n')
			lastIndex := len(b) - 1
			for ; found < l.lines && i >= 0; i-- {
				if b[i] == sep {
					lastIndex = i

					// Skip the first one
					if i != len(b)-1 {
						found++
					}
				}
			}

			// We found them all
			if found == l.lines {
				// Clear the buffer until the last index
				l.buffer.Next(lastIndex + 1)
			}

			l.foundLines = true
		}

		// Read from the buffer
		n := copy(p, l.buffer.Next(len(p)))
		return n, nil
	}

	// Just stream from the underlying reader now
	return l.ReadCloser.Read(p)
}

type JobGetter struct {
	// The fields below can be overwritten for tests
	testStdin io.Reader
}

// StructJob returns the Job struct from jobfile.
func (j *JobGetter) StructJob(jpath string) (*structs.Job, error) {
	var jobfile io.Reader
	switch jpath {
	case "-":
		if j.testStdin != nil {
			jobfile = j.testStdin
		} else {
			jobfile = os.Stdin
		}
	default:
		if len(jpath) == 0 {
			return nil, fmt.Errorf("Error jobfile path has to be specified.")
		}

		job, err := ioutil.TempFile("", "jobfile")
		if err != nil {
			return nil, err
		}
		defer os.Remove(job.Name())

		// Get the pwd
		pwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}

		client := &gg.Client{
			Src: jpath,
			Pwd: pwd,
			Dst: job.Name(),
		}

		if err := client.Get(); err != nil {
			return nil, fmt.Errorf("Error getting jobfile from %q: %v", jpath, err)
		} else {
			file, err := os.Open(job.Name())
			defer file.Close()
			if err != nil {
				return nil, fmt.Errorf("Error opening file %q: %v", jpath, err)
			}
			jobfile = file
		}
	}

	// Parse the JobFile
	jobStruct, err := jobspec.Parse(jobfile)
	if err != nil {
		fmt.Errorf("Error parsing job file from %s: %v", jpath, err)
		return nil, err
	}

	return jobStruct, nil
}
