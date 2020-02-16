package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli"

	"github.com/romapres2010/httpserver/daemon"
	myerror "github.com/romapres2010/httpserver/error"
	mylog "github.com/romapres2010/httpserver/log"
)

// параметры подменяемые компилятором при сборке бинарника
var (
	version   = "0.0.5"
	commit    = "unset"
	buildTime = "unset"
)

// Глобальные переменные в которые парсятся входные флаги
var httpConfigFileFlag string
var logFileFlag string
var listenStringFlag string
var debugFlag string
var HTTPUserIDFlag string
var HTTPUserPwdFlag string
var JwtKeyFlag string // JWT secret key

// входные флаги программы
var flags = []cli.Flag{
	cli.StringFlag{
		Name:        "httpconfig, httpcfg",
		Usage:       "HTTP Config file name",
		Required:    true,
		Destination: &httpConfigFileFlag,
	},
	cli.StringFlag{
		Name:        "listenstring, l",
		Usage:       "Listen string in format <host>:<port>",
		Required:    true,
		Destination: &listenStringFlag,
		Value:       "localhost:3000",
	},
	cli.StringFlag{
		Name:        "httpuser, httpu",
		Usage:       "User name for access to HTTP server",
		Required:    false,
		Destination: &HTTPUserIDFlag,
	},
	cli.StringFlag{
		Name:        "httppassword, httppwd",
		Usage:       "User password for access to HTTP server",
		Required:    false,
		Destination: &HTTPUserPwdFlag,
	},
	cli.StringFlag{
		Name:        "jwtkey, jwtk",
		Usage:       "JSON web token secret key",
		Required:    false,
		Destination: &JwtKeyFlag,
	},
	cli.StringFlag{
		Name:        "debug, d",
		Usage:       "Debug mode: DEBUG, WARN, INFO, ERROR",
		Required:    false,
		Destination: &debugFlag,
		Value:       "INFO",
	},
	cli.StringFlag{
		Name:        "logfile, log",
		Usage:       "Log file name",
		Required:    false,
		Destination: &logFileFlag,
		Value:       "ibmmq_%s.log", // %s время старта программы
	},
}

// checkFlags - check flags and arguments
func checkFlags() error {
	mylog.PrintfInfoStd("Starting")

	{ // входные проверки
		if httpConfigFileFlag == "" {
			errM := fmt.Sprintf("HTTP Config file name is null")
			mylog.PrintfErrorStd(errM)
			return myerror.New("6013", errM, "", "")
		}
		if listenStringFlag == "" {
			errM := fmt.Sprintf("HTTP listener string is null")
			mylog.PrintfErrorStd(errM)
			return myerror.New("6014", errM, "", "")
		}
	} // входные проверки

	// Установим фильтр логирования
	if debugFlag != "" {
		switch debugFlag {
		case "DEBUG", "WARN", "ERROR", "INFO":
			mylog.NewFilter(debugFlag)
		default:
			return myerror.New("9001", fmt.Sprintf("Incorrect debugFlag '%s'. Only avaliable: DEBUG, WARN, INFO, ERROR.", debugFlag), "", "")
		}
	}

	mylog.PrintfInfoStd("SUCCESS")
	return nil
}

//main - represent main function
// =====================================================================
func main() {
	app := cli.NewApp() // создаем приложение
	app.Name = "HTTP Server"
	app.Version = fmt.Sprintf("%s, commit '%s', build time '%s'", version, commit, buildTime)
	app.Author = "Presnyakov Roman"
	app.Email = "romapres@mail.ru"
	app.Usage = "represent HTTP Server"
	app.Flags = flags // присваиваем ранее определенные флаги
	app.Writer = os.Stderr
	app.Compiled = time.Now()

	// Определяем единственное действие - запуск демона
	app.Action = func(c *cli.Context) error {

		// настраиваем логирование в файл
		if logFileFlag != "" {
			// добавляем в имя лог файла дату и время
			if strings.Contains(logFileFlag, "%s") {
				logFileFlag = fmt.Sprintf(logFileFlag, time.Now().Format("2006_01_02_150405"))
			}

			f, err := os.OpenFile(logFileFlag, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
			if err != nil {
				myerr := myerror.WithCause("6020", "Error open log file", "os.OpenFile", fmt.Sprintf("logFileFlag='%s'", logFileFlag), "", err.Error())
				mylog.PrintfErrorStd(fmt.Sprintf("%+v", myerr))
				os.Exit(1)
			}
			// закрываем по выходу из демона
			if f != nil {
				defer f.Close()
			}

			// Параллельно пишем в os.Stderr и файл
			wrt := io.MultiWriter(os.Stderr, f)

			// Переопределяем глобальный логер на кастомный
			mylog.InitLogger(wrt)
		} else {
			mylog.InitLogger(os.Stderr)
		}

		mylog.PrintfInfoStd(fmt.Sprintf("HTTP Server Starting Up. Version '%s'. Log file '%s'", app.Version, logFileFlag))

		// проверяем флаги
		err := checkFlags()
		if err != nil {
			mylog.PrintfErrorStd(fmt.Sprintf("%+v", err))
			return err
		}

		// сформируем byte ключ из строки
		var JwtKey []byte
		if JwtKeyFlag == "" {
			JwtKey = nil
		} else {
			JwtKey = []byte(JwtKeyFlag)
		}

		// Инициализируем демон
		dmn, err := daemon.New(
			httpConfigFileFlag,
			listenStringFlag,
			HTTPUserIDFlag,
			HTTPUserPwdFlag,
			JwtKey)
		if err != nil {
			mylog.PrintfErrorStd(fmt.Sprintf("%+v", err))
			return err
		}

		// Стартуем демон
		err = dmn.Run()
		if err != nil {
			mylog.PrintfErrorStd(fmt.Sprintf("%+v", err))
			return err
		}

		mylog.PrintfInfoStd("HTTP Server is Shutdown")
		return nil
	}

	// Запускаем приложение
	err := app.Run(os.Args)
	if err != nil {
		mylog.PrintfErrorStd(fmt.Sprintf("%+v", err))
		os.Exit(1)
	}

	os.Exit(0)
}
