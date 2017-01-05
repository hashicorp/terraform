package logs

import (
	"sort"
	"sync"

	"github.com/cloudfoundry/loggregatorlib/logmessage"
)

type LoggregatorMessageQueue struct {
	messages []*logmessage.LogMessage
	mutex    sync.Mutex
}

func NewLoggregatorMessageQueue() *LoggregatorMessageQueue {
	return &LoggregatorMessageQueue{}
}

func (pq *LoggregatorMessageQueue) PushMessage(message *logmessage.LogMessage) {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	pq.messages = append(pq.messages, message)
}

// implement sort interface so we can sort messages as we receive them in PushMessage
func (pq *LoggregatorMessageQueue) Less(i, j int) bool {
	return *pq.messages[i].Timestamp < *pq.messages[j].Timestamp
}

func (pq *LoggregatorMessageQueue) Swap(i, j int) {
	pq.messages[i], pq.messages[j] = pq.messages[j], pq.messages[i]
}

func (pq *LoggregatorMessageQueue) Len() int {
	return len(pq.messages)
}

func (pq *LoggregatorMessageQueue) EnumerateAndClear(onMessage func(*logmessage.LogMessage)) {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	sort.Stable(pq)

	for _, x := range pq.messages {
		onMessage(x)
	}

	pq.messages = []*logmessage.LogMessage{}
}
