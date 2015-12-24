package google

import (
	"math/rand"
	"time"
)

func genRandInt() int {
	return rand.New(rand.NewSource(time.Now().UnixNano())).Int()
}
