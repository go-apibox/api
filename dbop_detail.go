package api

import (
	"reflect"
	"strings"

	"git.quyun.com/apibox/utils"
	"github.com/fatih/structs"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
)

func Detail(c *Context, bean interface{}, params *Params) interface{} {
	return DetailJoin(c, bean, params, nil)
}

func SessionDetail(c *Context, session *xorm.Session, beans interface{}, params *Params) interface{} {
	return SessionDetailJoin(c, session, beans, params, nil)
}

func DetailJoin(c *Context, bean interface{}, params *Params, joinConds [][]string) interface{} {
	db, err := getDB(c)
	if err != nil {
		c.App.Logger.Error("(dbop error): [DBNotExist] %s", err.Error())
		return c.Error.New(ErrorInternalError, "DBNotExist").SetMessage("Database not exist.")
	}
	defer closeDB(c, db)

	session := db.NewSession()
	defer session.Close()

	return SessionDetailJoin(c, session, bean, params, joinConds)
}

func SessionDetailJoin(c *Context, session *xorm.Session, bean interface{}, params *Params, joinConds [][]string) interface{} {
	if joinConds != nil {
		// format: (join_operator, tablename, condition)
		for _, joinCond := range joinConds {
			if len(joinCond) != 3 {
				return c.Error.New(ErrorInternalError, "WrongJoinCond").SetMessage("Join cond format must be: {join_operator, tablename, condition}!")
			}

			// 处理AS语句，如：user AS u
			var condTable interface{}
			isAs := false
			if strings.IndexByte(joinCond[1], ' ') > 0 {
				tableFields := strings.Split(joinCond[1], " ")
				if len(tableFields) == 3 {
					secondField := strings.ToLower(strings.Trim(tableFields[1], " "))
					if secondField == "as" {
						isAs = true
						condTable = []string{strings.Trim(tableFields[0], " "), strings.Trim(tableFields[2], " ")}
					}
				}
			}
			if !isAs {
				condTable = joinCond[1]
			}
			session.Join(joinCond[0], condTable, joinCond[2])
		}
	}

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

	// 取出ID字段
	modelDefine := c.Model.Get(modelName)
	if modelDefine == nil {
		return c.Error.New(ErrorInternalError, "ModelNotRegistered").SetMessage("Model " + modelName + " not registered!")
	}
	if isString {
		modelVal = reflect.Indirect(reflect.New(modelDefine.Type))
	}
	pkFields := modelDefine.TagFields("pk")
	if len(pkFields) == 0 {
		return c.Error.New(ErrorInternalError, "UnefinedPK").SetMessage("Primary key is undefined!")
	}

	// 从请求参数中复制值
	pk := core.PK{}
	pkCols := []string{}

	for _, pkField := range pkFields {
		if v := params.Get(pkField); v == nil {
			return c.Error.New(ErrorInternalError, "IncompletePKValue").SetMessage("Primary key value is incomplete!")
		} else {
			pk = append(pk, v)

			var columnName string
			if colTag := modelDefine.FieldGetTag(pkField, "column"); colTag != nil {
				columnName = colTag.Params[0]
			} else {
				columnName = session.Engine().ColumnMapper.Obj2Table(pkField)
			}
			pkCols = append(pkCols, columnName)
		}
	}

	// 要隐藏的字段
	omitColumns := []string{}
	hiddenDetailFields := []string{}
	hiddenFields := modelDefine.TagFields("hidden")
	for _, hdField := range hiddenFields {
		tags := modelDefine.FieldTags(hdField)
		for _, tag := range tags {
			if tag.Name == "hidden" {
				if len(tag.Params) == 0 || tag.Params[0] == "*" || tag.Params[0] == "detail" {
					var columnName string
					if colTag := modelDefine.FieldGetTag(hdField, "column"); colTag != nil {
						columnName = colTag.Params[0]
					} else {
						columnName = session.Engine().ColumnMapper.Obj2Table(hdField)
					}
					omitColumns = append(omitColumns, columnName)
					hiddenDetailFields = append(hiddenDetailFields, hdField)
				}
			}
		}
	}

	// ID作为条件
	pkConds := []string{}
	tableName := modelDefine.TableName(session.Engine())
	for _, pkCol := range pkCols {
		pkConds = append(pkConds, tableName+"."+pkCol+"=?")
	}
	whereClause := strings.Join(pkConds, " AND ")

	pModel := modelVal.Addr().Interface()
	has, err := session.Omit(omitColumns...).Where(whereClause, pk...).Get(pModel)
	if err != nil {
		c.App.Logger.Error("(dbop error): [GetFailed] %s", err.Error())
		return c.Error.New(ErrorInternalError, "GetFailed", modelDefine.MainModelName).SetMessage("Get failed.")
	}
	if !has {
		return c.Error.New(ErrorObjectNotExist, modelDefine.MainModelName)
	}

	var item interface{}
	if len(hiddenDetailFields) > 0 {
		mVals := structs.Map(pModel)
		rVals := make(map[string]interface{})
		for k, v := range mVals {
			if vMap, ok := v.(map[string]interface{}); ok {
				for kk, vv := range vMap {
					rVals[kk] = vv
				}
			} else {
				rVals[k] = v
			}
		}

		for _, hdField := range hiddenDetailFields {
			delete(rVals, hdField)
		}

		item = rVals
	} else {
		item = pModel
	}

	return utils.Combine(modelDefine.MainModelName, item)
}
