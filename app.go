// 应用

package api

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/go-apibox/config"
	"github.com/go-apibox/filter"
	"github.com/go-apibox/logging"
	"github.com/go-apibox/session"
	"github.com/go-apibox/utils"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/urfave/negroni"
)

type App struct {
	Name                string
	Host                string
	Addr                string
	ServerName          string
	Router              *mux.Router
	Config              *config.Config
	DB                  *DbManager
	Model               *ModelManager
	Error               *ErrorManager
	Logger              *Logger
	Routes              []*Route
	Hooks               map[string][]ActionFunc
	UnderMaintenance    bool
	Middlewares         map[string]negroni.Handler
	middlewareNames     []string //确保顺序
	listenEventHandlers []ListenEventHandler
	sockClosed          chan bool
	HTTPServer          *http.Server
}

// NewApp create an application with config file: config/app.yaml.
func NewApp() (*App, error) {
	return NewAppFromFile("config/app.yaml")
}

// NewAppFromFile create an application from config file.
func NewAppFromFile(configFile string) (*App, error) {
	// 读取配置
	progDir := filepath.Dir(os.Args[0])
	configFile = filepath.Join(progDir, configFile)
	cfg, err := config.FromFile(configFile)
	if err != nil {
		return nil, err
	}
	return newApp(cfg), nil
}

// NewAppFromYaml create an application from specified yaml config string.
func NewAppFromYaml(yamlStr string) (*App, error) {
	// 新建配置
	cfg, err := config.FromString(yamlStr)
	if err != nil {
		return nil, err
	}
	return newApp(cfg), nil
}

func newApp(cfg *config.Config) *App {
	dbm := NewDbManager()

	em := NewErrorManager()
	em.SetLang(cfg.GetDefaultString("api.default.lang", "en_us"))

	appName := cfg.GetDefaultString("app.name", "app")
	logger := NewLogger(appName)
	actionWhitelist := cfg.GetDefaultStringArray("api.log.actions.whitelist", []string{"*"})
	actionBlacklist := cfg.GetDefaultStringArray("api.log.actions.blacklist", []string{})
	logger.SetWhiteList(actionWhitelist)
	logger.SetBlackList(actionBlacklist)

	router := mux.NewRouter()

	app := &App{
		Name:                appName,
		ServerName:          "apibox/" + appName,
		Router:              router,
		Config:              cfg,
		DB:                  dbm,
		Model:               NewModelManager(),
		Error:               em,
		Logger:              logger,
		Hooks:               make(map[string][]ActionFunc),
		UnderMaintenance:    false,
		Middlewares:         map[string]negroni.Handler{},
		middlewareNames:     []string{},
		listenEventHandlers: []ListenEventHandler{},
		sockClosed:          nil,
	}

	// 加载各模块
	app.InitDb()
	app.InitError()

	// host 和 addr 初始化
	host := os.Getenv("APP_HOST")
	if host == "" {
		host = cfg.GetDefaultString("app.host", "")
	} else {
		os.Setenv("APP_HOST", "") // 读取完立即清空，防止被子进程继承
	}
	app.Host = host
	addr := os.Getenv("APP_ADDR")
	if addr == "" {
		addr = cfg.GetDefaultString("app.http_addr", ":80")
		if strings.HasPrefix(addr, "*") {
			addr = addr[1:]
		}
	} else {
		os.Setenv("APP_ADDR", "") // 读取完立即清空，防止被子进程继承
	}
	if strings.IndexByte(addr, ':') == -1 {
		// unix domain socket

		// remove unix domain socket file
		if fi, err := os.Stat(addr); err == nil {
			if fi.Mode()&os.ModeType == os.ModeSocket {
				os.Remove(addr)
			}
		}

		if !filepath.IsAbs(addr) {
			progDir := filepath.Dir(os.Args[0])
			addr = filepath.Join(progDir, addr)
		}
		// auto create addr parent directory if not exist
		addrDir := filepath.Dir(addr)
		if _, err := os.Stat(addrDir); err != nil && os.IsNotExist(err) {
			os.MkdirAll(addrDir, 0755)
		}
	}
	app.Addr = addr

	return app
}

// Use will run middleware around the request handler.
func (app *App) Use(middlewares ...negroni.Handler) {
	for _, m := range middlewares {
		t := reflect.TypeOf(m).String()
		mName := t[1:strings.Index(t, ".")]
		if _, has := app.Middlewares[mName]; !has {
			app.Middlewares[mName] = m
			app.middlewareNames = append(app.middlewareNames, mName)
		}
	}
}

// UseBefore will insert a middleware before specified middleware, and run middleware around the request handler.
func (app *App) UseBefore(refMwName string, middleware negroni.Handler) {
	t := reflect.TypeOf(middleware).String()
	mName := t[1:strings.Index(t, ".")]

	if _, has := app.Middlewares[mName]; has {
		return
	}

	insertPos := -1
	for i, mName := range app.middlewareNames {
		if mName == refMwName {
			insertPos = i
			break
		}
	}

	app.Middlewares[mName] = middleware
	if insertPos == -1 {
		// reference middleware not found, then append to the end
		app.middlewareNames = append(app.middlewareNames, mName)
	} else {
		app.middlewareNames = append(app.middlewareNames[:insertPos], append([]string{mName}, app.middlewareNames[insertPos:]...)...)
	}
}

// UseNamed will run a named middleware around the request handler.
func (app *App) UseNamed(mName string, middleware negroni.Handler) {
	if _, has := app.Middlewares[mName]; has {
		return
	}

	app.Middlewares[mName] = middleware
	app.middlewareNames = append(app.middlewareNames, mName)
}

// UseNamedBefore will insert a named middleware before specified middleware, and run middleware around the request handler.
func (app *App) UseNamedBefore(refMwName string, mName string, middleware negroni.Handler) {
	if _, has := app.Middlewares[mName]; has {
		return
	}

	insertPos := -1
	for i, mName := range app.middlewareNames {
		if mName == refMwName {
			insertPos = i
			break
		}
	}

	app.Middlewares[mName] = middleware
	if insertPos == -1 {
		// reference middleware not found, then append to the end
		app.middlewareNames = append(app.middlewareNames, mName)
	} else {
		app.middlewareNames = append(app.middlewareNames[:insertPos], append([]string{mName}, app.middlewareNames[insertPos:]...)...)
	}
}

// SetRoutes set the routes of application.
func (app *App) SetRoutes(routes []*Route) {
	app.Routes = routes
}

// Run will start a http server to run your application.
func (app *App) Run(routes []*Route) error {
	if routes == nil {
		routes = app.Routes
	} else {
		app.Routes = routes
	}

	cfg := app.Config
	apiPath := cfg.GetDefaultString("api.path", "/")

	// 注册中间件
	rec := negroni.NewRecovery()
	rec.PrintStack = false
	n := negroni.New(rec, app.Logger)
	for _, mName := range app.middlewareNames {
		n.Use(app.Middlewares[mName])
	}
	// 自动加RequestId
	n.Use(NewRequestIdMaker())

	// 注册http handler
	apiMux := http.NewServeMux()
	apiMux.HandleFunc(apiPath, app.Route(routes))
	n.UseHandler(apiMux)

	router := app.Router
	if app.Host != "" {
		router.Host(app.Host).Subrouter().Handle(apiPath, context.ClearHandler(n))
	} else {
		router.Handle(apiPath, context.ClearHandler(n))
	}

	// 运行
	app.Logger.Noticef("listening on %s", app.Addr)

	// 是否禁止应用运行为pid=1（根进程）的子进程
	var allowWild bool
	envAllowWild := os.Getenv("APP_ALLOW_WILD")
	if envAllowWild != "" {
		os.Setenv("APP_ALLOW_WILD", "") // 读取完立即清空，防止被子进程继承
	}
	switch envAllowWild {
	case "0":
		allowWild = false
	case "1":
		allowWild = true
	default:
		allowWild = cfg.GetDefaultBool("app.allow_wild", false)
	}

	if !allowWild {
		// 定时检测进程是否为野进程
		go func() {
			for {
				time.Sleep(time.Second)
				if os.Getppid() == 1 {
					app.Logger.Fatal("wild process killed")
				}
			}
		}()
	}

	// 判断是否unix
	isUnix := strings.IndexByte(app.Addr, ':') == -1
	tlsEnabled := cfg.GetDefaultBool("app.tls.enabled", false)
	var certPemBlock, keyPemBlock []byte

	if tlsEnabled {
		certFile := cfg.GetDefaultString("app.tls.cert", "server.crt")
		keyFile := cfg.GetDefaultString("app.tls.key", "server.key")

		var err error
		var autoGenCert bool
		if certFile == "" || keyFile == "" {
			autoGenCert = true
		} else if utils.FileExists(certFile) && utils.FileExists(certFile) {
			certPemBlock, keyPemBlock, err = loadX509PemBlock(certFile, keyFile)
			if err != nil {
				// 解析失败，不停止服务，转而使用自动生成证书
				app.Logger.Error(err.Error())
				autoGenCert = true
			} else {
				autoGenCert = false
			}
		} else {
			autoGenCert = true
		}

		if autoGenCert {
			// 自动生成证书
			var bindIp string
			if isUnix {
				bindIp = "127.0.0.1"
			} else {
				bindIp = app.Addr[0:strings.IndexByte(app.Addr, ':')]
			}
			certPemBlock, keyPemBlock, err = makeCert(app.Host, bindIp)
			if err != nil {
				app.Logger.Critical(err.Error())
				return err
			}
		}
	}

	var appErr error
	if isUnix {
		// unix domain socket
		if tlsEnabled {
			appErr = app.serveTLSWithPemBlock("unix", app.Addr, router, certPemBlock, keyPemBlock)
		} else {
			appErr = app.serve("unix", app.Addr, router)
		}
		if opErr, ok := appErr.(*net.OpError); ok {
			if opErr.Op == "accept" {
				// accept 失败，表示socket已断开
				appErr = nil
			}
		}
	} else {
		// tcp socket
		if tlsEnabled {
			appErr = app.serveTLSWithPemBlock("tcp", app.Addr, router, certPemBlock, keyPemBlock)
		} else {
			appErr = app.serve("tcp", app.Addr, router)
		}
	}

	// 等待连接关闭
	if app.sockClosed != nil {
		<-app.sockClosed
	}

	return appErr
}

// InitDb load mysql and sqlite3 config section to app.
func (app *App) InitDb() {
	cfg := app.Config

	// mysql初始化
	if dbAliases, err := cfg.GetSubKeys("mysql"); err == nil {
		if HasDriver("mysql") {
			for _, dbAlias := range dbAliases {
				keyPrefix := "mysql." + dbAlias + "."
				if !cfg.GetDefaultBool(keyPrefix+"enabled", true) {
					continue
				}
				err := app.DB.AddMysql(
					dbAlias,
					cfg.GetDefaultString(keyPrefix+"protocol", "tcp"),
					cfg.GetDefaultString(keyPrefix+"address", "127.0.0.1:3306"),
					cfg.GetDefaultString(keyPrefix+"user", ""),
					cfg.GetDefaultString(keyPrefix+"passwd", ""),
					cfg.GetDefaultString(keyPrefix+"dbname", ""),
					cfg.GetDefaultInt(keyPrefix+"max_idle_conns", 5),
					cfg.GetDefaultInt(keyPrefix+"max_open_conns", 100),
				)
				if err != nil {
					app.Logger.Warningf("(api) add mysql db failed: %s", err.Error())
				} else {
					app.DB.SetMysqlShowSQL(dbAlias, cfg.GetDefaultBool(keyPrefix+"show_sql", false))
					app.DB.SetMysqlLogLevel(dbAlias, cfg.GetDefaultString(keyPrefix+"log_level", "error"))
				}
			}
		} else {
			app.Logger.Warning("(api) mysql driver is not ready, ignore mysql db config.")
		}
	}

	// sqlite初始化
	sqliteDbs := make(map[string]string)
	sqliteDbPersitents := make(map[string]bool)
	envDb := os.Getenv("APP_DB")
	if envDb != "" {
		os.Setenv("APP_DB", "") // 读取完立即清空，防止被子进程继承

		// 环境变量中指定的DB，格式如：APP_DB="default:test.db:true;log:log.db"
		if HasDriver("sqlite3") {
			dbStrs := strings.Split(envDb, ";")
			for _, dbStr := range dbStrs {
				fields := strings.SplitN(dbStr, ":", -1)
				if len(fields) != 2 && len(fields) != 3 {
					continue
				}
				var dbAlias, dbPath, persistent string
				if len(fields) == 2 {
					dbAlias, dbPath = fields[0], fields[1]
				} else {
					dbAlias, dbPath, persistent = fields[0], fields[1], fields[2]
				}
				sqliteDbs[dbAlias] = dbPath
				sqliteDbPersitents[dbAlias] = persistent == "true"
			}
		}
	}
	if dbAliases, err := cfg.GetSubKeys("sqlite3"); err == nil {
		if HasDriver("sqlite3") {
			for _, dbAlias := range dbAliases {
				keyPrefix := "sqlite3." + dbAlias + "."
				if !cfg.GetDefaultBool(keyPrefix+"enabled", true) {
					continue
				}
				dbPath := cfg.GetDefaultString(keyPrefix+"db", "")
				if dbPath == "" {
					continue
				}
				persistent := cfg.GetDefaultBool(keyPrefix+"persistent", false)

				_, has := sqliteDbs[dbAlias]
				if has {
					// 检查是否允许被环境变量覆盖
					// 如果不允许覆盖，则使用配置中的值
					allowEnv := cfg.GetDefaultBool(keyPrefix+"allow_env", true)
					if !allowEnv {
						sqliteDbs[dbAlias] = dbPath
						sqliteDbPersitents[dbAlias] = persistent
					}
				} else {
					sqliteDbs[dbAlias] = dbPath
					sqliteDbPersitents[dbAlias] = persistent
				}
			}
		}
	}

	if len(sqliteDbs) > 0 {
		if !HasDriver("sqlite3") {
			app.Logger.Warning("(api) sqlite3 driver is not ready, ignore sqlite3 db config.")
		} else {
			for dbAlias, dbPath := range sqliteDbs {
				err := app.DB.AddSqlite3(dbAlias, dbPath, sqliteDbPersitents[dbAlias])
				if err != nil {
					app.Logger.Warningf("(api) add sqlite3 db failed: %s", err.Error())
				} else {
					keyPrefix := "sqlite3." + dbAlias + "."
					app.DB.SetSqlite3ShowSQL(dbAlias, cfg.GetDefaultBool(keyPrefix+"show_sql", false))
					app.DB.SetSqlite3LogLevel(dbAlias, cfg.GetDefaultString(keyPrefix+"log_level", "error"))
				}
			}
		}
	}
}

// InitError register error and word map for app.
func (app *App) InitError() {
	app.Error.RegisterGroupErrors("global", globalErrorDefines)
	app.Error.RegisterErrors(appErrorDefines)
	app.LoadConfigWords()

	app.LoadFilterErrors()
}

// RegisterModel register a model for app.
func (app *App) RegisterModel(model interface{}) {
	app.Model.Register(model)
}

// RegisterModels register models for app.
func (app *App) RegisterModels(models []interface{}) {
	app.Model.Register(models)
}

// LoadFilterErrors load filter errors to app.
func (app *App) LoadFilterErrors() {
	// load to default group
	app.Error.RegisterWords(filter.ErrorWordMap)
}

// LoadConfigWords load words config section to app.
func (app *App) LoadConfigWords() {
	if words, err := app.Config.Get("words"); err == nil {
		wMap := make(map[string]map[string]string)

		v, ok := words.(map[interface{}]interface{})
		if !ok {
			goto invalid_format
		}

		for tLangCode, vv := range v {
			langCode, ok := tLangCode.(string)
			if !ok {
				goto invalid_format
			}

			lwMap := make(map[string]string)

			vvv, ok := vv.(map[interface{}]interface{})
			if !ok {
				goto invalid_format
			}

			for tWord, vvvv := range vvv {
				word, ok := tWord.(string)
				if !ok {
					goto invalid_format
				}

				phrase, ok := vvvv.(string)
				if !ok {
					goto invalid_format
				}
				lwMap[word] = phrase
			}

			wMap[langCode] = lwMap
		}

		app.Error.RegisterWords(wMap)
	}
	return

invalid_format:
	app.Logger.Warning("(api) format error with words config: ignore words config.")
}

// Route return a handler func using by http.HandleFunc.
func (app *App) Route(routes []*Route) http.HandlerFunc {
	return newApiHandler(app, routes)
}

// Hook add set global hook action at specified tag.
func (app *App) Hook(tag string, actionFunc ActionFunc) *App {
	if _, has := app.Hooks[tag]; !has {
		app.Hooks[tag] = make([]ActionFunc, 0, 1)
	}
	app.Hooks[tag] = append(app.Hooks[tag], actionFunc)
	return app
}

// SessionStore return the session store of application.
func (app *App) SessionStore() (*session.CookieStore, error) {
	storeType := app.Config.GetDefaultString("session.store_type", "memory")
	if storeType == "memory" {
		return session.DefaultCookieStore()
	}
	keyPairsFile := app.Config.GetDefaultString("session.key_pairs_file", "")
	if keyPairsFile == "" {
		return session.DefaultCookieStore()
	} else {
		return session.NewCookieStore(true, keyPairsFile)
	}
}

func (app *App) GetLogger() *logging.Logger {
	return app.Logger.Logger
}

func (app *App) Close() error {
	return app.HTTPServer.Close()
}
