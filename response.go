// 输出模块

package api

import (
	"encoding/json"
	"net/http"
	"os"
)

type Result struct {
	ACTION  string
	CODE    string
	MESSAGE string
	DATA    interface{}
}

type SuccessResult struct {
	ACTION string
	CODE   string
	DATA   interface{}
}

type ErrorResult struct {
	ACTION  string
	CODE    string
	MESSAGE string
}

type ErrorResultWithData struct {
	ACTION  string
	CODE    string
	MESSAGE string
	DATA    interface{}
}

var debugLevel string

func init() {
	switch os.Getenv("DEBUG_LEVEL") {
	case "full":
		debugLevel = "full"
	case "code":
		debugLevel = "code"
	}

	// 读取完立即清空，防止被子进程继承
	// os.Setenv("DEBUG_LEVEL", "")
}

// WriteResponse format api result according to the format specified by request params.
func WriteResponse(c *Context, data interface{}) {
	if closed, ok := c.Get("response_closed").(bool); ok && closed {
		return
	}

	c.Response().Header().Set("Server", c.App.ServerName)

	defaultFormat := c.App.Config.GetDefaultString("api.default.format", "json")
	defaultCallback := c.App.Config.GetDefaultString("api.default.callback", "callback")
	var defaultDebug string
	if c.App.Config.GetDefaultBool("api.default.debug", false) {
		defaultDebug = "1"
	} else {
		defaultDebug = "0"
	}

	// 根据请求参数中的配置值格式化Api输出
	apiAction := c.Input.Get("api_action")
	apiFormat := c.Input.GetDefault("api_format", defaultFormat)
	apiCallback := c.Input.GetDefault("api_callback", defaultCallback)
	apiDebug := c.Input.GetDefault("api_debug", defaultDebug)
	allowFormats := c.App.Config.GetDefaultStringArray("api.allow_formats", []string{})
	if len(allowFormats) == 0 {
		apiFormat = "json"
	} else {
		formatAllow := false
		for _, v := range allowFormats {
			if v == apiFormat {
				formatAllow = true
				break
			}
		}
		if !formatAllow {
			apiFormat = allowFormats[0]
		}
	}

	c.Set("returnData", data)

	switch debugLevel {
	case "full":
		apiData := makeData(apiAction, data)

		var jsonBytes []byte
		jsonBeauty := apiDebug == "1"

		if apiData.CODE == "ok" {
			result := new(SuccessResult)
			result.CODE = apiData.CODE
			result.ACTION = apiData.ACTION
			result.DATA = apiData.DATA
			jsonBytes = toJSON(result, jsonBeauty)
		} else {
			if apiData.DATA == nil {
				result := new(ErrorResult)
				result.CODE = apiData.CODE
				result.ACTION = apiData.ACTION
				result.MESSAGE = apiData.MESSAGE
				jsonBytes = toJSON(result, jsonBeauty)
			} else {
				result := new(ErrorResultWithData)
				result.CODE = apiData.CODE
				result.ACTION = apiData.ACTION
				result.MESSAGE = apiData.MESSAGE
				result.DATA = apiData.DATA
				jsonBytes = toJSON(result, jsonBeauty)
			}
		}

		c.App.Logger.Debug("\n"+
			">>>>>>>>>>>>>>>>>>>>> DEBUG >>>>>>>>>>>>>>>>>>>>\n"+
			"%s\n>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>", string(jsonBytes))

	case "code":
		apiData := makeData(apiAction, data)

		c.App.Logger.Debug("DEBUG: %s", apiData.CODE)
	}

	WriteData(c.Response(), c.Request(), data, apiAction, apiFormat, apiCallback, apiDebug)
}

func WriteData(w http.ResponseWriter, r *http.Request, data interface{},
	apiAction string, apiFormat string, apiCallback string, apiDebug string) {
	apiData := makeData(apiAction, data)

	var jsonBytes []byte
	jsonBeauty := apiDebug == "1"

	if apiData.CODE == "ok" {
		result := new(SuccessResult)
		result.CODE = apiData.CODE
		result.ACTION = apiData.ACTION
		result.DATA = apiData.DATA
		jsonBytes = toJSON(result, jsonBeauty)
	} else {
		if apiData.DATA == nil {
			result := new(ErrorResult)
			result.CODE = apiData.CODE
			result.ACTION = apiData.ACTION
			result.MESSAGE = apiData.MESSAGE
			jsonBytes = toJSON(result, jsonBeauty)
		} else {
			result := new(ErrorResultWithData)
			result.CODE = apiData.CODE
			result.ACTION = apiData.ACTION
			result.MESSAGE = apiData.MESSAGE
			result.DATA = apiData.DATA
			jsonBytes = toJSON(result, jsonBeauty)
		}
	}

	if apiFormat == "json" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(jsonBytes)
	} else { // jsonp
		jsonpBytes := make([]byte, 0, len(apiCallback)+len(jsonBytes)+3)
		jsonpBytes = append(jsonpBytes, []byte(apiCallback)...)
		jsonpBytes = append(jsonpBytes, '(')
		jsonpBytes = append(jsonpBytes, jsonBytes...)
		jsonpBytes = append(jsonpBytes, ')', ';')
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Write(jsonpBytes)
	}
}

func makeData(action string, data interface{}) *Result {
	switch err := data.(type) {
	case *Error:
		resData := &Result{
			ACTION:  action,
			CODE:    err.Code,
			MESSAGE: err.Message,
			DATA:    nil,
		}
		if err.Data != nil {
			resData.DATA = err.Data
		}
		return resData
	default:
		resData := &Result{
			ACTION:  action,
			CODE:    "ok",
			MESSAGE: "",
			DATA:    data,
		}
		return resData
	}
}

// 根据handler返回结果格式化输出JSON数据
func toJSON(apiData interface{}, needIndent bool) []byte {
	if needIndent {
		resStr, _ := json.MarshalIndent(apiData, "", "    ")
		return resStr
	} else {
		resStr, _ := json.Marshal(apiData)
		return resStr
	}
}
