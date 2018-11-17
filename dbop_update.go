package api

import (
	"fmt"
	"net"
	"reflect"
	"strings"

	"github.com/go-apibox/filter"
	"github.com/go-apibox/utils"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
)

func Update(c *Context, bean interface{}, params *Params) interface{} {
	return UpdateEx(c, bean, params, nil)
}

func SessionUpdate(c *Context, session *xorm.Session, bean interface{}, params *Params) interface{} {
	return SessionUpdateEx(c, session, bean, params, nil)
}

func UpdateEx(c *Context, bean interface{}, params *Params, querySettings map[string]string) interface{} {
	db, err := getDB(c)
	if err != nil {
		c.App.Logger.Error("(dbop error): [DBNotExist] %s", err.Error())
		return c.Error.New(ErrorInternalError, "DBNotExist").SetMessage("Database not exist.")
	}
	defer closeDB(c, db)

	session := db.NewSession()
	defer session.Close()

	return SessionUpdateEx(c, session, bean, params, querySettings)
}

func SessionUpdateEx(c *Context, session *xorm.Session, bean interface{}, params *Params, querySettings map[string]string) interface{} {
	var modelName string
	var modelVal reflect.Value

	modelName, isString := bean.(string)
	if !isString {
		modelVal = reflect.Indirect(reflect.ValueOf(bean))
		if modelVal.Kind() != reflect.Struct {
			return c.Error.New(ErrorInternalError, "WrongParamType").SetMessage("Expected a pointer to a struct!")
		}
		modelName = modelVal.Type().Name()
	}

	modelDefine := c.Model.Get(modelName)
	if modelDefine == nil {
		return c.Error.New(ErrorInternalError, "ModelNotRegistered").SetMessage("Model " + modelName + " not registered!")
	}
	if isString {
		modelVal = reflect.Indirect(reflect.New(modelDefine.Type))
	}

	// 取出ID字段
	pkFields := modelDefine.TagFields("pk")
	if len(pkFields) == 0 {
		c.App.Logger.Error("(dbop error): [NoPrimaryKey]")
		return c.Error.New(ErrorInternalError, "NoPrimaryKey").SetMessage("No primary key.")
	}

	// 查询定义处理
	allQueryDefines := parseQuerySettings(querySettings)

	// 从请求参数中复制值
	pk := core.PK{}
	columns := []string{}

	needUpdate := false
	fields := modelDefine.Fields()
	for _, field := range fields {
		fieldVal := modelVal.FieldByName(field)
		if v := params.Get(field); v != nil {
			// ID不允许更新
			if modelDefine.FieldHasTag(field, "pk") {
				pk = append(pk, v)
			} else {
				if v != nil {
					var columnName string
					if colTag := modelDefine.FieldGetTag(field, "column"); colTag != nil {
						columnName = colTag.Params[0]
					} else {
						columnName = session.Engine().ColumnMapper.Obj2Table(field)
					}

					qDefines, qDefined := allQueryDefines[field]
					processed := false
					if qDefined {
						for _, qDefine := range qDefines {
							switch qDefine.queryType {
							case "expr":
								expr := strings.Replace(qDefine.queryArgs[0], "$", fmt.Sprint(v), -1)
								session.SetExpr(columnName, expr)
								processed = true
							}
						}
					}

					if !processed {
						columns = append(columns, columnName)
						var rv reflect.Value
						switch tv := v.(type) {
						case *net.IP, net.IP,
							*filter.CIDRAddr, filter.CIDRAddr:
							rv = reflect.ValueOf(fmt.Sprint(tv))
						case []string:
							rv = reflect.ValueOf(strings.Join(tv, ","))
						default:
							rv = reflect.ValueOf(tv)
						}
						fieldVal.Set(rv)
					}
					needUpdate = true
				}
			}
		} else {
			// 检查model的自动填值项
			tags := modelDefine.FieldTags(field)
			for _, tag := range tags {
				switch tag.Name {
				case "updatetime":
					fieldVal.Set(reflect.ValueOf(utils.Timestamp()))
					needUpdate = true

					var columnName string
					if colTag := modelDefine.FieldGetTag(field, "column"); colTag != nil {
						columnName = colTag.Params[0]
					} else {
						columnName = session.Engine().ColumnMapper.Obj2Table(field)
					}
					columns = append(columns, columnName)
				}
			}
		}
	}

	if !needUpdate {
		return utils.Combine("Affected", 0)
	}

	// ID作为条件
	affected, err := session.Cols(columns...).Id(pk).Update(modelVal.Addr().Interface())
	if err != nil {
		c.App.Logger.Error("(dbop error): [UpdateFailed] %s", err.Error())
		return c.Error.New(ErrorInternalError, "UpdateFailed", modelName).SetMessage("Update failed.")
	}

	return utils.Combine("Affected", affected)
}
