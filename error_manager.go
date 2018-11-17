// 错误管理

package api

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type ErrorType uint

type ErrorManager struct {
	lang         string
	groupDefines map[string]map[ErrorType]*ErrorDefine   // group-errortype-define
	groupWords   map[string]map[string]map[string]string // group-lang-word-phrase
	mutex        sync.RWMutex
}

type ErrorDefine struct {
	code        string
	fieldCounts []int
	msgTmpls    map[string]map[int]string
}

// NewErrorDefine return an error define.
func NewErrorDefine(code string, fieldCounts []int, msgTmpls map[string]map[int]string) *ErrorDefine {
	return &ErrorDefine{code, fieldCounts, msgTmpls}
}

// NewErrorManager return an error manager.
func NewErrorManager() *ErrorManager {
	em := new(ErrorManager)
	em.lang = "en_us"
	em.groupDefines = make(map[string]map[ErrorType]*ErrorDefine, 0)
	em.groupWords = make(map[string]map[string]map[string]string, 0)
	return em
}

// SetLang set the default language when generate message.
func (em *ErrorManager) SetLang(lang string) {
	em.mutex.Lock()
	defer em.mutex.Unlock()
	em.lang = lang
}

// GetLang return the default language of ErrorManager.
func (em *ErrorManager) GetLang() string {
	em.mutex.RLock()
	defer em.mutex.RUnlock()
	return em.lang
}

// RegisterError register a new error define in default group.
func (em *ErrorManager) RegisterError(errType ErrorType, define *ErrorDefine) {
	em.RegisterGroupError("default", errType, define)
}

// RegisterErrors register new error defines in default group.
func (em *ErrorManager) RegisterErrors(defines map[ErrorType]*ErrorDefine) {
	em.RegisterGroupErrors("default", defines)
}

// RegisterWords register new word map in default group.
func (em *ErrorManager) RegisterWords(wordMap map[string]map[string]string) {
	em.RegisterGroupWords("default", wordMap)
}

// RegisterGroupError register a new error define in specified group.
func (em *ErrorManager) RegisterGroupError(group string, errType ErrorType, define *ErrorDefine) {
	g, ok := em.groupDefines[group]
	if !ok {
		em.groupDefines[group] = make(map[ErrorType]*ErrorDefine)
		g = em.groupDefines[group]
	}
	g[errType] = define
}

// RegisterGroupErrors register new error defines in specified group.
func (em *ErrorManager) RegisterGroupErrors(group string, defines map[ErrorType]*ErrorDefine) {
	g, ok := em.groupDefines[group]
	if !ok {
		em.groupDefines[group] = make(map[ErrorType]*ErrorDefine, 0)
		g = em.groupDefines[group]
	}
	for errType, define := range defines {
		g[errType] = define
	}
}

// RegisterGroupWords register new word map in specified group.
func (em *ErrorManager) RegisterGroupWords(group string, wordMap map[string]map[string]string) {
	g, ok := em.groupWords[group]
	if !ok {
		em.groupWords[group] = make(map[string]map[string]string, 0)
		g = em.groupWords[group]
	}
	for lang, m := range wordMap {
		vl, ok := g[lang]
		if !ok {
			g[lang] = make(map[string]string)
			vl = g[lang]
		}
		for k, v := range m {
			vl[k] = v
		}
	}
}

// New return an new error with specified error type.
// If error type need more fields, fields should be specified and will fill
// to error code.
// The message output in response will be automatically generated according
// to error code.
func (em *ErrorManager) New(errType ErrorType, fields ...string) *Error {
	return em.NewGroupError("default", errType, fields...)
}

// NewGroupError return an new error with specified error type of specified group.
func (em *ErrorManager) NewGroupError(group string, errType ErrorType, fields ...string) *Error {
	groupDefines, ok := em.groupDefines[group]
	if !ok {
		return NewError("InternalError", "error group not found: "+group)
	}

	define, ok := groupDefines[errType]
	if !ok {
		return NewError("InternalError", "error type not found in group: "+group)
	}

	code, err := em.buildErrorCode(group, define, fields...)
	if err != nil {
		return NewError("InternalError", err.Error())
	}

	message, err := em.buildErrorMessage(group, define, fields...)
	if err != nil {
		return NewError("InternalError", err.Error())
	}

	return NewError(code, message)
}

// 检查字段是否是大驼峰命名
// 允许带中括号[]
func checkField(str string) bool {
	fields := strings.Split(str, "|")
	for _, field := range fields {
		if len(field) == 0 {
			return false
		}

		// 首字母必须大写
		if !(field[0] >= 'A' && field[0] <= 'Z') {
			return false
		}

		// 必须由数字、字母或中括号组成
		bracketOpen := false
		for _, b := range field[1:] {
			if b == '[' {
				if bracketOpen {
					return false
				}
				bracketOpen = true
				continue
			}
			if b == ']' {
				if !bracketOpen {
					return false
				}
				bracketOpen = false
				continue
			}

			// 中括号中允许带其它符号
			if !bracketOpen {
				if !(b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9') {
					return false
				}
			}
		}
		// 中括号未闭合
		if bracketOpen {
			return false
		}
	}
	return true
}

func (em *ErrorManager) transWord(group string, word string) string {
	g, ok := em.groupWords[group]
	if !ok {
		return word
	}
	words, ok := g[em.lang]
	if !ok {
		return word
	}
	phrase, ok := words[word]
	if !ok {
		return word
	}
	return phrase
}

func (em *ErrorManager) buildErrorCode(group string, define *ErrorDefine, fields ...string) (string, error) {
	errorCode := define.code
	fieldCount := len(fields)
	for _, count := range define.fieldCounts {
		if fieldCount == count { // 参数个数匹配
			goto combine_flelds
		}
	}
	return "", errors.New("Wrong fields count given with " + errorCode)

combine_flelds:
	// 验证是否合法
	for _, field := range fields {
		if !checkField(field) {
			return "", errors.New("Wrong field format of " + field)
		}
	}
	if len(fields) > 0 {
		errorCode += ":" + strings.Join(fields, ":")
	}
	return errorCode, nil
}

func (em *ErrorManager) buildErrorMessage(group string, define *ErrorDefine, fields ...string) (string, error) {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	lang := em.lang
	msgTmpls, ok := define.msgTmpls[lang]
	if !ok {
		return "", nil
	}

	msgTmpl, ok := msgTmpls[len(fields)]
	if !ok {
		return "", errors.New("Wrong fields count given with " + define.code)
	}

	// match like: {1}, {1: or }
	var placement = regexp.MustCompile(`\{\d(?:\:[^\}]+)?\}`)
	msg := placement.ReplaceAllStringFunc(msgTmpl, func(match string) string {
		match = strings.Trim(match, "{}")
		matchFields := strings.SplitN(match, ":", 2)
		index, _ := strconv.Atoi(matchFields[0])

		// 找出错误码中的相应字段
		var result string
		if index > 0 && index <= len(fields) {
			result = fields[index-1]
		}

		// 替换字段中的 |
		if strings.Contains(result, "|") {
			resultWords := strings.Split(result, "|")
			for i, resultWord := range resultWords {
				resultWords[i] = em.transWord(group, resultWord)
			}
			orStr := ""
			if len(matchFields) == 2 {
				orStr = matchFields[1]
			} else {
				orStr = "|"
			}
			result = strings.Join(resultWords, orStr)
		} else {
			result = em.transWord(group, result)
		}

		return result
	})

	return msg, nil
}
