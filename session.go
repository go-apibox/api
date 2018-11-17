// Session模块

package api

import (
	"net/http"

	"github.com/go-apibox/session"
)

type Session struct {
	session.Session
	w http.ResponseWriter
}

func (s *Session) Save() error {
	return s.Session.Save(s.w)
}

func (s *Session) Destroy() error {
	return s.Session.Destroy(s.w)
}
