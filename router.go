// 路由模块

package api

import (
	"net/http"
)

type ActionFunc func(c *Context) (data interface{})

type Route struct {
	ActionCode string
	ActionFunc ActionFunc
	Hooks      map[string][]ActionFunc
}

// NewRoute return a new route.
func NewRoute(actionCode string, actionFunc ActionFunc) *Route {
	return &Route{
		actionCode, actionFunc, make(map[string][]ActionFunc),
	}
}

// Hook add set hook action at specified tag.
func (r *Route) Hook(tag string, actionFunc ActionFunc) *Route {
	if _, has := r.Hooks[tag]; !has {
		r.Hooks[tag] = make([]ActionFunc, 0, 1)
	}
	r.Hooks[tag] = append(r.Hooks[tag], actionFunc)
	return r
}

// buildActionMap return an action map according to routes.
// The key of the map is action code.
func buildActionMap(routes []*Route) (actionMap map[string]*Route) {
	// 分析路由生成ACTION MAP：{ActionCode: Route}
	actionMap = make(map[string]*Route)
	for _, route := range routes {
		actionMap[route.ActionCode] = route
	}
	return actionMap
}

// newApiHandler return a handler func using by http.HandleFunc.
func newApiHandler(app *App, routes []*Route) func(w http.ResponseWriter, r *http.Request) {
	// 转化为MAP，提高性能
	actionMap := buildActionMap(routes)

	return func(w http.ResponseWriter, r *http.Request) {
		// 只支持GET和POST
		if r.Method != "GET" && r.Method != "POST" {
			http.Error(w, "Unsupported request method!", http.StatusMethodNotAllowed)
			return
		}

		var resData interface{}

		ctx, err := NewContext(app, w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// 清理context操作移至handler最外层
		// defer ctx.Clear()

		// 系统维护中
		if app.UnderMaintenance {
			resData := ctx.Error.NewGroupError("global", errorSystemMaintenance)
			WriteResponse(ctx, resData)
		}

		apiAction := ctx.Input.GetAction()
		if apiAction == "" {
			resData = ctx.Error.NewGroupError("global", errorActionNotExist)
			goto output
		}

		if route, ok := actionMap[apiAction]; !ok {
			resData = ctx.Error.NewGroupError("global", errorActionNotExist)
			goto output
		} else {
			// Action之前的操作
			if beforeActions, has := app.Hooks["BeforeAction"]; has {
				if beforeActions != nil && len(beforeActions) > 0 {
					for _, beforeAction := range beforeActions {
						data := beforeAction(ctx)
						if data != nil {
							resData = data
							goto output
						}
					}
				}
			}
			if beforeActions, has := route.Hooks["BeforeAction"]; has {
				if beforeActions != nil && len(beforeActions) > 0 {
					for _, beforeAction := range beforeActions {
						data := beforeAction(ctx)
						if data != nil {
							resData = data
							goto output
						}
					}
				}
			}

			if route.ActionFunc != nil {
				// 执行 action
				resData = route.ActionFunc(ctx)
			}

			// Action之后的操作
			if afterActions, has := route.Hooks["AfterAction"]; has {
				if afterActions != nil && len(afterActions) > 0 {
					for _, afterAction := range afterActions {
						ctx.Set("result", resData)
						data := afterAction(ctx)
						if resData == nil && data != nil {
							resData = data
						}
					}
				}
			}
			if afterActions, has := app.Hooks["AfterAction"]; has {
				if afterActions != nil && len(afterActions) > 0 {
					for _, afterAction := range afterActions {
						ctx.Set("result", resData)
						data := afterAction(ctx)
						if resData == nil && data != nil {
							resData = data
						}
					}
				}
			}
		}

	output:
		// 输出结果
		WriteResponse(ctx, resData)
	}
}
