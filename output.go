// 输出模块

package api

import (
	"net/http"
)

type Output struct {
	ResponseWriter http.ResponseWriter
}

func (o *Output) Write(data []byte) (int, error) {
	return o.ResponseWriter.Write(data)
}

func (o *Output) SetHeader(key, value string) {
	o.ResponseWriter.Header().Set(key, value)
}
