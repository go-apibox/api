// 数据库管理

package api

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
)

type DbManager struct {
	mutex        sync.Mutex
	mysqlDBMap   map[string]*MySQLDB
	sqlite3DBMap map[string]*Sqlite3DB
}

type MySQLDB struct {
	Engine       *xorm.Engine
	Protocol     string
	Address      string
	User         string
	Password     string
	DBName       string
	MaxIdleConns int
	MaxOpenConns int
	ShowSQL      bool
	LogLevel     string
}

type Sqlite3DB struct {
	Engine     *xorm.Engine
	DBPath     string
	Persistent bool
	ShowSQL    bool
	LogLevel   string
}

// NewDbManager create and return a db manager object.
func NewDbManager() *DbManager {
	return &DbManager{
		mysqlDBMap:   make(map[string]*MySQLDB),
		sqlite3DBMap: make(map[string]*Sqlite3DB),
	}
}

// HasDriver return if sql driver is registered.
func HasDriver(name string) bool {
	dbDrivers := sql.Drivers()
	for _, dbDriver := range dbDrivers {
		if dbDriver == name {
			return true
		}
	}
	return false
}

func setLogLevel(engine *xorm.Engine, level string) {
	switch level {
	case "error":
		engine.Logger().SetLevel(core.LOG_ERR)
	case "warning":
		engine.Logger().SetLevel(core.LOG_WARNING)
	case "info":
		engine.Logger().SetLevel(core.LOG_INFO)
	case "debug":
		engine.Logger().SetLevel(core.LOG_DEBUG)
	case "off":
		engine.Logger().SetLevel(core.LOG_OFF)
	}
}

// AddMysql add a mysql database to db mananger object.
func (dbm *DbManager) AddMysql(dbAlias, protocol, address, user, passwd, dbName string, maxIdleConns, maxOpenConns int) error {
	dbm.mutex.Lock()
	defer dbm.mutex.Unlock()

	if _, has := dbm.mysqlDBMap[dbAlias]; has {
		return errors.New("db alias `" + dbAlias + "` already exists")
	}
	dataSource := user + ":" + passwd + "@" + protocol + "(" + address + ")" + "/" + dbName + "?charset=utf8"
	if engine, err := xorm.NewEngine("mysql", dataSource); err != nil {
		return err
	} else {
		engine.SetMaxIdleConns(maxIdleConns)
		engine.SetMaxOpenConns(maxOpenConns)
		dbm.mysqlDBMap[dbAlias] = &MySQLDB{
			Engine:       engine,
			Protocol:     protocol,
			Address:      address,
			User:         user,
			Password:     passwd,
			DBName:       dbName,
			MaxIdleConns: maxIdleConns,
			MaxOpenConns: maxOpenConns,
			ShowSQL:      false,
			LogLevel:     "off",
		}
		return nil
	}
}

// GetMysql return a mysql db engine.
func (dbm *DbManager) GetMysql(dbAlias string) (*xorm.Engine, error) {
	dbm.mutex.Lock()
	defer dbm.mutex.Unlock()

	if db, has := dbm.mysqlDBMap[dbAlias]; !has {
		return nil, errors.New("db alias `" + dbAlias + "` is not exists")
	} else {
		return db.Engine, nil
	}
}

// SetMysql set a mysql db engine to dbAlias.
func (dbm *DbManager) SetMysql(dbAlias string, engine *xorm.Engine) {
	dbm.mutex.Lock()
	defer dbm.mutex.Unlock()

	if db, has := dbm.mysqlDBMap[dbAlias]; has {
		db.Engine = engine
	}
}

// SetMysqlShowSQL set whether to show sql for the mysql xorm engine.
func (dbm *DbManager) SetMysqlShowSQL(dbAlias string, show bool) {
	dbm.mutex.Lock()
	defer dbm.mutex.Unlock()

	if db, has := dbm.mysqlDBMap[dbAlias]; has {
		db.ShowSQL = show
		db.Engine.ShowSQL(show)
	}
}

// SetMysqlLogLevel set log level for the mysql xorm engine.
func (dbm *DbManager) SetMysqlLogLevel(dbAlias string, level string) {
	dbm.mutex.Lock()
	defer dbm.mutex.Unlock()

	if db, has := dbm.mysqlDBMap[dbAlias]; has {
		db.LogLevel = level
		setLogLevel(db.Engine, level)
	}
}

// RemoveMysql remove a mysql database from db mananger object.
func (dbm *DbManager) RemoveMysql(dbAlias string) {
	dbm.mutex.Lock()
	defer dbm.mutex.Unlock()

	delete(dbm.mysqlDBMap, dbAlias)
}

// AddSqlite3 add a sqlite3 database to db mananger object.
func (dbm *DbManager) AddSqlite3(dbAlias, sqliteDb string, persistent bool) error {
	dbm.mutex.Lock()
	defer dbm.mutex.Unlock()

	if persistent {
		if _, has := dbm.sqlite3DBMap[dbAlias]; has {
			return errors.New("db alias `" + dbAlias + "` already exists")
		}
		if sqliteDb[0] != '/' {
			progDir := filepath.Dir(os.Args[0])
			sqliteDb = filepath.Join(progDir, sqliteDb)
		}
		// if not db directory not exist, create it
		dbDir := filepath.Dir(sqliteDb)
		if _, err := os.Stat(dbDir); err != nil {
			os.MkdirAll(dbDir, 0755)
		}
		if engine, err := xorm.NewEngine("sqlite3", sqliteDb); err != nil && os.IsNotExist(err) {
			return err
		} else {
			dbm.sqlite3DBMap[dbAlias] = &Sqlite3DB{
				Engine:     engine,
				DBPath:     sqliteDb,
				Persistent: persistent,
				ShowSQL:    false,
				LogLevel:   "off",
			}
			return nil
		}
	} else {
		dbm.sqlite3DBMap[dbAlias] = &Sqlite3DB{
			DBPath:     sqliteDb,
			Persistent: persistent,
			ShowSQL:    false,
			LogLevel:   "off",
		}
		return nil
	}
}

// GetSqlite3 return a sqlite3 db engine.
func (dbm *DbManager) GetSqlite3(dbAlias string) (*xorm.Engine, error) {
	dbm.mutex.Lock()
	defer dbm.mutex.Unlock()

	if db, has := dbm.sqlite3DBMap[dbAlias]; !has {
		return nil, errors.New("db alias `" + dbAlias + "` is not exists")
	} else {
		if db.Persistent {
			return db.Engine, nil
		} else {
			sqliteDb := db.DBPath
			if sqliteDb[0] != '/' {
				progDir := filepath.Dir(os.Args[0])
				sqliteDb = filepath.Join(progDir, sqliteDb)
			}
			// if not db directory not exist, create it
			dbDir := filepath.Dir(sqliteDb)
			if _, err := os.Stat(dbDir); err != nil {
				os.MkdirAll(dbDir, 0755)
			}
			if engine, err := xorm.NewEngine("sqlite3", sqliteDb); err != nil && os.IsNotExist(err) {
				return nil, err
			} else {
				engine.ShowSQL(db.ShowSQL)
				setLogLevel(engine, db.LogLevel)
				return engine, nil
			}
		}
	}
}

// SetSqlite3 return a sqlite3 db engine.
func (dbm *DbManager) SetSqlite3(dbAlias string, engine *xorm.Engine) {
	dbm.mutex.Lock()
	defer dbm.mutex.Unlock()

	if db, has := dbm.sqlite3DBMap[dbAlias]; has {
		db.Engine = engine
	}
}

// SetSqlite3ShowSQL set whether to show sql for the sqlite3 xorm engine.
func (dbm *DbManager) SetSqlite3ShowSQL(dbAlias string, show bool) {
	dbm.mutex.Lock()
	defer dbm.mutex.Unlock()

	if db, has := dbm.sqlite3DBMap[dbAlias]; has && db.Engine != nil {
		db.ShowSQL = show
		db.Engine.ShowSQL(show)
	}
}

// SetSqlite3LogLevel set log level for the sqlite3 xorm engine.
func (dbm *DbManager) SetSqlite3LogLevel(dbAlias string, level string) {
	dbm.mutex.Lock()
	defer dbm.mutex.Unlock()

	if db, has := dbm.sqlite3DBMap[dbAlias]; has && db.Engine != nil {
		db.LogLevel = level
	}
}

// RemoveSqlite remove a sqlite3 database from db mananger object.
func (dbm *DbManager) RemoveSqlite3(dbAlias string) {
	dbm.mutex.Lock()
	defer dbm.mutex.Unlock()

	delete(dbm.sqlite3DBMap, dbAlias)
}
