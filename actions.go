// 内置action

package api

// 返回 Cookie 密钥对
func APIBoxSessionGetKeyAction(c *Context) interface{} {
	store, err := c.App.SessionStore()
	if err != nil {
		return c.Error.New(ErrorInternalError, "SessionStoreFailed").SetMessage(err.Error())
	}

	return store.GetKeyPairs()
}
