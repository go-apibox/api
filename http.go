package api

import (
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
// REF: http://godoc.quyun.net/src/net/http/server.go?s=51101:51115#L1971
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

func (app *App) listen(netType, laddr string) (net.Listener, error) {
	ln, err := net.Listen(netType, laddr)
	for _, listenEventHandler := range app.listenEventHandlers {
		listenEventHandler(err)
	}
	return ln, err
}

func (app *App) signalHandler(ln net.Listener) {
	// REF: http://stackoverflow.com/questions/16681944/how-to-reliably-unlink-a-unix-domain-socket-in-go-programming-language
	// Unix sockets must be unlinked before being reused again.
	// Handle common process-killing signals so we can gracefully shut down:
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func(c chan os.Signal, ln net.Listener) {
		app.sockClosed = make(chan bool)

		// Wait for a SIGINT or SIGKILL:
		sig := <-c
		app.Logger.Notice("Caught signal %s: shutting down.", sig)
		// Stop listening (and unlink the socket if unix type):
		ln.Close()

		signal.Stop(c)

		app.sockClosed <- true
	}(sigc, ln)
}

// http server on specified handler
func (app *App) serve(network, addr string, handler http.Handler) error {
	app.HTTPServer = &http.Server{Addr: addr, Handler: handler}
	ln, err := app.listen(network, app.HTTPServer.Addr)
	if err != nil {
		return err
	}

	// 与unix domain socket统一处理
	// 当项目多次调用API库时（多个APP），有些APP使用unix domain socket，有些使用 tcp，
	// 则可能导致使用 tcp 的 APP 不退出（因为signal被其它APP处理）
	app.signalHandler(ln)

	if network == "tcp" {
		ln = tcpKeepAliveListener{ln.(*net.TCPListener)}
	}

	return app.HTTPServer.Serve(ln)
}

// http server on tls
func (app *App) serveTLS(network, addr string, handler http.Handler, certFile, keyFile string) error {
	// tcp socket
	config := &tls.Config{
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			// 必须放前面，否则会运行报错：
			// http2: TLSConfig.CipherSuites index 8 contains an HTTP/2-approved cipher suite (0xc02f), but it comes
			// after unapproved cipher suites. With this configuration, clients that don't support previous, approved
			// cipher suites may be given an unapproved one and reject the connection.
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			// tls.TLS_RSA_WITH_RC4_128_SHA, // ssllabs test: INSECURE
			tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			// tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA, // ssllabs test: INSECURE
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			// tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA, // ssllabs test: INSECURE
			tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		},
		MinVersion: tls.VersionTLS10,
		MaxVersion: tls.VersionTLS12,
		NextProtos: []string{},
	}

	app.HTTPServer = &http.Server{Addr: addr, Handler: handler, TLSConfig: config}
	if addr == "" {
		addr = ":https"
	}

	var err error
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}

	ln, err := app.listen("tcp", app.HTTPServer.Addr)
	if err != nil {
		return err
	}

	if network == "tcp" {
		tcpLn := tcpKeepAliveListener{ln.(*net.TCPListener)}
		ln = tls.NewListener(tcpLn, config)
	}

	// 与unix domain socket统一处理
	// 当项目多次调用API库时（多个APP），有些APP使用unix domain socket，有些使用 tcp，
	// 则可能导致使用 tcp 的 APP 不退出（因为signal被其它APP处理）
	app.signalHandler(ln)

	return app.HTTPServer.Serve(ln)
}

// http server with cert pem and key pem
func (app *App) serveTLSWithPemBlock(network, addr string, handler http.Handler, certPemBlock, keyPemBlock []byte) error {
	config := &tls.Config{
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			// 必须放前面，否则会运行报错：
			// http2: TLSConfig.CipherSuites index 8 contains an HTTP/2-approved cipher suite (0xc02f), but it comes
			// after unapproved cipher suites. With this configuration, clients that don't support previous, approved
			// cipher suites may be given an unapproved one and reject the connection.
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			// tls.TLS_RSA_WITH_RC4_128_SHA, // ssllabs test: INSECURE
			tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			// tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA, // ssllabs test: INSECURE
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			// tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA, // ssllabs test: INSECURE
			tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		},
		MinVersion: tls.VersionTLS10,
		MaxVersion: tls.VersionTLS12,
		NextProtos: []string{},
	}

	app.HTTPServer = &http.Server{Addr: addr, Handler: handler, TLSConfig: config}
	if addr == "" {
		addr = ":https"
	}

	var err error
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.X509KeyPair(certPemBlock, keyPemBlock)
	if err != nil {
		return err
	}

	ln, err := app.listen(network, addr)
	if err != nil {
		return err
	}

	if network == "tcp" {
		tcpLn := tcpKeepAliveListener{ln.(*net.TCPListener)}
		ln = tls.NewListener(tcpLn, config)
	} else {
		ln = tls.NewListener(ln, config)
	}

	// 与unix domain socket统一处理
	// 当项目多次调用API库时（多个APP），有些APP使用unix domain socket，有些使用 tcp，
	// 则可能导致使用 tcp 的 APP 不退出（因为signal被其它APP处理）
	app.signalHandler(ln)

	return app.HTTPServer.Serve(ln)
}

// 监听事件处理函数
type ListenEventHandler func(err error)

// AddListenEventHandler add handler when listen return.
func (app *App) AddListenEventHandler(h ListenEventHandler) {
	app.listenEventHandlers = append(app.listenEventHandlers, h)
}

func strSliceContains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
