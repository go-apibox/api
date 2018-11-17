package api

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-apibox/types"
	"github.com/fatih/structs"
	"github.com/go-xorm/xorm"
)

func List(c *Context, beans interface{}, params *Params, queryDefines map[string]string) interface{} {
	return ListJoin(c, beans, params, queryDefines, nil)
}

func SessionList(c *Context, session *xorm.Session, beans interface{}, params *Params, querySettings map[string]string) interface{} {
	return SessionListJoin(c, session, beans, params, querySettings, nil)
}

func ListJoin(c *Context, beans interface{}, params *Params, querySettings map[string]string, joinConds [][]string) interface{} {
	db, err := getDB(c)
	if err != nil {
		c.App.Logger.Error("(dbop error): [DBNotExist] %s", err.Error())
		return c.Error.New(ErrorInternalError, "DBNotExist").SetMessage("Database not exist.")
	}
	defer closeDB(c, db)

	session := db.NewSession()
	defer session.Close()

	return SessionListJoin(c, session, beans, params, querySettings, joinConds)
}

func SessionListJoin(c *Context, session *xorm.Session, beans interface{}, params *Params, querySettings map[string]string, joinConds [][]string) interface{} {
	countSession := session.Engine().NewSession()
	defer countSession.Close()
	findSession := session.Engine().NewSession()
	defer findSession.Close()

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
			countSession.Join(joinCond[0], condTable, joinCond[2])
			findSession.Join(joinCond[0], condTable, joinCond[2])
		}
	}

	sliceValue := reflect.Indirect(reflect.ValueOf(beans))
	if sliceValue.Kind() != reflect.Slice {
		return c.Error.New(ErrorInternalError, "WrongParamType").SetMessage("Expected a pointer to a slice!")
	}

	// 从请求参数中构建查询条件
	sliceElemType := sliceValue.Type().Elem()
	if sliceElemType.Kind() == reflect.Ptr {
		sliceElemType = sliceElemType.Elem()
	}
	modelName := sliceElemType.Name()
	modelDefine := c.Model.Get(modelName)
	if modelDefine == nil {
		return c.Error.New(ErrorInternalError, "ModelNotRegistered").SetMessage("Model " + modelName + " not registered!")
	}

	// 查询定义处理
	allQueryDefines := parseQuerySettings(querySettings)
	tableDefines, has := allQueryDefines[":table:"]
	if has && len(tableDefines) > 0 {
		for _, qDefine := range tableDefines {
			if qDefine.queryType == "table" && len(qDefine.queryArgs) == 1 {
				tableName := qDefine.queryArgs[0]
				countSession = countSession.Table(tableName)
				findSession = findSession.Table(tableName)
				delete(allQueryDefines, ":table:")
				break
			}
		}
	}

	// or 查询类型处理
	// 复制参数值到所有 or 字段
	orFieldGroups := [][]string{}
	for field, queryDefines := range allQueryDefines {
		for _, qDefine := range queryDefines {
			if qDefine.queryType == "or" {
				if params.Has(field) {
					paramVal := params.Get(field)
					orFieldGroups = append(orFieldGroups, qDefine.queryArgs)
					for _, v := range qDefine.queryArgs {
						params.Set(v, paramVal)
					}
				}
			}
		}
	}

	// 查询条件处理
	conds := map[string]string{}
	condArgs := map[string][]interface{}{}
	allFields := modelDefine.Fields()
	for _, field := range allFields {
		if params.Has(field) {
			var columnName string
			if colTag := modelDefine.FieldGetTag(field, "column"); colTag != nil {
				columnName = colTag.Params[0]
			} else {
				columnName = session.Engine().ColumnMapper.Obj2Table(field)
			}
			tableField := fmt.Sprintf("`%s`", columnName)

			qDefines, qDefined := allQueryDefines[field]

			// 字段表名前缀处理
			for _, qDefine := range qDefines {
				if qDefine.queryType == "table" {
					tableField = fmt.Sprintf("`%s`.%s", qDefine.queryArgs[0], tableField)
				}
			}

			paramValue := params.Get(field)
			switch v := paramValue.(type) {
			case []int:
				if len(v) > 0 {
					countSession.In(tableField, v)
					findSession.In(tableField, v)
				} else {
					// IN ()语法错误，需要特殊处理
					countSession.Where("1=0")
				}
			case []int32:
				if len(v) > 0 {
					countSession.In(tableField, v)
					findSession.In(tableField, v)
				} else {
					countSession.Where("1=0")
				}
			case []int64:
				if len(v) > 0 {
					countSession.In(tableField, v)
					findSession.In(tableField, v)
				} else {
					countSession.Where("1=0")
				}
			case []uint:
				if len(v) > 0 {
					countSession.In(tableField, v)
					findSession.In(tableField, v)
				} else {
					countSession.Where("1=0")
				}
			case []uint32:
				if len(v) > 0 {
					countSession.In(tableField, v)
					findSession.In(tableField, v)
				} else {
					countSession.Where("1=0")
				}
			case []uint64:
				if len(v) > 0 {
					countSession.In(tableField, v)
					findSession.In(tableField, v)
				} else {
					countSession.Where("1=0")
				}
			case []string:
				if len(v) > 0 {
					countSession.In(tableField, v)
					findSession.In(tableField, v)
				} else {
					countSession.Where("1=0")
				}
			case *types.IntRange:
				conds[field] = buildRangeCond(tableField, v.LeftClosed, v.RightClosed)
				condArgs[field] = []interface{}{v.Left, v.Right}
			case types.IntRange:
				conds[field] = buildRangeCond(tableField, v.LeftClosed, v.RightClosed)
				condArgs[field] = []interface{}{v.Left, v.Right}
			case *types.Int32Range:
				conds[field] = buildRangeCond(tableField, v.LeftClosed, v.RightClosed)
				condArgs[field] = []interface{}{v.Left, v.Right}
			case types.Int32Range:
				conds[field] = buildRangeCond(tableField, v.LeftClosed, v.RightClosed)
				condArgs[field] = []interface{}{v.Left, v.Right}
			case *types.Int64Range:
				conds[field] = buildRangeCond(tableField, v.LeftClosed, v.RightClosed)
				condArgs[field] = []interface{}{v.Left, v.Right}
			case types.Int64Range:
				conds[field] = buildRangeCond(tableField, v.LeftClosed, v.RightClosed)
				condArgs[field] = []interface{}{v.Left, v.Right}
			case *types.UintRange:
				conds[field] = buildRangeCond(tableField, v.LeftClosed, v.RightClosed)
				condArgs[field] = []interface{}{v.Left, v.Right}
			case types.UintRange:
				conds[field] = buildRangeCond(tableField, v.LeftClosed, v.RightClosed)
				condArgs[field] = []interface{}{v.Left, v.Right}
			case *types.Uint32Range:
				conds[field] = buildRangeCond(tableField, v.LeftClosed, v.RightClosed)
				condArgs[field] = []interface{}{v.Left, v.Right}
			case types.Uint32Range:
				conds[field] = buildRangeCond(tableField, v.LeftClosed, v.RightClosed)
				condArgs[field] = []interface{}{v.Left, v.Right}
			case *types.Uint64Range:
				conds[field] = buildRangeCond(tableField, v.LeftClosed, v.RightClosed)
				condArgs[field] = []interface{}{v.Left, v.Right}
			case types.Uint64Range:
				conds[field] = buildRangeCond(tableField, v.LeftClosed, v.RightClosed)
				condArgs[field] = []interface{}{v.Left, v.Right}
			case *types.TimestampRange:
				conds[field] = buildRangeCond(tableField, v.LeftClosed, v.RightClosed)
				condArgs[field] = []interface{}{v.Left, v.Right}
			case types.TimestampRange:
				conds[field] = buildRangeCond(tableField, v.LeftClosed, v.RightClosed)
				condArgs[field] = []interface{}{v.Left, v.Right}
			case *types.TimeRange:
				conds[field] = buildRangeCond(tableField, v.LeftClosed, v.RightClosed)
				condArgs[field] = []interface{}{v.Left.Unix(), v.Right.Unix()}
			case types.TimeRange:
				conds[field] = buildRangeCond(tableField, v.LeftClosed, v.RightClosed)
				condArgs[field] = []interface{}{v.Left.Unix(), v.Right.Unix()}
			case string:
				var sqlSlause string
				var sqlValue string
				if qDefined {
					for _, qDefine := range qDefines {
						switch qDefine.queryType {
						case "like":
							sqlSlause = fmt.Sprintf("%s LIKE ?", tableField)
							sqlValue = qDefine.queryArgs[0] + v + qDefine.queryArgs[1]
							break
						}
					}
				}
				if sqlSlause == "" {
					sqlSlause = fmt.Sprintf("%s=?", tableField)
					sqlValue = v
				}
				conds[field] = sqlSlause
				condArgs[field] = []interface{}{sqlValue}
			default:
				conds[field] = fmt.Sprintf("%s=?", tableField)
				condArgs[field] = []interface{}{v}
			}
		}
	}

	var queryString string
	var queryArgs []interface{}

	// 拼装查询字符串与条件参数值
	// 先拼装 or 语句
	sqlClauses := make([]string, 0, len(conds))
	for _, orFields := range orFieldGroups {
		subSqlClauses := make([]string, 0, len(orFields))
		for _, orField := range orFields {
			if condClause, has := conds[orField]; has {
				subSqlClauses = append(subSqlClauses, condClause)
				queryArgs = append(queryArgs, condArgs[orField]...)
				delete(conds, orField)
				delete(condArgs, orField)
			}
		}
		subQuery := fmt.Sprintf("(%s)", strings.Join(subSqlClauses, " OR "))
		sqlClauses = append(sqlClauses, subQuery)
	}

	// 拼装剩余
	for field, v := range conds {
		sqlClauses = append(sqlClauses, v)
		queryArgs = append(queryArgs, condArgs[field]...)
	}
	queryString = strings.Join(sqlClauses, " AND ")

	// 总数
	pModel := reflect.New(sliceElemType).Interface()
	if queryString != "" {
		countSession.Where(queryString, queryArgs...)
	}
	totalCount, err := countSession.Count(pModel)
	if err != nil {
		c.App.Logger.Error("(dbop error): [CountFailed] %s", err.Error())
		return c.Error.New(ErrorInternalError, "CountFailed", modelName).SetMessage("Count failed.")
	}

	// 记录顺序控制相关变量
	var showIndexField string
	var morePreCount, moreNextCount int

	result := make(map[string]interface{})
	pageNumber, pageSize := 1, 10
	if params.Has("_pageSize") {
		pageSize = params.GetInt("_pageSize")
		if pageSize < 1 || pageSize > 1000 {
			pageSize = 10
		}
	}
	var items interface{}
	var pageCount int64

	if totalCount > 0 {
		// 要隐藏的字段
		omitColumns := []string{}
		hiddenListFields := []string{}
		hiddenFields := modelDefine.TagFields("hidden")
		for _, hdField := range hiddenFields {
			tags := modelDefine.FieldTags(hdField)
			for _, tag := range tags {
				if tag.Name == "hidden" {
					if len(tag.Params) == 0 || tag.Params[0] == "*" || tag.Params[0] == "list" {
						var columnName string
						if colTag := modelDefine.FieldGetTag(hdField, "column"); colTag != nil {
							columnName = colTag.Params[0]
						} else {
							columnName = session.Engine().ColumnMapper.Obj2Table(hdField)
						}
						omitColumns = append(omitColumns, columnName)
						hiddenListFields = append(hiddenListFields, hdField)
					}
				}
			}
		}
		findSession.Omit(omitColumns...)

		// 排序
		orderBys := []string{}
		orders := []string{}
		if params.Has("_orderBy") {
			orderBys = params.GetStringArray("_orderBy")
			for i, v := range orderBys {
				var tableField string
				if colTag := modelDefine.FieldGetTag(v, "column"); colTag != nil {
					tableField = colTag.Params[0]
				} else {
					tableField = session.Engine().ColumnMapper.Obj2Table(v)
				}

				// 字段表名前缀处理
				if qDefines, ok := allQueryDefines[v]; ok {
					for _, qDefine := range qDefines {
						if qDefine.queryType == "table" {
							tableField = fmt.Sprintf("`%s`.%s", qDefine.queryArgs[0], tableField)
						}
					}
				}

				orderBys[i] = tableField
			}
		}
		if params.Has("_order") {
			orders = params.GetStringArray("_order")
		}
		orderCount := len(orders)
		for i, orderBy := range orderBys {
			var order string
			if i < orderCount {
				order = orders[i]
			} else {
				order = "asc"
			}
			if order == "asc" {
				findSession.Asc(orderBy)
			} else {
				findSession.Desc(orderBy)
			}
		}

		// 检测是否存在showIndex字段
		showIndexFields := modelDefine.TagFields("showindex")
		if len(showIndexFields) > 0 {
			showIndexField = showIndexFields[0]
		}

		// 分页
		if params.Has("_pageNumber") {
			pageNumber = params.GetInt("_pageNumber")
			if pageNumber < 1 {
				pageNumber = pageNumber
			}
		}
		// if params.Has("_pageSize") {
		// 	pageSize = params.GetInt("_pageSize")
		// 	if pageSize < 1 || pageSize > 1000 {
		// 		pageSize = 10
		// 	}
		// }
		pageCount = int64((totalCount + int64(pageSize) - 1) / int64(pageSize))
		if showIndexField == "" {
			findSession.Limit(pageSize, (pageNumber-1)*pageSize)
		} else {
			// 多返回两条数据（上一条&下一条）
			offset := (pageNumber - 1) * pageSize
			if pageNumber > 1 && int64(pageNumber) <= pageCount {
				// 非第一页，有上一条
				morePreCount = 1
			}
			if int64(pageNumber) < pageCount {
				// 非最后一页，有下一条
				moreNextCount = 1
			}
			findSession.Limit(pageSize+morePreCount+moreNextCount, offset-morePreCount)
		}

		// 获取结果
		err = findSession.Where(queryString, queryArgs...).Find(beans)
		if err != nil {
			c.App.Logger.Error("(dbop error): [FindFailed] %s", err.Error())
			return c.Error.New(ErrorInternalError, "FindFailed", modelName).SetMessage("Find failed.")
		}

		modelVals := reflect.Indirect(reflect.ValueOf(beans))

		// 在返回结果中带上上一页最后一个showindex和下一页第一个showindex
		if showIndexField != "" {
			indexInfo := map[string]interface{}{"pre": -1, "next": -1}
			if morePreCount > 0 {
				firstVal := modelVals.Index(0)
				if firstVal.Kind() == reflect.Ptr {
					firstVal = firstVal.Elem()
				}
				indexInfo["pre"] = firstVal.FieldByName(showIndexField).Interface()
				modelVals.Set(modelVals.Slice(1, modelVals.Len()))
			}
			if moreNextCount > 0 {
				lasti := modelVals.Len() - 1
				lastVal := modelVals.Index(lasti)
				if lastVal.Kind() == reflect.Ptr {
					lastVal = lastVal.Elem()
				}
				indexInfo["next"] = lastVal.FieldByName(showIndexField).Interface()
				modelVals.Set(modelVals.Slice(0, lasti))
			}
			result["ShowIndex"] = indexInfo
		}

		if len(hiddenListFields) > 0 {
			// 除去要隐藏的字段
			msVals := make([]map[string]interface{}, 0, modelVals.Len())
			for i := 0; i < modelVals.Len(); i++ {
				mVals := structs.Map(modelVals.Index(i).Interface())
				rVals := make(map[string]interface{})

				// 提取出匿名结构中的字段
				for k, v := range mVals {
					if vMap, ok := v.(map[string]interface{}); ok {
						for kk, vv := range vMap {
							rVals[kk] = vv
						}
					} else {
						rVals[k] = v
					}
				}

				for _, hdField := range hiddenListFields {
					delete(rVals, hdField)
				}
				msVals = append(msVals, rVals)
			}
			items = msVals
		} else {
			items = modelVals.Interface()
		}
	} else {
		items = []map[string]interface{}{}
	}

	result["PageNumber"] = pageNumber
	result["PageSize"] = pageSize
	result["TotalCount"] = totalCount
	result["PageCount"] = pageCount
	result[modelDefine.MainModelName+"List"] = items

	return result
}
