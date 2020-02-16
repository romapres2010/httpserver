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
	handler "github.com/romapres2010/httpserver/http/handler"
	httplog "github.com/romapres2010/httpserver/http/httplog"
	mylog "github.com/romapres2010/httpserver/log"
)

// Daemon repesent top level daemon
type Daemon struct {
	HTTPConfigFile string             // основной файл конфигурации
	Server         *myhttp.Server     // HTTP server
	Ctx            context.Context    // корневой контекст при инициации сервиса
	Cancel         context.CancelFunc // функция закрытия глобального контекста
}

// New create new Daemon
// =====================================================================
func New(HTTPConfigFile string,
	listenSpec string,
	HTTPUserID string,
	HTTPUserPwd string,
	JwtKey []byte) (*Daemon, error) {

	mylog.PrintfInfoStd("START")

	var err error
	var config *mini.Config

	// Настраиваем демон
	daemon := &Daemon{
		HTTPConfigFile: HTTPConfigFile,
	}

	// создаем новый контекст с отменой
	// daemon.cancel используется при остановке сервера для остановки всех обработчиков и текущих запросов
	daemon.Ctx, daemon.Cancel = context.WithCancel(context.Background())

	// Загружаем конфигурационный файл
	if config, err = _loadConfigFile(daemon.HTTPConfigFile); err != nil {
		return nil, err
	}

	{ // HTTP server
		// конфигурационные параметры HTTPLogger
		HTTPLoggerCfg := &httplog.Config{}
		if err = _loadHTTPLoggerConfig(config, HTTPLoggerCfg); err != nil {
			return nil, err
		}

		// конфигурационные параметры HTTP сервера
		HTTPServerCfg := &myhttp.Config{
			ListenSpec: listenSpec, // адрес листинера передается не через конфиг файл, а через строку запуска
		}
		// считываем конфигурацию HTTP servera
		if err = _loadHTTPServerConfig(config, HTTPServerCfg); err != nil {
			return nil, err
		}

		// конфигурационные параметры HTTP обработчика
		HTTPHandlerCfg := &handler.Config{
			// Параметры из командной строки
			HTTPUserID:  HTTPUserID,
			HTTPUserPwd: HTTPUserPwd,
			JwtKey:      JwtKey,
			// Параметры уровня HTTP сервера
			UseTLS:       HTTPServerCfg.UseTLS,
			UseHSTS:      HTTPServerCfg.UseHSTS,
			MaxBodyBytes: HTTPServerCfg.MaxBodyBytes,
		}

		// считываем конфигурацию HTTP handler
		if err := _loadHTTPHandlerConfig(config, HTTPHandlerCfg); err != nil {
			return nil, err
		}

		// Секретный ключ для формирования JSON web token (JWT)
		// проверим задал ли в строке запуска JSON web token secret key
		if HTTPHandlerCfg.UseJWT && JwtKey == nil {
			errM := fmt.Sprintf("JSON web token secret key is null")
			mylog.PrintfErrorStd(errM)
			return nil, myerror.New("6023", errM, "", "")
		}

		// Создаем HTTP server
		daemon.Server, err = myhttp.NewServer(daemon.Ctx, HTTPHandlerCfg, HTTPServerCfg, HTTPLoggerCfg)
	} // HTTP server

	return daemon, nil
}

// Run - cтартует демон и ожидает сигнала выхода
// =====================================================================
func (d *Daemon) Run() error {
	mylog.PrintfInfoStd("Starting")

	//var err error
	errCh := make(chan error, 1)        // канал ошибок
	syscalCh := make(chan os.Signal, 1) // канал системных прирываний

	// запускаем в фоне HTTP сервер, возврат в канал ошибок
	go func() { errCh <- d.Server.Run() }()

	mylog.PrintfInfoStd("Daemon is running. For exit <CTRL-c>")

	// подписываемся на системные прирывания
	signal.Notify(syscalCh, syscall.SIGINT, syscall.SIGTERM)

	// ожидаем прерывания или возврат в канал ошибок
	for {
		select {
		case s := <-syscalCh: //  ожидаем системное прирывание
			mylog.PrintfInfoStd(fmt.Sprintf("Got signal: %v, exiting", s))

			// Останавливаем HTTP сервер, ожидаем завершения активных подключений
			err := d.Server.Shutdown()

			// Закрываем корневой контекст
			mylog.PrintfInfoStd("Closing daemon context")
			d.Cancel()
			return err
		case err := <-errCh: // получили возврат от HTTP сервера - ошибка запуска
			// Закрываем корневой контекст
			mylog.PrintfInfoStd("Closing daemon context")
			d.Cancel()
			return err
		}
	}
}
