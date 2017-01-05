package logs

import (
	"sort"
	"sync"

	"github.com/cloudfoundry/sonde-go/events"
)

type NoaaMessageQueue struct {
	messages []*events.LogMessage
	mutex    sync.Mutex
}

func NewNoaaMessageQueue() *NoaaMessageQueue {
	return &NoaaMessageQueue{}
}

func (pq *NoaaMessageQueue) PushMessage(message *events.LogMessage) {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	pq.messages = append(pq.messages, message)
}

// implement sort interface so we can sort messages as we receive them in PushMessage
func (pq *NoaaMessageQueue) Less(i, j int) bool {
	return *pq.messages[i].Timestamp < *pq.messages[j].Timestamp
}

func (pq *NoaaMessageQueue) Swap(i, j int) {
	pq.messages[i], pq.messages[j] = pq.messages[j], pq.messages[i]
}

func (pq *NoaaMessageQueue) Len() int {
	return len(pq.messages)
}

func (pq *NoaaMessageQueue) EnumerateAndClear(onMessage func(*events.LogMessage)) {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	sort.Stable(pq)

	for _, x := range pq.messages {
		onMessage(x)
	}

	pq.messages = []*events.LogMessage{}
}
