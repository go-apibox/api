package api

import (
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"

	"git.quyun.com/apibox/filter"
	"git.quyun.com/apibox/utils"
	"github.com/go-xorm/xorm"
)

func Create(c *Context, bean interface{}, params *Params) interface{} {
	return CreateEx(c, bean, params, nil)
}

func CreateEx(c *Context, bean interface{}, params *Params, querySettings map[string]string) interface{} {
	db, err := getDB(c)
	if err != nil {
		c.App.Logger.Error("(dbop error): [DBNotExist] %s", err.Error())
		return c.Error.New(ErrorInternalError, "DBNotExist").SetMessage("Database not exist.")
	}
	defer closeDB(c, db)

	session := db.NewSession()
	defer session.Close()

	return SessionCreateEx(c, session, bean, params, querySettings)
}

func SessionCreate(c *Context, session *xorm.Session, bean interface{}, params *Params) interface{} {
	return SessionCreateEx(c, session, bean, params, nil)
}

func SessionCreateEx(c *Context, session *xorm.Session, bean interface{}, params *Params, querySettings map[string]string) interface{} {
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

	// 查询定义处理
	allQueryDefines := parseQuerySettings(querySettings)

	// 收集需要插入的列（含值的列）
	columns := []string{}

	// 从请求参数中复制值
	fields := modelDefine.Fields()
	randFields := []string{}
	indexColumnTypes := map[string]string{}
	for _, field := range fields {
		// 标记当前字段是否需要插入
		inserted := false

		fieldVal := modelVal.FieldByName(field)
		var columnName string
		if colTag := modelDefine.FieldGetTag(field, "column"); colTag != nil {
			columnName = colTag.Params[0]
		} else {
			columnName = session.Engine().ColumnMapper.Obj2Table(field)
		}

		if v := params.Get(field); v != nil {
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
				inserted = true
			}
		} else {
			// 已有值，则不自动填值
			if fieldVal.Interface() != reflect.Zero(fieldVal.Type()).Interface() {
				continue
			}

			// 检查model的自动填值项
			tags := modelDefine.FieldTags(field)
			for _, tag := range tags {
				switch tag.Name {
				case "rand":
					randFields = append(randFields, field)
					randType := "uint32"
					var vStart, vEnd uint64
					if len(tag.Params) > 0 {
						randType = tag.Params[0]
						randTypeParts := strings.Split(randType, ":")
						if len(randTypeParts) >= 2 {
							randType = randTypeParts[0]
							intRangeParts := strings.Split(randTypeParts[1], "-")
							if len(intRangeParts) == 2 {
								vStart, _ = strconv.ParseUint(intRangeParts[0], 10, 64)
								vEnd, _ = strconv.ParseUint(intRangeParts[1], 10, 64)
								if vEnd <= vStart {
									vEnd = 0
									vStart = 0
								}
							}
						}
					}
					switch randType {
					case "uint":
						inserted = true
						if vEnd > 0 {
							fieldVal.Set(reflect.ValueOf(utils.RandUintIn(uint(vStart), uint(vEnd))))
						} else {
							fieldVal.Set(reflect.ValueOf(utils.RandUint()))
						}
					case "uint32":
						inserted = true
						if vEnd > 0 {
							fieldVal.Set(reflect.ValueOf(utils.RandUint32In(uint32(vStart), uint32(vEnd))))
						} else {
							fieldVal.Set(reflect.ValueOf(utils.RandUint32()))
						}
					case "uint64":
						inserted = true
						if vEnd > 0 {
							fieldVal.Set(reflect.ValueOf(utils.RandUint64In(vStart, vEnd)))
						} else {
							fieldVal.Set(reflect.ValueOf(utils.RandUint64()))
						}
					case "dateprefix":
						inserted = true
						fieldVal.Set(reflect.ValueOf(utils.RandDatePrefixUint64()))
					}

				case "randstr":
					randFields = append(randFields, field)
					strLength := 16
					strCase := ""
					if len(tag.Params) > 0 {
						strLength, _ = strconv.Atoi(tag.Params[0])
						if len(tag.Params) > 1 {
							strCase = tag.Params[1]
						}
					}
					inserted = true
					v := utils.RandStringN(strLength)
					if strCase == "upper" {
						v = strings.ToUpper(v)
					} else if strCase == "lower" {
						v = strings.ToLower(v)
					}
					fieldVal.Set(reflect.ValueOf(v))

				case "createtime":
					inserted = true
					fieldVal.Set(reflect.ValueOf(utils.Timestamp()))

				case "updatetime":
					inserted = true
					fieldVal.Set(reflect.ValueOf(utils.Timestamp()))

				case "showindex":
					// 需要初始化为0，否则sqlite3下该字段为NOT NULL时会插入报错
					inserted = true
					fieldVal.Set(reflect.ValueOf(uint32(0)))

					indexType := "append"
					if len(tag.Params) > 0 {
						indexType = tag.Params[0]
					}
					indexColumnTypes[columnName] = indexType
				}
			}
		}

		if inserted {
			columns = append(columns, columnName)
		}
	}

	// 如果有多条语句要处理，则启用事务
	enableTrans := false
	if len(indexColumnTypes) > 0 {
		// 如果外部已经启动事务，内部不再启动
		if session.Tx == nil {
			enableTrans = true
		}
	}
	if enableTrans {
		session.Begin()
	}

	// retry 3 times if primary key duplicate
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		_, err := session.Cols(columns...).Insert(modelPt)
		if err == nil {
			break
		}

		// Error 1062: Duplicate entry '3283486027' for key 'PRIMARY'
		errStr := strings.ToLower(err.Error())
		if strings.Index(errStr, "duplicate") > 0 &&
			strings.Index(errStr, "primary") > 0 &&
			i != maxRetries-1 {
			if len(randFields) > 0 {
				// regenerate
				for _, field := range randFields {
					fieldVal := modelVal.FieldByName(field)
					// 检查model的自动填值项
					tags := modelDefine.FieldTags(field)
					for _, tag := range tags {
						switch tag.Name {
						case "rand":
							randType := "uint32"
							var vStart, vEnd uint64
							if len(tag.Params) > 0 {
								randType = tag.Params[0]
								randTypeParts := strings.Split(randType, ":")
								if len(randTypeParts) >= 2 {
									randType = randTypeParts[0]
									intRangeParts := strings.Split(randTypeParts[1], "-")
									if len(intRangeParts) == 2 {
										vStart, _ = strconv.ParseUint(intRangeParts[0], 10, 64)
										vEnd, _ = strconv.ParseUint(intRangeParts[1], 10, 64)
										if vEnd <= vStart {
											vEnd = 0
											vStart = 0
										}
									}
								}
							}
							switch randType {
							case "uint":
								if vEnd > 0 {
									fieldVal.Set(reflect.ValueOf(utils.RandUintIn(uint(vStart), uint(vEnd))))
								} else {
									fieldVal.Set(reflect.ValueOf(utils.RandUint()))
								}
							case "uint32":
								if vEnd > 0 {
									fieldVal.Set(reflect.ValueOf(utils.RandUint32In(uint32(vStart), uint32(vEnd))))
								} else {
									fieldVal.Set(reflect.ValueOf(utils.RandUint32()))
								}
							case "uint64":
								if vEnd > 0 {
									fieldVal.Set(reflect.ValueOf(utils.RandUint64In(vStart, vEnd)))
								} else {
									fieldVal.Set(reflect.ValueOf(utils.RandUint64()))
								}
							case "dateprefix":
								fieldVal.Set(reflect.ValueOf(utils.RandDatePrefixUint64()))
							}

						case "randstr":
							strLength := 16
							strCase := ""
							if len(tag.Params) > 0 {
								strLength, _ = strconv.Atoi(tag.Params[0])
								if len(tag.Params) > 1 {
									strCase = tag.Params[1]
								}
							}
							v := utils.RandStringN(strLength)
							if strCase == "upper" {
								v = strings.ToUpper(v)
							} else if strCase == "lower" {
								v = strings.ToLower(v)
							}
							fieldVal.Set(reflect.ValueOf(v))
						}
					}
				}
				continue
			}
		}

		if enableTrans {
			session.Rollback()
		}
		c.App.Logger.Error("(dbop error): [InsertFailed] %s", err.Error())
		return c.Error.New(ErrorInternalError, "InsertFailed", modelName).SetMessage("Insert failed.")
	}

	// 返回ID
	pkFields := modelDefine.TagFields("pk")
	if len(pkFields) == 0 {
		return nil
	}

	var last_insert_id interface{}
	rt := make(map[string]interface{})
	for _, pkField := range pkFields {
		last_insert_id = modelVal.FieldByName(pkField).Interface()
		rt[pkField] = last_insert_id
	}

	if len(indexColumnTypes) > 0 {
		// 处理排序序号
		for columnName, indexType := range indexColumnTypes {
			tableName := modelDefine.TableName(session.Engine())
			switch indexType {
			case "insert":
				_, err := session.Exec(fmt.Sprintf("UPDATE `%s` SET `%s`=`%s`+1", tableName, columnName, columnName))
				if err != nil {
					if enableTrans {
						session.Rollback()
					}
					c.App.Logger.Error("(dbop error): [UpdateShowIndexFailed] %s", err.Error())
					return c.Error.New(ErrorInternalError, "UpdateShowIndexFailed").SetMessage("Update ShowIndex Failed.")
				}
			case "append":
				var pkColumnName string
				if colTag := modelDefine.FieldGetTag(pkFields[0], "column"); colTag != nil {
					pkColumnName = colTag.Params[0]
				} else {
					pkColumnName = session.Engine().ColumnMapper.Obj2Table(pkFields[0])
				}
				sql := fmt.Sprintf("UPDATE `%s` SET `%s`=? WHERE `%s`=?", tableName, columnName, pkColumnName)
				_, err := session.Exec(sql, last_insert_id, last_insert_id)
				if err != nil {
					if enableTrans {
						session.Rollback()
					}
					c.App.Logger.Error("(dbop error): [UpdateShowIndexFailed] %s", err.Error())
					return c.Error.New(ErrorInternalError, "UpdateShowIndexFailed").SetMessage("Update ShowIndex Failed.")
				}
			}
		}
	}

	if enableTrans {
		err := session.Commit()
		if err != nil {
			c.App.Logger.Error("(dbop error): [SessionCommitFailed] %s", err.Error())
			return c.Error.New(ErrorInternalError, "SessionCommitFailed").SetMessage("Session Commit Failed.")
		}
	}

	return rt
}
