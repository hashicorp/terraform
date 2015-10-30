package google


import (
	"time"
	"math/rand"
)

func genRandInt() int {
	return rand.New(rand.NewSource(time.Now().UnixNano())).Int()
}
