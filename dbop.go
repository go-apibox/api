package api

import (
	"errors"
	"fmt"
	"strings"

	"xorm.io/xorm"
)

type queryDefine struct {
	queryType string
	queryArgs []string
}

// 查询定义处理
// 多个定义用 | 分隔，格式：
// 条件类型:参数1,参数2,...|条件类型:参数1,参数2,...
func parseQuerySettings(querySettings map[string]string) map[string][]queryDefine {
	allQueryDefines := make(map[string][]queryDefine)
	for field, querySetting := range querySettings {
		items := strings.Split(querySetting, "|")
		if len(items) == 0 {
			continue
		}

		var queryType string
		var queryArgs []string
		queryDefines := make([]queryDefine, 0, 2)

		for _, item := range items {
			qInfo := strings.SplitN(item, ":", 2)
			queryType = qInfo[0]
			if len(qInfo) == 2 {
				queryArgs = strings.Split(qInfo[1], ",")
			} else {
				queryArgs = []string{}
			}

			switch queryType {
			case "like": // 模糊条件
				if len(queryArgs) != 2 {
					queryArgs = []string{"", "%"}
				} else {
					if queryArgs[0] != "" && queryArgs[0] != "%" {
						queryArgs = []string{"", "%"}
					}
					if queryArgs[1] != "" && queryArgs[1] != "%" {
						queryArgs = []string{"", "%"}
					}
				}
			case "table": // 指定表名
				if len(queryArgs) != 1 {
					//未指定表名，忽略
					continue
				}
			case "expr": // SQL表达式
				if len(queryArgs) == 0 {
					continue
				}
				// 表达式中如果有逗号，则合成一个
				if len(queryArgs) > 1 {
					queryArgs = []string{qInfo[1]}
				}
			}

			queryDefines = append(queryDefines, queryDefine{
				queryType: queryType,
				queryArgs: queryArgs,
			})
		}
		allQueryDefines[field] = queryDefines
	}
	return allQueryDefines
}

func getDB(c *Context) (*xorm.Engine, error) {
	dbType := c.App.Config.GetDefaultString("dbop.db_type", "mysql")
	dbAlias := c.App.Config.GetDefaultString("dbop.db_alias", "default")

	switch dbType {
	case "mysql":
		db, err := c.DB.GetMysql(dbAlias)
		if err != nil {
			return db, err
		}
		keyPrefix := "mysql." + dbAlias + "."
		db.ShowSQL(c.App.Config.GetDefaultBool(keyPrefix+"show_sql", false))
		setLogLevel(db, c.App.Config.GetDefaultString(keyPrefix+"log_level", "error"))
		return db, nil
	case "sqlite3":
		db, err := c.DB.GetSqlite3(dbAlias)
		if err != nil {
			return db, err
		}
		keyPrefix := "sqlite3." + dbAlias + "."
		db.ShowSQL(c.App.Config.GetDefaultBool(keyPrefix+"show_sql", false))
		setLogLevel(db, c.App.Config.GetDefaultString(keyPrefix+"log_level", "error"))
		return db, nil
	default:
		return nil, errors.New("Unknown db engine: " + dbType + ".")
	}
}

func closeDB(c *Context, db *xorm.Engine) error {
	dbType := c.App.Config.GetDefaultString("dbop.db_type", "mysql")

	switch dbType {
	case "mysql":
		return nil
	case "sqlite3":
		return db.Close()
	default:
		return errors.New("Unknown db engine: " + dbType + ".")
	}
}

func buildRangeCond(field string, leftClosed, rightClosed bool) string {
	var leftCond, rightCond string
	if leftClosed {
		leftCond = ">="
	} else {
		leftCond = ">"
	}
	if rightClosed {
		rightCond = "<="
	} else {
		rightCond = "<"
	}
	return fmt.Sprintf("%s%s? AND %s%s?", field, leftCond, field, rightCond)
}
