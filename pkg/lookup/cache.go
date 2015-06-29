package lookup

import (
	"time"

	"github.com/youtube/vitess/go/cache"
)

const (
	cacheCapacity = 1024 * 1024 // 1MB
	cacheTTL      = 1           // 1 second
)

var (
	resolveCache = cache.NewLRUCache(cacheCapacity)
)

type cacheValue struct {
	Value     []string
	CreatedAt int64
}

func (cv *cacheValue) Size() int {
	var size int
	for _, s := range cv.Value {
		size += len(s)
	}
	return size
}

func (cv *cacheValue) Expired() bool {
	return (time.Now().Unix() - cv.CreatedAt) > cacheTTL
}
