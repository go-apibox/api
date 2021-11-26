// HTTP服务器日志处理

package api

import (
	"net/http"
	"time"

	"github.com/go-apibox/logging"
	"github.com/go-apibox/utils"
	"github.com/urfave/negroni"
)

// Logger is a middleware handler that logs the request as it goes in and the response as it goes out.
type Logger struct {
	// Logger inherits from log.Logger used to log messages with the Logger middleware
	*logging.Logger
	*utils.Matcher
}

// NewLogger returns a new Logger instance
func NewLogger(appName string) *Logger {
	return &Logger{logging.NewLogger(appName), utils.NewMatcher()}
}

func (l *Logger) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	// 仅过滤GET请求中的action
	action := r.URL.Query().Get("api_action")
	if !l.Matcher.Match(action) {
		next(rw, r)
		return
	}

	start := time.Now()

	var ipPrefix string
	if r.RemoteAddr != "@" {
		ipPrefix = r.RemoteAddr + " - "
	}

	l.Infof("%vStarted %s %s", ipPrefix, r.Method, r.RequestURI)

	next(rw, r)

	res := rw.(negroni.ResponseWriter)
	l.Infof("%vCompleted %v %s in %v", ipPrefix, res.Status(), http.StatusText(res.Status()), time.Since(start))
}
