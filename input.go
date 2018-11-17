// 输入模块

package api

import (
	"net"
	"net/http"
	"net/url"
)

type Input struct {
	Request *http.Request
}

// 获取请求值
func (i *Input) Get(key string) string {
	return i.Request.Form.Get(key)
}

// 获取请求值，如果请求值为空，则返回default值
func (i *Input) GetDefault(key string, defaultValue string) string {
	if vals, has := i.Request.Form[key]; has && len(vals) > 0 {
		return vals[0]
	}
	return defaultValue
}

// GetAction return the api_action param value.
func (i *Input) GetAction() string {
	return i.Request.Form.Get("api_action")
}

// GetRealIp returns the real ip address of request client.
func (i *Input) GetRealIp() string {
	realIp := i.Request.Header.Get("X-Real-IP")
	if realIp != "" {
		return realIp
	}
	if i.Request.RemoteAddr == "@" {
		return "@"
	}
	remoteIp, _, _ := net.SplitHostPort(i.Request.RemoteAddr)
	return remoteIp
}

// GetAll return all request params.
func (i *Input) GetAll() map[string]string {
	params := make(map[string]string)
	for key, val := range i.Request.Form {
		params[key] = val[0]
	}
	return params
}

// GetAll return the row request params.
func (i *Input) GetForm() url.Values {
	return i.Request.Form
}

// Has check if the key is exist in request params.
func (i *Input) Has(key string) bool {
	if _, ok := i.Request.Form[key]; ok {
		return true
	}
	return false
}
