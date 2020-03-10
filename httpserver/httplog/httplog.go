package httplog

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"net/http"
	"net/http/httputil"

	myerror "github.com/romapres2010/httpserver/error"
	mylog "github.com/romapres2010/httpserver/log"
)

// Logger represent аn HTTP logger
type Logger struct {
	file *os.File // файл логирования HTTP вызовов
	cfg  *Config  // файл конфигурации
}

// Config represent аn HTTP logger config
type Config struct {
	Enable     bool   // состояние логирования
	LogInReq   bool   // логировать входящие запросы
	LogOutReq  bool   // логировать исходящие запросы
	LogInResp  bool   // логировать входящие ответы
	LogOutResp bool   // логировать исходящие ответы
	LogBody    bool   // логировать тело запроса
	FileName   string // наименование файл логирования
}

// NewLogger - создает новый Logger
func NewLogger(ctx context.Context, cfg *Config) (*Logger, error) {

	log := &Logger{
		cfg: cfg,
	}

	if cfg != nil && cfg.FileName != "" {
		// добавляем в имя лог файла дату и время
		if strings.Contains(cfg.FileName, "%s") {
			cfg.FileName = fmt.Sprintf(cfg.FileName, time.Now().Format("2006_01_02_150405"))
		}

		// Открываем файл для логирования
		f, err := os.OpenFile(cfg.FileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			myerr := myerror.WithCause("6020", "Error open HTTP log file: Filename", err, cfg.FileName)
			mylog.PrintfErrorInfo(myerr)
			return nil, myerr
		}

		// сохраняем открытый дескриптор файла логирования
		log.file = f
	}

	return log, nil
}

// Close Logger
func (log *Logger) Close() {
	if log.file != nil {
		_ = log.file.Close()
	}
}

// LogHTTPOutRequest process HTTP logging for Out request
func (log *Logger) LogHTTPOutRequest(ctx context.Context, req *http.Request, reqID uint64) error {
	if log.cfg.Enable && log.file != nil {
		if req != nil && log.cfg.LogOutReq {
			dump, err := httputil.DumpRequestOut(req, log.cfg.LogBody)
			if err != nil {
				myerr := myerror.WithCause("8020", "Error dump HTTP Request: reqID", err, reqID)
				mylog.PrintfErrorInfo(myerr)
				return myerr
			}
			fmt.Fprintf(log.file, "'%s' Out Request '%v' BEGIN ==================================================================== \n", mylog.GetTimestampStr(), reqID)
			fmt.Fprintf(log.file, "%+v\n", string(dump))
			fmt.Fprintf(log.file, "'%s' Out Request '%v' END ==================================================================== \n", mylog.GetTimestampStr(), reqID)
		}
	}
	return nil
}

// LogHTTPInResponse process HTTP logging for In response
func (log *Logger) LogHTTPInResponse(ctx context.Context, resp *http.Response, reqID uint64) error {
	if log.cfg.Enable && log.file != nil {
		if resp != nil && log.cfg.LogInResp {
			dump, err := httputil.DumpResponse(resp, log.cfg.LogBody)
			if err != nil {
				myerr := myerror.WithCause("8020", "Error dump HTTP Request: reqID", err, reqID)
				mylog.PrintfErrorInfo(myerr)
				return myerr
			}
			fmt.Fprintf(log.file, "'%s' Out Response '%v' BEGIN ==================================================================== \n", mylog.GetTimestampStr(), reqID)
			fmt.Fprintf(log.file, "%+v\n", string(dump))
			fmt.Fprintf(log.file, "'%s' Out Response '%v' End ==================================================================== \n", mylog.GetTimestampStr(), reqID)
		}
	}
	return nil
}

// LogHTTPInRequest process HTTP logging for In request
func (log *Logger) LogHTTPInRequest(ctx context.Context, req *http.Request, reqID uint64) error {
	if log.cfg.Enable && log.file != nil {
		if req != nil && log.cfg.LogInReq {
			dump, err := httputil.DumpRequest(req, log.cfg.LogBody)
			if err != nil {
				myerr := myerror.WithCause("8020", "Error dump HTTP Request: reqID", err, reqID)
				mylog.PrintfErrorInfo(myerr)
				return myerr
			}
			fmt.Fprintf(log.file, "'%s' In Request '%v' BEGIN ==================================================================== \n", mylog.GetTimestampStr(), reqID)
			fmt.Fprintf(log.file, "%+v\n", string(dump))
			fmt.Fprintf(log.file, "'%s' In Request '%v' End ==================================================================== \n", mylog.GetTimestampStr(), reqID)
		}
	}
	return nil
}

// LogHTTPOutResponse process HTTP logging for Oout Response
func (log *Logger) LogHTTPOutResponse(ctx context.Context, header map[string]string, responseBuf []byte, status int, reqID uint64) error {
	if log.cfg.Enable && log.file != nil && log.cfg.LogOutResp {
		// сформируем буффер с ответом
		dump := make([]byte, 0)

		// добавим статус ответа
		dump = append(dump, []byte(fmt.Sprintf("HTTP %v %s\n", status, http.StatusText(status)))...)

		// соберем все заголовки в буфер для логирования
		if header != nil {
			for k, v := range header {
				dump = append(dump, []byte(fmt.Sprintf("%s: %s\n", k, v))...)
			}
		}

		// Логируем тело
		if log.cfg.LogBody && responseBuf != nil {
			dump = append(dump, []byte("\n")...)
			dump = append(dump, responseBuf...)
		}

		fmt.Fprintf(log.file, "'%s' In Response '%v' BEGIN ==================================================================== \n", mylog.GetTimestampStr(), reqID)
		fmt.Fprintf(log.file, "%+v\n", string(dump))
		fmt.Fprintf(log.file, "'%s' In Response '%v' End ==================================================================== \n", mylog.GetTimestampStr(), reqID)
	}
	return nil
}
