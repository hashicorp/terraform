package log

import (
	"fmt"
	"sync"
)

type HistoryRecorder struct {
	lock    *sync.Mutex
	records []string
}

func NewHistoryRecorder() *HistoryRecorder {
	return &HistoryRecorder{
		lock:    &sync.Mutex{},
		records: []string{},
	}
}

func (ml *HistoryRecorder) History() []string {
	return ml.records
}

func (ml *HistoryRecorder) Record(args ...interface{}) {
	ml.lock.Lock()
	defer ml.lock.Unlock()
	ml.records = append(ml.records, fmt.Sprint(args...))
}

func (ml *HistoryRecorder) Recordf(fmtString string, args ...interface{}) {
	ml.lock.Lock()
	defer ml.lock.Unlock()
	ml.records = append(ml.records, fmt.Sprintf(fmtString, args...))
}
