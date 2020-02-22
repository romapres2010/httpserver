package http

// Обертка над стандартным пакетом http используется для изоляции HTTP кода и обработчиков

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/gorilla/mux"
	myerror "github.com/romapres2010/httpserver/error"
	handler "github.com/romapres2010/httpserver/http/handler"
	mylog "github.com/romapres2010/httpserver/log"
)

// Server repesent HTTP server
type Server struct {
	ctx    context.Context    // корневой контекст при инициации сервиса
	cancel context.CancelFunc // функция закрытия глобального контекста
	cfg    *Config            // конфигурация HTTP сервера

	listener net.Listener     // листинер HTTP сервера
	router   *mux.Router      // роутер HTTP сервера
	server   *http.Server     // HTTP сервер
	handler  *handler.Handler // обработчик HTTP запросов
}

// Config repesent HTTP server options
type Config struct {
	ListenSpec     string // строка HTTP листенера
	ReadTimeout    int    // HTTP read timeout duration in sec - default 60 sec
	WriteTimeout   int    // HTTP write timeout duration in sec - default 60 sec
	IdleTimeout    int    // HTTP idle timeout duration in sec - default 60 sec
	MaxHeaderBytes int    // HTTP max header bytes - default 1 MB
	MaxBodyBytes   int    // HTTP max body bytes - default 0 - unlimited
	UseTLS         bool   // use Transport Level Security
	UseHSTS        bool   // use HTTP Strict Transport Security
	TLSSertFile    string // TLS Sertificate file name
	TLSKeyFile     string // TLS Private key file name
	TLSMinVersion  uint16 // TLS min version VersionTLS13, VersionTLS12, VersionTLS11, VersionTLS10, VersionSSL30
	TLSMaxVersion  uint16 // TLS max version VersionTLS13, VersionTLS12, VersionTLS11, VersionTLS10, VersionSSL30

	// конфигурация вложенных сервисов
	HandlerCfg handler.Config // конфигурация HTTP обработчика
}

// NewServer - create new HTTP server
func NewServer(ctx context.Context, cfg *Config) (*Server, error) {
	var err error
	var server *Server

	mylog.PrintfInfoStd("Starting to create new HTTP server")

	{ // Конфигурация HTTP сервера
		server = &Server{
			cfg: cfg,
			server: &http.Server{
				ReadTimeout:  time.Duration(cfg.ReadTimeout * int(time.Second)),
				WriteTimeout: time.Duration(cfg.WriteTimeout * int(time.Second)),
				IdleTimeout:  time.Duration(cfg.IdleTimeout * int(time.Second)),
			},
		}

		// создаем контекст с отменой
		if ctx == nil {
			server.ctx, server.cancel = context.WithCancel(context.Background())
		} else {
			server.ctx, server.cancel = context.WithCancel(ctx)
		}

		// Если задано ограничение на header
		if cfg.MaxHeaderBytes > 0 {
			server.server.MaxHeaderBytes = cfg.MaxHeaderBytes
		}

		// Новый HTTP обработчик
		if server.handler, err = handler.NewHandler(server.ctx, &cfg.HandlerCfg); err != nil {
			return nil, err
		}

	} // Конфигурация HTTP сервера

	{ // Определяем  листенер
		server.listener, err = net.Listen("tcp", cfg.ListenSpec)
		if err != nil {
			errM := fmt.Sprintf("Failed create new TCP listener network='tcp', address='%s'", cfg.ListenSpec)
			mylog.PrintfErrorStd(errM)
			return nil, myerror.WithCause("5006", errM, "net.Listen()", fmt.Sprintf("network='tcp', address='%s'", cfg.ListenSpec), "", err.Error())
		}

		mylog.PrintfInfoStd(fmt.Sprintf("Create new TCP listener network='tcp', address='%s'", cfg.ListenSpec))
	} // Определяем  листенер

	{ // Настраиваем роутер
		server.router = mux.NewRouter()

		// страница эхо с входными параметрами и body
		server.router.HandleFunc("/echo", server.handler.RecoverWrap(server.handler.EchoHandler)).Methods("GET")
		mylog.PrintfInfoStd("'/echo' GET handler is registered")

		// страница авторизации и renew JWT
		server.router.HandleFunc("/signin", server.handler.RecoverWrap(server.handler.SinginHandler)).Methods("POST")
		server.router.HandleFunc("/refresh", server.handler.RecoverWrap(server.handler.JWTRefreshHandler)).Methods("POST")

		// Регистрация pprof-обработчиков
		{
			pprofrouter := server.router.PathPrefix("/debug/pprof").Subrouter()
			pprofrouter.HandleFunc("/", pprof.Index)
			pprofrouter.HandleFunc("/cmdline", pprof.Cmdline)
			pprofrouter.HandleFunc("/symbol", pprof.Symbol)
			pprofrouter.HandleFunc("/trace", pprof.Trace)

			profile := pprofrouter.PathPrefix("/profile").Subrouter()
			profile.HandleFunc("", pprof.Profile)
			profile.Handle("/goroutine", pprof.Handler("goroutine"))
			profile.Handle("/threadcreate", pprof.Handler("threadcreate"))
			profile.Handle("/heap", pprof.Handler("heap"))
			profile.Handle("/block", pprof.Handler("block"))
			profile.Handle("/mutex", pprof.Handler("mutex"))
		}
	} // Настраиваем роутер

	// Устанавливаем роутер в качестве корневого обработчика
	http.Handle("/", server.router)

	// настраиваем параметры TLS
	if server.cfg.UseTLS {
		mylog.PrintfInfoStd(fmt.Sprintf("Starting HTTPS server, TLSSertFile='%s', TLSKeyFile='%s'", server.cfg.TLSSertFile, server.cfg.TLSKeyFile))
		// определим минимальную и максимальную версию TLS
		tlsCfg := &tls.Config{
			MinVersion: server.cfg.TLSMinVersion,
			MaxVersion: server.cfg.TLSMaxVersion,
			/*
				//отлючение методов шифрования с ключами менее 256 бит
					CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
					PreferServerCipherSuites: true,
					CipherSuites: []uint16{
						tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
						tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_RSA_WITH_AES_256_CBC_SHA,
					},
			*/
		}
		server.server.TLSConfig = tlsCfg
		/*
			//Отключение HTTP/2, чтобы исключить поддержку ключа с 128 битами TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
			server.server.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0)
		*/
	}

	mylog.PrintfInfoStd("New HTTP server is created")
	return server, nil
}

// Run HTTP server - wait for error or exit
// =====================================================================
func (s *Server) Run() error {
	if s.cfg.UseTLS {
		mylog.PrintfInfoStd(fmt.Sprintf("Starting HTTPS server, TLSSertFile='%s', TLSKeyFile='%s', TLSConfig='%+v'", s.cfg.TLSSertFile, s.cfg.TLSKeyFile, s.server.TLSConfig))
		return s.server.ServeTLS(s.listener, s.cfg.TLSSertFile, s.cfg.TLSKeyFile)
	}
	mylog.PrintfInfoStd(fmt.Sprintf("Starting HTTP server"))
	return s.server.Serve(s.listener)
}

// Shutdown HTTP server
// =====================================================================
func (s *Server) Shutdown() error {
	mylog.PrintfInfoStd("START")

	// закрываем корневой контекст с ожидаением на закрытие простаивающих подключений
	mylog.PrintfInfoStd("Waiting for shutdown of HTTP Server 30 sec")
	cancelCtx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	err := s.server.Shutdown(cancelCtx)
	if err != nil {
		errM := "Error shutdown HTTP server"
		mylog.PrintfErrorStd(errM)
		return myerror.WithCause("8003", errM, "server.Shutdown", "", "", err.Error())
	}

	// Закрываем HTTPLogger для корректного закрытия лог файла
	s.handler.Shutdown()

	return nil
}
