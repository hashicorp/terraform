package logger

import (
	"errors"
	"io"
	"log"
	"os"
	"time"
)

type asyncWriter struct {
	w       io.Writer
	queue   chan []byte
	flushCh chan chan<- struct{}
}

func newAsyncWriter(w io.Writer) *asyncWriter {
	aw := &asyncWriter{
		w:       w,
		queue:   make(chan []byte, 512),
		flushCh: make(chan chan<- struct{}),
	}
	go aw.doWork()
	return aw
}

func (w *asyncWriter) Flush() error {
	ch := make(chan struct{})
	w.flushCh <- ch
	<-ch
	return nil
}

func (w *asyncWriter) Write(p []byte) (int, error) {
	b := make([]byte, len(p))
	copy(b, p)
	w.queue <- b
	return len(p), nil
}

func (w *asyncWriter) doFlush() {
	n := len(w.queue)
	for i := 0; i < n; i++ {
		select {
		case p := <-w.queue:
			w.w.Write(p)
		default:
		}
	}
}

func (w *asyncWriter) doWork() {
	for {
		select {
		case c := <-w.flushCh:
			w.doFlush()
			close(c)
		case p := <-w.queue:
			w.w.Write(p)
		}
	}
}

type asyncLogger struct {
	out *asyncWriter
	err *asyncWriter
	log *logger
}

func (l *asyncLogger) Flush() error {
	l.out.Flush()
	l.err.Flush()
	return nil
}

func (l *asyncLogger) FlushTimeout(d time.Duration) error {
	ch := make(chan error, 1)
	go func() {
		ch <- l.Flush()
	}()
	select {
	case err := <-ch:
		return err
	case <-time.After(d):
		return errors.New("logger: flush timed out after " + d.String())
	}
}

func NewAsyncWriterLogger(level LogLevel, out, err io.Writer) Logger {
	wout := newAsyncWriter(out)
	werr := newAsyncWriter(err)
	return &asyncLogger{
		out: wout,
		err: werr,
		log: &logger{
			level: level,
			out:   log.New(wout, "", log.LstdFlags),
			err:   log.New(werr, "", log.LstdFlags),
		},
	}
}

func (l *asyncLogger) Debug(tag, msg string, args ...interface{}) {
	l.log.Debug(tag, msg, args...)
}

func (l *asyncLogger) DebugWithDetails(tag, msg string, args ...interface{}) {
	l.log.DebugWithDetails(tag, msg, args...)
}

func (l *asyncLogger) Info(tag, msg string, args ...interface{}) {
	l.log.Info(tag, msg, args...)
}

func (l *asyncLogger) Warn(tag, msg string, args ...interface{}) {
	l.log.Warn(tag, msg, args...)
}

func (l *asyncLogger) Error(tag, msg string, args ...interface{}) {
	l.log.Error(tag, msg, args...)
}

func (l *asyncLogger) ErrorWithDetails(tag, msg string, args ...interface{}) {
	l.log.ErrorWithDetails(tag, msg, args...)
}

func (l *asyncLogger) HandlePanic(tag string) {
	if l.log.recoverPanic(tag) {
		l.FlushTimeout(time.Second * 30)
		os.Exit(2)
	}
}

func (l *asyncLogger) ToggleForcedDebug() {
	l.log.ToggleForcedDebug()
}
