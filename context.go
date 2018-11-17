// 运行环境模块

package api

import (
	// "mime"
	"net/http"
	"strings"

	"github.com/go-xorm/xorm"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
)

type Context struct {
	App    *App
	Input  *Input
	Output *Output
	DB     *DbManager
	Model  *ModelManager
	Error  *ErrorManager
}

// 修复HTTP头部中的Content-Type
// 支付宝Notify通知结果头部如：
// Content-Type: application/x-www-form-urlencoded; text/html; charset=utf-8
// 为造成ParseForm()报错：mime: invalid media parameter
func fixHeader(h http.Header) {
	if ct, has := h["Content-Type"]; has && len(ct) > 0 {
		fields := strings.Split(ct[0], ";")
		okFields := []string{}
		if len(fields) > 1 {
			okFields = append(okFields, fields[0])
			for _, field := range fields[1:] {
				// 必须带有=，而且不能包含/
				if strings.IndexByte(field, '=') < 0 {
					continue
				}
				if strings.IndexByte(field, '/') >= 0 {
					continue
				}
				okFields = append(okFields, field)
			}
			h["Content-Type"][0] = strings.Join(okFields, ";")
		}
	}
}

func NewContext(app *App, w http.ResponseWriter, r *http.Request) (*Context, error) {
	fixHeader(r.Header)

	// 仅支持从GET/POST参数中获取Action
	// contentType := r.Header.Get("Content-Type")
	// if contentType != "" {
	// 	mediaType, _, err := mime.ParseMediaType(contentType)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	if strings.HasPrefix(mediaType, "multipart/") {
	// 		if err := r.ParseMultipartForm(0xffffffff); err != nil {
	// 			return nil, err
	// 		}
	// 	}
	// }

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	input := &Input{r}
	apiLang := input.Get("api_lang")
	if apiLang != "" {
		app.Error.SetLang(apiLang)
	} else {
		// 重置为应用默认配置
		app.Error.SetLang(app.Config.GetDefaultString("api.default.lang", "en_us"))
	}

	context := &Context{app, input, &Output{w}, app.DB, app.Model, app.Error}
	return context, nil
}

// Response return current request.
func (c *Context) Request() *http.Request {
	return c.Input.Request
}

// Response return current response writer.
func (c *Context) Response() http.ResponseWriter {
	return c.Output.ResponseWriter
}

// Session return the session with specified name.
func (c *Context) Session(sessionName string) (*Session, error) {
	store, err := c.App.SessionStore()
	if err != nil {
		return nil, err
	}
	session, err := store.Get(c.Input.Request, sessionName)
	if err != nil {
		return nil, err
	}

	session.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
	}
	return &Session{*session, c.Output.ResponseWriter}, nil
}

// Clear removes all values stored for a given request.
// This is usually called by a handler wrapper to clean up request variables at the end of a request lifetime. See ClearHandler().
func (c *Context) NewParams() *Params {
	return NewParams(c.Input.GetForm(), c.Error)
}

// Clear removes all values stored for a given request.
// This is usually called by a handler wrapper to clean up request variables at the end of a request lifetime. See ClearHandler().
func (c *Context) Clear() {
	context.Clear(c.Input.Request)
}

// GetDB returns a *xorm.Engine value stored for a given key in a given request.
func (c *Context) GetDB(key interface{}) *xorm.Engine {
	v, has := context.GetOk(c.Input.Request, key)
	if !has {
		return nil
	}
	db, ok := v.(*xorm.Engine)
	if !ok {
		return nil
	}
	return db
}

// Set stores a value for a given key in a given request.
func (c *Context) Set(key, val interface{}) {
	context.Set(c.Input.Request, key, val)
}

// Delete removes a value stored for a given key in a given request.
func (c *Context) Delete(key interface{}) {
	context.Delete(c.Input.Request, key)
}

// Get returns a value stored for a given key in a given request.
func (c *Context) Get(key interface{}) interface{} {
	return context.Get(c.Input.Request, key)
}

// GetOk returns stored value and presence state like multi-value return of map access.
func (c *Context) GetOk(key interface{}) (interface{}, bool) {
	return context.GetOk(c.Input.Request, key)
}

// GetAll returns all stored values for the request as a map. Nil is returned for invalid requests.
func (c *Context) GetAll() map[interface{}]interface{} {
	return context.GetAll(c.Input.Request)
}

// GetAllOk returns all stored values for the request as a map and a boolean value that indicates if the request was registered.
func (c *Context) GetAllOk() (map[interface{}]interface{}, bool) {
	return context.GetAllOk(c.Input.Request)
}

// GetString returns a string value stored for a given key in a given request.
func (c *Context) GetString(key interface{}) string {
	v, has := context.GetOk(c.Input.Request, key)
	if !has {
		return ""
	}
	str, ok := v.(string)
	if !ok {
		return ""
	}
	return str
}

// GetString returns a string value stored and presence state like multi-value return of map access.
func (c *Context) GetStringOk(key interface{}) (string, bool) {
	v, has := context.GetOk(c.Input.Request, key)
	if !has {
		return "", false
	}
	str, ok := v.(string)
	if !ok {
		return "", false
	}
	return str, true
}

// GetInt returns a int value stored for a given key in a given request.
func (c *Context) GetInt(key interface{}) int {
	v, has := context.GetOk(c.Input.Request, key)
	if !has {
		return 0
	}
	str, ok := v.(int)
	if !ok {
		return 0
	}
	return str
}

// GetInt returns a int value stored and presence state like multi-value return of map access.
func (c *Context) GetIntOk(key interface{}) (int, bool) {
	v, has := context.GetOk(c.Input.Request, key)
	if !has {
		return 0, false
	}
	str, ok := v.(int)
	if !ok {
		return 0, false
	}
	return str, true
}

// GetInt32 returns a int32 value stored for a given key in a given request.
func (c *Context) GetInt32(key interface{}) int32 {
	v, has := context.GetOk(c.Input.Request, key)
	if !has {
		return 0
	}
	str, ok := v.(int32)
	if !ok {
		return 0
	}
	return str
}

// GetInt32 returns a int32 value stored and presence state like multi-value return of map access.
func (c *Context) GetInt32Ok(key interface{}) (int32, bool) {
	v, has := context.GetOk(c.Input.Request, key)
	if !has {
		return 0, false
	}
	str, ok := v.(int32)
	if !ok {
		return 0, false
	}
	return str, true
}

// GetInt64 returns a int64 value stored for a given key in a given request.
func (c *Context) GetInt64(key interface{}) int64 {
	v, has := context.GetOk(c.Input.Request, key)
	if !has {
		return 0
	}
	str, ok := v.(int64)
	if !ok {
		return 0
	}
	return str
}

// GetInt64 returns a int64 value stored and presence state like multi-value return of map access.
func (c *Context) GetInt64Ok(key interface{}) (int64, bool) {
	v, has := context.GetOk(c.Input.Request, key)
	if !has {
		return 0, false
	}
	str, ok := v.(int64)
	if !ok {
		return 0, false
	}
	return str, true
}

// GetUint returns a uint value stored for a given key in a given request.
func (c *Context) GetUint(key interface{}) uint {
	v, has := context.GetOk(c.Input.Request, key)
	if !has {
		return 0
	}
	str, ok := v.(uint)
	if !ok {
		return 0
	}
	return str
}

// GetUint returns a uint value stored and presence state like multi-value return of map access.
func (c *Context) GetUintOk(key interface{}) (uint, bool) {
	v, has := context.GetOk(c.Input.Request, key)
	if !has {
		return 0, false
	}
	str, ok := v.(uint)
	if !ok {
		return 0, false
	}
	return str, true
}

// GetUint32 returns a uint32 value stored for a given key in a given request.
func (c *Context) GetUint32(key interface{}) uint32 {
	v, has := context.GetOk(c.Input.Request, key)
	if !has {
		return 0
	}
	str, ok := v.(uint32)
	if !ok {
		return 0
	}
	return str
}

// GetUint32 returns a uint32 value stored and presence state like multi-value return of map access.
func (c *Context) GetUint32Ok(key interface{}) (uint32, bool) {
	v, has := context.GetOk(c.Input.Request, key)
	if !has {
		return 0, false
	}
	str, ok := v.(uint32)
	if !ok {
		return 0, false
	}
	return str, true
}

// GetUint64 returns a uint64 value stored for a given key in a given request.
func (c *Context) GetUint64(key interface{}) uint64 {
	v, has := context.GetOk(c.Input.Request, key)
	if !has {
		return 0
	}
	str, ok := v.(uint64)
	if !ok {
		return 0
	}
	return str
}

// GetUint64 returns a uint64 value stored and presence state like multi-value return of map access.
func (c *Context) GetUint64Ok(key interface{}) (uint64, bool) {
	v, has := context.GetOk(c.Input.Request, key)
	if !has {
		return 0, false
	}
	str, ok := v.(uint64)
	if !ok {
		return 0, false
	}
	return str, true
}

// CloseResponse ignore the action return value and will not response anymore when action return.
func (c *Context) CloseResponse() {
	c.Set("response_closed", true)
}

// UpgradeWebsocket upgrade current request to websocket, and return the websocket connection.
func (c *Context) UpgradeWebsocket() (*websocket.Conn, error) {
	c.CloseResponse()

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(c.Output.ResponseWriter, c.Input.Request, nil)
	if err != nil {
		// 不能再输出HTTP错误，否则会报：
		// http: response.WriteHeader on hijacked connection
		// c.Response().WriteHeader(http.StatusInternalServerError)
		return nil, err
	}
	return conn, nil
}
