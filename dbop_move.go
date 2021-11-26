package api

import (
	"fmt"
	"reflect"

	"xorm.io/xorm"
)

func Move(c *Context, bean interface{}, srcIndex, dstIndex uint32) interface{} {
	db, err := getDB(c)
	if err != nil {
		c.App.Logger.Error("(dbop error): [DBNotExist] %s", err.Error())
		return c.Error.New(ErrorInternalError, "DBNotExist").SetMessage("Database not exist.")
	}
	defer closeDB(c, db)

	session := db.NewSession()
	defer session.Close()

	return SessionMove(c, session, bean, srcIndex, dstIndex)
}

func SessionMove(c *Context, session *xorm.Session, bean interface{}, srcIndex, dstIndex uint32) interface{} {
	if srcIndex == dstIndex {
		return c.Error.New(ErrorInternalError, "NoObjectMoved").SetMessage("No object moved!")
	}

	var modelName string
	var modelVal reflect.Value
	var modelPt interface{}

	modelName, isString := bean.(string)
	if !isString {
		modelVal = reflect.Indirect(reflect.ValueOf(bean))
		if modelVal.Kind() != reflect.Struct {
			return c.Error.New(ErrorInternalError, "WrongParamType").SetMessage("Expected a pointer to a struct!")
		}
		modelName = modelVal.Type().Name()
		modelPt = bean
	}

	modelDefine := c.Model.Get(modelName)
	if modelDefine == nil {
		return c.Error.New(ErrorInternalError, "ModelNotRegistered").SetMessage("Model " + modelName + " not registered!")
	}
	if isString {
		t := reflect.New(modelDefine.Type)
		modelPt = t.Interface()
		modelVal = reflect.Indirect(t)
	}

	// 查询表名和排序字段名
	showIndexFields := modelDefine.TagFields("showindex")
	if len(showIndexFields) == 0 {
		return c.Error.New(ErrorInternalError, "NoShowIndexField").SetMessage("No ShowIndex Field.")
	}
	showIndexField := showIndexFields[0]
	var columnName string
	if colTag := modelDefine.FieldGetTag(showIndexField, "column"); colTag != nil {
		columnName = colTag.Params[0]
	} else {
		columnName = session.Engine().GetColumnMapper().Obj2Table(showIndexField)
	}
	tableName := modelDefine.TableName(session.Engine())

	// 检查源和目标是否存在
	total, err := session.In(columnName, []uint32{srcIndex, dstIndex}).Count(modelPt)
	if err != nil {
		c.App.Logger.Error("(dbop error): [CountFailed] %s", err.Error())
		return c.Error.New(ErrorInternalError, "CountFailed").SetMessage("Count Failed.")
	}
	if total != 2 {
		return c.Error.New(ErrorInternalError, "ObjectNotExists").SetMessage("One of the object does not exist.")
	}

	var otherOffset int
	var indexFrom, indexTo uint32
	if dstIndex < srcIndex {
		// 上移
		otherOffset = 1
		indexFrom = dstIndex
		indexTo = srcIndex
	} else {
		// 下移
		otherOffset = -1
		indexFrom = srcIndex
		indexTo = dstIndex
	}

	var sql string
	if session.Engine().DriverName() == "mysql" {
		sql = fmt.Sprintf(
			"UPDATE `%s` SET `%s`=IF(`%s`=%d, %d, `%s`+%d) WHERE `%s`>=%d AND `%s`<=%d",
			tableName, columnName, columnName, srcIndex, dstIndex, columnName, otherOffset,
			columnName, indexFrom, columnName, indexTo,
		)
	} else {
		// sqlite3不支持IF语句
		sql = fmt.Sprintf(
			"UPDATE `%s` SET `%s`=(CASE WHEN `%s`=%d THEN %d ELSE `%s`+%d END) WHERE `%s`>=%d AND `%s`<=%d",
			tableName, columnName, columnName, srcIndex, dstIndex, columnName, otherOffset,
			columnName, indexFrom, columnName, indexTo,
		)
	}
	if _, err := session.Exec(sql); err != nil {
		c.App.Logger.Error("(dbop error): [UpdateShowIndexFailed] %s", err.Error())
		return c.Error.New(ErrorInternalError, "UpdateShowIndexFailed").SetMessage("Update ShowIndex Failed.")
	}

	return nil
}
