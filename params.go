// 参数

package api

import (
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/go-apibox/filter"
	"github.com/go-apibox/types"
)

type Params struct {
	Rules        []*ParamRule
	RawValues    url.Values
	ParsedValues map[string]interface{}
	Error        *ErrorManager
}

type ParamRule struct {
	ParamName string
	Filters   []filter.Filter
	Parsed    bool
}

// NewParams return a new Params object.
func NewParams(values url.Values, em *ErrorManager) *Params {
	rules := make([]*ParamRule, 0)
	parsedValues := make(map[string]interface{})
	return &Params{rules, values, parsedValues, em}
}

// Add add a new param with filter functions.
func (p *Params) Add(paramName string, filters ...filter.Filter) *Params {
	p.Rules = append(p.Rules, &ParamRule{paramName, filters, false})
	return p
}

// AddPagination add a pagination params with filter functions.
func (p *Params) AddPagination() *Params {
	p.Add("_pageNumber", filter.Default(1), filter.Int().Min(1))
	p.Add("_pageSize", filter.Default(10), filter.Int().Min(1).Max(1000))
	return p
}

// SetDefaultOrder set default order of pagination.
func (p *Params) AddOrderBy(defaultOrderBy string, defaultOrder string, allowOrderBys []string, allowMultiOrder bool) *Params {
	orderByFilter := filter.StringSet().ItemIn(allowOrderBys)
	if !allowMultiOrder {
		orderByFilter.MaxCount(1)
	}
	orderFilter := filter.StringSet().ToLower().ItemIn([]string{"asc", "desc"})
	p.Add("_orderBy", filter.Default(defaultOrderBy), orderByFilter)
	p.Add("_order", filter.Default(defaultOrder), orderFilter)
	return p
}

// SetDefaultOrder set default order of pagination.
func (p *Params) SetDefaultOrder(string) *Params {
	return p
}

// Del remove params from list.
func (p *Params) Del(paramNames ...string) *Params {
	for i := 0; i < len(p.Rules); i++ {
		rule := p.Rules[i]
		for _, paramName := range paramNames {
			if rule.ParamName == paramName {
				p.Rules = append(p.Rules[:i], p.Rules[i+1:]...)
				i--
				break
			}
		}
	}
	// 同时删除已分析的值
	for _, paramName := range paramNames {
		delete(p.ParsedValues, paramName)
	}
	return p
}

// Validate one param according to the rule and save the parsed value.
func (p *Params) Validate(paramName string, filters ...filter.Filter) *Error {
	var val interface{}
	var ok bool
	var err *filter.Error

	p.Rules = append(p.Rules, &ParamRule{paramName, filters, true})

	val, ok = p.RawValues[paramName]
	if ok {
		if len(p.RawValues[paramName]) == 1 {
			val = p.RawValues[paramName][0]
		}
	} else {
		val = nil
	}

	for _, f := range filters {
		val, err = f.Run(paramName, val)
		if err != nil {
			return p.Error.New(ErrorType(err.Type), err.Fields...)
		}
	}

	p.ParsedValues[paramName] = val
	return nil
}

// Parse parse the values according to the rules.
func (p *Params) Parse() *Error {
	var val interface{}
	var ok bool
	var err *filter.Error

	for _, rule := range p.Rules {
		if rule.Parsed {
			continue
		}
		rule.Parsed = true

		paramName := rule.ParamName
		filters := rule.Filters

		val, ok = p.RawValues[paramName]
		if ok {
			if len(p.RawValues[paramName]) == 1 {
				val = p.RawValues[paramName][0]
			}
		} else {
			val = nil
		}

		for _, f := range filters {
			val, err = f.Run(paramName, val)
			if err != nil {
				return p.Error.New(ErrorType(err.Type), err.Fields...)
			}
		}

		p.ParsedValues[paramName] = val
	}

	// 不能直接复制，结合dbop后将产生不安全因素
	// // 将未分析的参数直接复制
	// for paramName, paramValue := range p.RawValues {
	// 	if _, ok := p.ParsedValues[paramName]; !ok {
	// 		p.ParsedValues[paramName] = paramValue
	// 	}
	// }

	return nil
}

// Set set the param value.
func (p *Params) Set(paramName string, paramValue interface{}) *Params {
	p.ParsedValues[paramName] = paramValue
	return p
}

// Has return the if param exist.
func (p *Params) Has(paramName string) bool {
	if v, ok := p.ParsedValues[paramName]; ok {
		if v != nil {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

// Get return the parsed value of param.
func (p *Params) Get(paramName string) interface{} {
	if v, ok := p.ParsedValues[paramName]; ok {
		return v
	} else {
		return nil
	}
}

// GetAll return all values in params.
func (p *Params) GetAll() map[string]interface{} {
	return p.ParsedValues
}

// GetString return the parsed value of param as string.
func (p *Params) GetString(paramName string) string {
	v := p.Get(paramName)
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case filter.CIDRAddr:
		return val.RawString
	case *filter.CIDRAddr:
		return val.RawString
	}
	return fmt.Sprint(v)
}

// GetInt return the parsed value of param as int.
func (p *Params) GetInt(paramName string) int {
	v := p.Get(paramName)
	switch val := v.(type) {
	case int:
		return val
	}
	return 0
}

// GetInt32 return the parsed value of param as int32.
func (p *Params) GetInt32(paramName string) int32 {
	v := p.Get(paramName)
	switch val := v.(type) {
	case int32:
		return val
	}
	return 0
}

// GetInt64 return the parsed value of param as int64.
func (p *Params) GetInt64(paramName string) int64 {
	v := p.Get(paramName)
	switch val := v.(type) {
	case int64:
		return val
	}
	return 0
}

// GetUint return the parsed value of param as uint.
func (p *Params) GetUint(paramName string) uint {
	v := p.Get(paramName)
	switch val := v.(type) {
	case uint:
		return val
	}
	return 0
}

// GetUint32 return the parsed value of param as uint32.
func (p *Params) GetUint32(paramName string) uint32 {
	v := p.Get(paramName)
	switch val := v.(type) {
	case uint32:
		return val
	}
	return 0
}

// GetUint64 return the parsed value of param as uint64.
func (p *Params) GetUint64(paramName string) uint64 {
	v := p.Get(paramName)
	switch val := v.(type) {
	case uint64:
		return val
	}
	return 0
}

// GetTime return the parsed value of param as time.
func (p *Params) GetTime(paramName string) *time.Time {
	v := p.Get(paramName)
	switch val := v.(type) {
	case time.Time:
		return &val
	case *time.Time:
		return val
	}
	return nil
}

// GetTimestamp return the parsed value of param as Timestamp.
func (p *Params) GetTimestamp(paramName string) uint32 {
	v := p.Get(paramName)
	switch val := v.(type) {
	case uint32:
		return val
	}
	return 0
}

// GetIntRange return the parsed value of param as IntRange.
func (p *Params) GetIntRange(paramName string) *types.IntRange {
	v := p.Get(paramName)
	switch val := v.(type) {
	case *types.IntRange:
		return val
	case types.IntRange:
		return &val
	}

	return nil
}

// GetInt32Range return the parsed value of param as Int32Range.
func (p *Params) GetInt32Range(paramName string) *types.Int32Range {
	v := p.Get(paramName)
	switch val := v.(type) {
	case *types.Int32Range:
		return val
	case types.Int32Range:
		return &val
	}

	return nil
}

// GetInt64Range return the parsed value of param as Int64Range.
func (p *Params) GetInt64Range(paramName string) *types.Int64Range {
	v := p.Get(paramName)
	switch val := v.(type) {
	case *types.Int64Range:
		return val
	case types.Int64Range:
		return &val
	}

	return nil
}

// GetUintRange return the parsed value of param as UintRange.
func (p *Params) GetUintRange(paramName string) *types.UintRange {
	v := p.Get(paramName)
	switch val := v.(type) {
	case *types.UintRange:
		return val
	case types.UintRange:
		return &val
	}

	return nil
}

// GetUint32Range return the parsed value of param as Uint32Range.
func (p *Params) GetUint32Range(paramName string) *types.Uint32Range {
	v := p.Get(paramName)
	switch val := v.(type) {
	case *types.Uint32Range:
		return val
	case types.Uint32Range:
		return &val
	}

	return nil
}

// GetUint64Range return the parsed value of param as Uint64Range.
func (p *Params) GetUint64Range(paramName string) *types.Uint64Range {
	v := p.Get(paramName)
	switch val := v.(type) {
	case *types.Uint64Range:
		return val
	case types.Uint64Range:
		return &val
	}

	return nil
}

// GetTimeRange return the parsed value of param as TimeRange.
func (p *Params) GetTimeRange(paramName string) *types.TimeRange {
	v := p.Get(paramName)

	switch val := v.(type) {
	case *types.TimeRange:
		return val
	case types.TimeRange:
		return &val
	}

	return nil
}

// GetTimeRange return the parsed value of param as TimestampRange.
func (p *Params) GetTimestampRange(paramName string) *types.TimestampRange {
	v := p.Get(paramName)

	switch val := v.(type) {
	case *types.TimestampRange:
		return val
	case types.TimestampRange:
		return &val
	}

	return nil
}

// GetStringArray return the parsed value of param as string array.
func (p *Params) GetStringArray(paramName string) []string {
	v := p.Get(paramName)
	switch val := v.(type) {
	case []string:
		return val
	}
	return []string{}
}

// GetIntArray return the parsed value of param as int array.
func (p *Params) GetIntArray(paramName string) []int {
	v := p.Get(paramName)
	switch val := v.(type) {
	case []int:
		return val
	}
	return []int{}
}

// GetInt32Array return the parsed value of param as int32 array.
func (p *Params) GetInt32Array(paramName string) []int32 {
	v := p.Get(paramName)
	switch val := v.(type) {
	case []int32:
		return val
	}
	return []int32{}
}

// GetInt64Array return the parsed value of param as int64 array.
func (p *Params) GetInt64Array(paramName string) []int64 {
	v := p.Get(paramName)
	switch val := v.(type) {
	case []int64:
		return val
	}
	return []int64{}
}

// GetUintArray return the parsed value of param as uint array.
func (p *Params) GetUintArray(paramName string) []uint {
	v := p.Get(paramName)
	switch val := v.(type) {
	case []uint:
		return val
	}

	return []uint{}
}

// GetUint32Array return the parsed value of param as uint32 array.
func (p *Params) GetUint32Array(paramName string) []uint32 {
	v := p.Get(paramName)
	switch val := v.(type) {
	case []uint32:
		return val
	}

	return []uint32{}
}

// GetUint64Array return the parsed value of param as uint64 array.
func (p *Params) GetUint64Array(paramName string) []uint64 {
	v := p.Get(paramName)
	switch val := v.(type) {
	case []uint64:
		return val
	}

	return []uint64{}
}

// GetTimeArray return the parsed value of param as time array.
func (p *Params) GetTimeArray(paramName string) []*time.Time {
	v := p.Get(paramName)

	switch val := v.(type) {
	case []time.Time:
		rt := make([]*time.Time, 0, len(val))
		for _, v := range val {
			rt = append(rt, &v)
		}
		return rt
	case []*time.Time:
		return val
	}

	return []*time.Time{}
}

// GetIP return the parsed value of param as net.IP.
func (p *Params) GetIP(paramName string) net.IP {
	v := p.Get(paramName)

	switch val := v.(type) {
	case net.IP:
		return val
	}

	return nil
}

// GetIPArray return the parsed value of param as net.IP array.
func (p *Params) GetIPArray(paramName string) []net.IP {
	v := p.Get(paramName)

	switch val := v.(type) {
	case []net.IP:
		return val
	}

	return nil
}
