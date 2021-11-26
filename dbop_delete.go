package api

import (
	"reflect"

	"github.com/go-apibox/utils"
	"xorm.io/core"
	"xorm.io/xorm"
)

func Delete(c *Context, bean interface{}, params *Params) interface{} {
	db, err := getDB(c)
	if err != nil {
		c.App.Logger.Error("(dbop error): [DBNotExist] %s", err.Error())
		return c.Error.New(ErrorInternalError, "DBNotExist").SetMessage("Database not exist.")
	}
	defer closeDB(c, db)

	session := db.NewSession()
	defer session.Close()
	return SessionDelete(c, session, bean, params)
}

func SessionDelete(c *Context, session *xorm.Session, bean interface{}, params *Params) interface{} {
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

	for _, pkField := range pkFields {
		if v := params.Get(pkField); v == nil {
			return c.Error.New(ErrorInternalError, "IncompletePKValue").SetMessage("Primary key value is incomplete!")
		} else {
			pk = append(pk, v)
		}
	}

	// 检查是否软删除
	softDelete := false
	fields := modelDefine.TagFields("deletetime")
	if len(fields) > 0 {
		softDelete = true
		for _, field := range fields {
			fieldVal := modelVal.FieldByName(field)
			fieldVal.Set(reflect.ValueOf(utils.Timestamp()))
		}
	}

	// ID作为条件
	var affected int64
	var err error
	pModel := modelVal.Addr().Interface()
	if softDelete {
		affected, err = session.ID(pk).Update(pModel)
	} else {
		affected, err = session.ID(pk).Delete(pModel)
	}
	if err != nil {
		c.App.Logger.Error("(dbop error): [DeleteFailed] %s", err.Error())
		return c.Error.New(ErrorInternalError, "DeleteFailed", modelName).SetMessage("Delete failed.")
	}

	return utils.Combine("Affected", affected)
}
