package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sasbury/mini"

	myerror "github.com/romapres2010/httpserver/error"
	"github.com/romapres2010/httpserver/httpserver"
	mylog "github.com/romapres2010/httpserver/log"
)

// Daemon repesent top level daemon
type Daemon struct {
	ctx    context.Context    // корневой контекст
	cancel context.CancelFunc // функция закрытия корневого контекста
	cfg    *Config            // конфигурация демона

	// Сервисы демона
	httpserver *httpserver.Server // HTTP httpserver
	// ... здесь добавляем новые сервисы
}

// Config repesent daemon options
type Config struct {
	ConfigFileName string // основной файл конфигурации
	ListenSpec     string // строка HTTP листенера
	JwtKey         []byte // JWT secret key
	HTTPUserID     string // пользователь для HTTP Basic Authentication
	HTTPUserPwd    string // пароль для HTTP Basic Authentication

	// Конфигурация вложенных сервисов
	httpServerCfg httpserver.Config // конфигурация HTTP сервера
	// ... здесь добавляем новые конфигурации
}

// New create new Daemon
func New(ctx context.Context, cfg *Config) (*Daemon, error) {
	var err error
	var config *mini.Config

	mylog.PrintfInfoStd("Starting to create new daemon")

	{ // входные проверки
		if cfg == nil {
			errM := fmt.Sprintf("Empty config")
			mylog.PrintfErrorStd(errM)
			return nil, myerror.New("6030", errM, "", "")
		}
		if cfg.ConfigFileName == "" {
			errM := fmt.Sprintf("Empty config file name")
			mylog.PrintfErrorStd(errM)
			return nil, myerror.New("6030", errM, "", "")
		}
		// ... дополнительные проверки
	} // входные проверки

	// Создаем новый демон
	daemon := &Daemon{
		cfg: cfg,
	}

	// создаем контекст с отменой
	if ctx == nil {
		daemon.ctx, daemon.cancel = context.WithCancel(context.Background())
	} else {
		daemon.ctx, daemon.cancel = context.WithCancel(ctx)
	}

	// Загружаем конфигурационный файл
	if config, err = loadConfigFile(daemon.cfg.ConfigFileName); err != nil {
		return nil, err
	}

	{ // HTTP server
		// Настраиваем конфигурацию HTTP server
		if err = loadHTTPServerConfig(config, &daemon.cfg.httpServerCfg); err == nil {
			daemon.cfg.httpServerCfg.ListenSpec = daemon.cfg.ListenSpec
		} else {
			return nil, err
		}

		// Настраиваем конфигурацию HTTP handler
		if err = loadHTTPHandlerConfig(config, &daemon.cfg.httpServerCfg.ServiceCfg); err == nil {
			// Параметры из командной строки
			daemon.cfg.httpServerCfg.ServiceCfg.HTTPUserID = daemon.cfg.HTTPUserID
			daemon.cfg.httpServerCfg.ServiceCfg.HTTPUserPwd = daemon.cfg.HTTPUserPwd
			daemon.cfg.httpServerCfg.ServiceCfg.JwtKey = daemon.cfg.JwtKey
			// Параметры уровня HTTP сервера
			daemon.cfg.httpServerCfg.ServiceCfg.UseTLS = daemon.cfg.httpServerCfg.UseTLS
			daemon.cfg.httpServerCfg.ServiceCfg.UseHSTS = daemon.cfg.httpServerCfg.UseHSTS
			daemon.cfg.httpServerCfg.ServiceCfg.MaxBodyBytes = daemon.cfg.httpServerCfg.MaxBodyBytes

			// задан ли в командной строке JSON web token secret key
			if daemon.cfg.httpServerCfg.ServiceCfg.UseJWT && daemon.cfg.JwtKey == nil {
				errM := fmt.Sprintf("JSON web token secret key is null")
				mylog.PrintfErrorStd(errM)
				return nil, myerror.New("6023", errM, "", "")
			}

			// Настраиваем конфигурацию HTTP Logger
			if err = loadHTTPLoggerConfig(config, &daemon.cfg.httpServerCfg.ServiceCfg.LogCfg); err != nil {
				return nil, err
			}

		} else {
			return nil, err
		}

		// Создаем HTTP server
		if daemon.httpserver, err = httpserver.New(daemon.ctx, &daemon.cfg.httpServerCfg); err != nil {
			return nil, err
		}
	} // HTTP server

	{ // Настройка остальных сервисов
	} // Настройка остальных сервисов

	mylog.PrintfInfoStd("New daemon is created")
	return daemon, nil
}

// Run - cтартует демон и ожидает сигнала выхода
func (d *Daemon) Run() error {
	mylog.PrintfInfoStd("Starting")

	errCh := make(chan error, 1)        // канал ошибок
	syscalCh := make(chan os.Signal, 1) // канал системных прирываний

	// запускаем в фоне HTTP сервер, возврат в канал ошибок
	go func() { errCh <- d.httpserver.Run() }()

	// ... запуск остальных сервисов

	mylog.PrintfInfoStd("Daemon is running. For exit <CTRL-c>")

	// подписываемся на системные прирывания
	signal.Notify(syscalCh, syscall.SIGINT, syscall.SIGTERM)

	// ожидаем прерывания или возврат в канал ошибок
	for {
		select {
		case s := <-syscalCh: //  ожидаем системное прирывание
			mylog.PrintfInfoStd(fmt.Sprintf("Got signal: %v, exiting", s))

			// Останавливаем HTTP сервер, ожидаем завершения активных подключений
			err := d.httpserver.Shutdown()

			// ... остановка остальных сервисов

			// Закрываем корневой контекст
			mylog.PrintfInfoStd("Closing daemon context")
			d.cancel()
			return err
		case err := <-errCh: // возврат от сервисов в канал ошибок

			// ... анализ ошибки и остановка остальных сервисов

			// Закрываем корневой контекст
			mylog.PrintfInfoStd("Closing daemon context")
			d.cancel()
			return err
		}
	}
}
