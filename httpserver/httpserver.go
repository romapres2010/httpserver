package httpserver

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
	"github.com/romapres2010/httpserver/httpserver/httplog"
	"github.com/romapres2010/httpserver/httpserver/httpservice"
	mylog "github.com/romapres2010/httpserver/log"
)

// Server repesent HTTP server
type Server struct {
	ctx    context.Context    // контекст при инициации сервиса
	cancel context.CancelFunc // функция закрытия контекста
	cfg    *Config            // конфигурация HTTP сервера
	errCh  chan error         // канал ошибок
	stopCh chan struct{}      // канал подтверждения об успешном закрытии HTTP сервера

	// вложенные сервисы
	listener    net.Listener         // листинер HTTP сервера
	router      *mux.Router          // роутер HTTP сервера
	httpServer  *http.Server         // собственно HTTP сервер
	httpService *httpservice.Service // сервис HTTP запросов
	logger      *httplog.Logger      // сервис логирования HTTP трафика
}

// Config repesent HTTP server options
type Config struct {
	ListenSpec      string // HTTP listener address string
	ReadTimeout     int    // HTTP read timeout duration in sec - default 60 sec
	WriteTimeout    int    // HTTP write timeout duration in sec - default 60 sec
	IdleTimeout     int    // HTTP idle timeout duration in sec - default 60 sec
	MaxHeaderBytes  int    // HTTP max header bytes - default 1 MB
	MaxBodyBytes    int    // HTTP max body bytes - default 0 - unlimited
	UseProfile      bool   // use Go profiling
	UseTLS          bool   // use Transport Level Security
	UseHSTS         bool   // use HTTP Strict Transport Security
	TLSCertFile     string // TLS Certificate file name
	TLSKeyFile      string // TLS Private key file name
	TLSMinVersion   uint16 // TLS min version VersionTLS13, VersionTLS12, VersionTLS11, VersionTLS10, VersionSSL30
	TLSMaxVersion   uint16 // TLS max version VersionTLS13, VersionTLS12, VersionTLS11, VersionTLS10, VersionSSL30
	ShutdownTimeout int    // service shutdown timeout in sec - default 30 sec

	// конфигурация вложенных сервисов
	ServiceCfg httpservice.Config // конфигурация HTTP сервиса
}

// New create HTTP server
func New(ctx context.Context, errCh chan error, cfg *Config) (*Server, error) {
	var err error
	var server *Server

	mylog.PrintfInfoMsg("Creating new HTTP server")

	{ // входные проверки
		if cfg == nil {
			myerr := myerror.New("6030", "Empty HTTP server config")
			mylog.PrintfErrorInfo(myerr)
			return nil, myerr
		}
		if cfg.ListenSpec == "" {
			myerr := myerror.New("6030", "Empty HTTP listener address")
			mylog.PrintfErrorInfo(myerr)
			return nil, myerr
		}
	} // входные проверки

	// Создаем новый сервер
	server = &Server{
		cfg:    cfg,
		errCh:  errCh,
		stopCh: make(chan struct{}, 1), // канал подтверждения об успешном закрытии HTTP сервера
	}

	// создаем контекст с отменой
	if ctx == nil {
		server.ctx, server.cancel = context.WithCancel(context.Background())
	} else {
		server.ctx, server.cancel = context.WithCancel(ctx)
	}

	// Новый HTTP сервис и HTTP logger
	if server.httpService, server.logger, err = httpservice.New(server.ctx, &cfg.ServiceCfg); err != nil {
		return nil, err
	}

	{ // Конфигурация HTTP сервера
		server.httpServer = &http.Server{
			ReadTimeout:  time.Duration(cfg.ReadTimeout * int(time.Second)),
			WriteTimeout: time.Duration(cfg.WriteTimeout * int(time.Second)),
			IdleTimeout:  time.Duration(cfg.IdleTimeout * int(time.Second)),
		}

		// Если задано ограничение на header
		if cfg.MaxHeaderBytes > 0 {
			server.httpServer.MaxHeaderBytes = cfg.MaxHeaderBytes
		}

		// настраиваем параметры TLS
		if server.cfg.UseTLS {
			// определим минимальную и максимальную версию TLS
			tlsCfg := &tls.Config{
				MinVersion: server.cfg.TLSMinVersion,
				MaxVersion: server.cfg.TLSMaxVersion,
				/*
					//отлючение методов шифрования с ключами менее 256 бит
						CurvePreferences:  []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
						PreferServerCipherSuites: true,
						CipherSuites: []uint16{
							tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
							tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
							tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
							tls.TLS_RSA_WITH_AES_256_CBC_SHA,
						},
				*/
			}
			server.httpServer.TLSConfig = tlsCfg
			/*
				//Отключение HTTP/2, чтобы исключить поддержку ключа с 128 битами TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
				server.httpServer.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0)
			*/
		}
	} // Конфигурация HTTP сервера

	{ // Определяем  листенер
		server.listener, err = net.Listen("tcp", cfg.ListenSpec)
		if err != nil {
			myerr := myerror.WithCause("5006", "Failed to create new TCP listener: network = 'tcp', address", err, cfg.ListenSpec)
			mylog.PrintfErrorInfo(myerr)
			return nil, myerr
		}

		mylog.PrintfInfoMsg("Created new TCP listener: network = 'tcp', address", cfg.ListenSpec)
	} // Определяем  листенер

	{ // Настраиваем роутер
		// создаем новый роутер
		server.router = mux.NewRouter()

		// Устанавливаем роутер в качестве корневого обработчика
		http.Handle("/", server.router)

		// Зарегистрируем HTTP обработчиков
		if server.httpService.Handlers != nil {
			for _, h := range server.httpService.Handlers {
				server.router.HandleFunc(h.Path, h.HundlerFunc).Methods(h.Method)
				mylog.PrintfInfoMsg("Handler is registered: Path, Method", h.Path, h.Method)
			}
		}

		// Регистрация pprof-обработчиков
		if server.cfg.UseProfile {
			mylog.PrintfInfoMsg("'/debug/pprof' is registered")

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

	mylog.PrintfInfoMsg("HTTP server is created")
	return server, nil
}

// Run HTTP server - wait for error or exit
func (s *Server) Run() error {
	// Функция восстановления после паники
	defer func() {
		var myerr error
		r := recover()
		if r != nil {
			msg := "HTTP server recover from panic"
			switch t := r.(type) {
			case string:
				myerr = myerror.New("8888", msg, t)
			case error:
				myerr = myerror.WithCause("8888", msg, t)
			default:
				myerr = myerror.New("8888", msg)
			}
			mylog.PrintfErrorInfo(myerr) // логируем ошибку
			s.errCh <- myerr             // передаем ошибку в канал для уведомления daemon
		}
	}()

	// Запускаем HTTP сервер
	if s.cfg.UseTLS {
		mylog.PrintfInfoMsg("Starting HTTPS server: TLSSertFile, TLSKeyFile", s.cfg.TLSCertFile, s.cfg.TLSKeyFile)
		return s.httpServer.ServeTLS(s.listener, s.cfg.TLSCertFile, s.cfg.TLSKeyFile)
	}
	mylog.PrintfInfoMsg(fmt.Sprintf("Starting HTTP server"))
	return s.httpServer.Serve(s.listener)
}

// Shutdown HTTP server
func (s *Server) Shutdown() error {
	mylog.PrintfInfoMsg("Waiting for shutdown HTTP Server: sec", s.cfg.ShutdownTimeout)

	// создаем новый контекст с отменой и отсрочкой ShutdownTimeout
	cancelCtx, cancel := context.WithTimeout(s.ctx, time.Duration(s.cfg.ShutdownTimeout*int(time.Second)))
	defer cancel()

	// ожидаем закрытия активных подключений в течении ShutdownTimeout
	if err := s.httpServer.Shutdown(cancelCtx); err != nil {
		myerr := myerror.WithCause("8003", "Failed to shutdown HTTP server: sec", err, s.cfg.ShutdownTimeout)
		mylog.PrintfErrorInfo(myerr)
		return myerr
	}

	// Останавливаем служебные сервисы
	s.httpService.Shutdown()

	mylog.PrintfInfoMsg("HTTP Server shutdown successfuly")

	// подтверждение об успешном закрытии HTTP сервера
	s.stopCh <- struct{}{}

	return nil
}
