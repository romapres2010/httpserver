package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli"

	"github.com/romapres2010/httpserver/daemon"
	myerror "github.com/romapres2010/httpserver/error"
	mylog "github.com/romapres2010/httpserver/log"
)

// Параметры, подменяемые компилятором при сборке бинарника
var (
	version   = "0.0.5"
	commit    = "unset"
	buildTime = "unset"
)

// Глобальные переменные, в которые парсятся входные флаги
var (
	httpConfigFileFlag string
	logFileFlag        string
	listenStringFlag   string
	debugFlag          string
	httpUserIDFlag     string
	httpUserPwdFlag    string
	jwtKeyFlag         string
)

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
		Required:    true,
		Destination: &httpUserIDFlag,
	},
	cli.StringFlag{
		Name:        "httppassword, httppwd",
		Usage:       "User password for access to HTTP server",
		Required:    true,
		Destination: &httpUserPwdFlag,
	},
	cli.StringFlag{
		Name:        "jwtkey, jwtk",
		Usage:       "JSON web token secret key",
		Required:    false,
		Destination: &jwtKeyFlag,
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
		Required:    true,
		Destination: &logFileFlag,
		Value:       "httpserver_%s.log", // %s время старта программы
	},
}

//main - represent main function
func main() {
	// Create new Application
	app := cli.NewApp()
	app.Name = "HTTP Server"
	app.Version = fmt.Sprintf("%s, commit '%s', build time '%s'", version, commit, buildTime)
	app.Author = "Roman Presnyakov"
	app.Email = "romapres@mail.ru"
	app.Flags = flags // присваиваем ранее определенные флаги
	app.Writer = os.Stderr

	// Определяем единственное действие - запуск демона
	app.Action = func(c *cli.Context) error {

		// настраиваем параллельное логирование в файл
		if logFileFlag != "" {
			// добавляем в имя лог файла дату и время
			if strings.Contains(logFileFlag, "%s") {
				logFileFlag = fmt.Sprintf(logFileFlag, time.Now().Format("2006_01_02_150405"))
			}

			f, err := os.OpenFile(logFileFlag, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
			if err != nil {
				myerr := myerror.WithCause("6020", "Error open log file", "os.OpenFile", fmt.Sprintf("logFileFlag='%s'", logFileFlag), "", err.Error())
				mylog.PrintfErrorStd(fmt.Sprintf("%+v", myerr))
				return myerr
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

		mylog.PrintfInfoStd(fmt.Sprintf("HTTP Server is starting up. Version '%s'. Log file '%s'", app.Version, logFileFlag))

		// Установим фильтр логирования
		if debugFlag != "" {
			mylog.PrintfInfoStd("Set log level", debugFlag)
			switch debugFlag {
			case "DEBUG", "WARN", "ERROR", "INFO":
				mylog.NewFilter(debugFlag)
			default:
				myerr := myerror.New("9001", fmt.Sprintf("Incorrect debugFlag '%s'. Only avaliable: DEBUG, WARN, INFO, ERROR.", debugFlag), "", "")
				mylog.PrintfErrorStd(fmt.Sprintf("%+v", myerr))
				return myerr
			}
		}

		// Создаем новый демон
		daemon, err := daemon.New(httpConfigFileFlag, listenStringFlag, httpUserIDFlag, httpUserPwdFlag, []byte(jwtKeyFlag))
		if err != nil {
			mylog.PrintfErrorStd(fmt.Sprintf("%+v", err))
			return err
		}

		// Стартуем демон и ожидаем завершения
		err = daemon.Run()
		if err != nil {
			mylog.PrintfErrorStd(fmt.Sprintf("%+v", err))
			return err
		}

		mylog.PrintfInfoStd("HTTP Server is shutdown")
		return nil
	}

	// Запускаем приложение
	err := app.Run(os.Args)
	if err != nil {
		log.Printf("%+v", err)
		os.Exit(1)
	}

	os.Exit(0)
}
