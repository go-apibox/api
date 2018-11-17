// 增加增加RequestId处理

package api

import (
	"net/http"
	"time"

	"github.com/go-apibox/cache"
	"github.com/streadway/simpleuuid"
)

// RequestIdMaker is a middleware handler that auto generate the request id and append to response header.
type RequestIdMaker struct {
	reqIdCache *cache.Cache
}

// NewRequestIdMaker returns a new RequestIdMaker instance
func NewRequestIdMaker() *RequestIdMaker {
	return &RequestIdMaker{cache.NewCache(time.Duration(10) * time.Second)}
}

func (m *RequestIdMaker) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	var reqId string
	for {
		uuid, err := simpleuuid.NewTime(time.Now())
		if err != nil {
			continue
		}
		reqId = uuid.String()
		_, exists := m.reqIdCache.Get(reqId)
		if !exists {
			break
		}
	}
	rw.Header().Set("X-Request-Id", reqId)

	next(rw, r)
}
