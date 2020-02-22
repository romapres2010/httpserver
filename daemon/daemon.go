package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sasbury/mini"

	myerror "github.com/romapres2010/httpserver/error"
	myhttp "github.com/romapres2010/httpserver/http"
	mylog "github.com/romapres2010/httpserver/log"
)

// Daemon repesent top level daemon
type Daemon struct {
	ctx            context.Context    // корневой контекст при инициации сервиса
	cancel         context.CancelFunc // функция закрытия глобального контекста
	configFileName string             // основной файл конфигурации

	// Сервисы демона
	server *myhttp.Server // HTTP server
	// ... здесь добавляем новые сервисы

	// Конфигурация всех сервисов
	httpServerCfg myhttp.Config // конфигурация HTTP сервера

	// ... здесь добавляем новые конфигурации
}

// New create new Daemon
func New(configFileName string, listenSpec string, httpUserID string, httpUserPwd string, jwtKey []byte) (*Daemon, error) {
	var err error
	var config *mini.Config

	mylog.PrintfInfoStd("Starting to create new daemon", configFileName, listenSpec)

	// Создаем новый демон
	daemon := &Daemon{
		configFileName: configFileName,
	}

	// создаем новый контекст с отменой, используется при остановке сервера, всех сервисов и текущих запросов
	daemon.ctx, daemon.cancel = context.WithCancel(context.Background())

	// Загружаем конфигурационный файл
	if config, err = loadConfigFile(daemon.configFileName); err != nil {
		return nil, err
	}

	{ // HTTP server
		// Настраиваем конфигурацию HTTP server
		if err = loadHTTPServerConfig(config, &daemon.httpServerCfg); err == nil {
			daemon.httpServerCfg.ListenSpec = listenSpec // адрес листенера передается через строку запуска
		} else {
			return nil, err
		}

		// Настраиваем конфигурацию HTTP handler
		if err = loadHTTPHandlerConfig(config, &daemon.httpServerCfg.HandlerCfg); err == nil {
			// Параметры из командной строки
			daemon.httpServerCfg.HandlerCfg.HTTPUserID = httpUserID
			daemon.httpServerCfg.HandlerCfg.HTTPUserPwd = httpUserPwd
			daemon.httpServerCfg.HandlerCfg.JwtKey = jwtKey
			// Параметры уровня HTTP сервера
			daemon.httpServerCfg.HandlerCfg.UseTLS = daemon.httpServerCfg.UseTLS
			daemon.httpServerCfg.HandlerCfg.UseHSTS = daemon.httpServerCfg.UseHSTS
			daemon.httpServerCfg.HandlerCfg.MaxBodyBytes = daemon.httpServerCfg.MaxBodyBytes

			// задан ли в строке запуска JSON web token secret key
			if daemon.httpServerCfg.HandlerCfg.UseJWT && jwtKey == nil {
				errM := fmt.Sprintf("JSON web token secret key is null")
				mylog.PrintfErrorStd(errM)
				return nil, myerror.New("6023", errM, "", "")
			}

			// Настраиваем конфигурацию HTTP Logger
			if err = loadHTTPLoggerConfig(config, &daemon.httpServerCfg.HandlerCfg.LogCfg); err != nil {
				return nil, err
			}

		} else {
			return nil, err
		}

		// Создаем HTTP server
		if daemon.server, err = myhttp.NewServer(daemon.ctx, &daemon.httpServerCfg); err != nil {
			return nil, err
		}
	} // HTTP server

	{ // Настройка остальных сервисов
	} // Настройка остальных сервисов

	mylog.PrintfInfoStd("New daemon is created", configFileName, listenSpec)
	return daemon, nil
}

// Run - cтартует демон и ожидает сигнала выхода
func (d *Daemon) Run() error {
	mylog.PrintfInfoStd("Starting")

	errCh := make(chan error, 1)        // канал ошибок
	syscalCh := make(chan os.Signal, 1) // канал системных прирываний

	// запускаем в фоне HTTP сервер, возврат в канал ошибок
	go func() { errCh <- d.server.Run() }()

	mylog.PrintfInfoStd("Daemon is running. For exit <CTRL-c>")

	// подписываемся на системные прирывания
	signal.Notify(syscalCh, syscall.SIGINT, syscall.SIGTERM)

	// ожидаем прерывания или возврат в канал ошибок
	for {
		select {
		case s := <-syscalCh: //  ожидаем системное прирывание
			mylog.PrintfInfoStd(fmt.Sprintf("Got signal: %v, exiting", s))

			// Останавливаем HTTP сервер, ожидаем завершения активных подключений
			err := d.server.Shutdown()

			// Закрываем корневой контекст
			mylog.PrintfInfoStd("Closing daemon context")
			d.cancel()
			return err
		case err := <-errCh: // получили возврат от HTTP сервера - ошибка запуска
			// Закрываем корневой контекст
			mylog.PrintfInfoStd("Closing daemon context")
			d.cancel()
			return err
		}
	}
}
