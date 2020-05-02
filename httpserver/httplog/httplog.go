package httplog

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"net/http"
	"net/http/httputil"

	myctx "github.com/romapres2010/httpserver/ctx"
	myerror "github.com/romapres2010/httpserver/error"
	mylog "github.com/romapres2010/httpserver/log"
)

// Logger represent аn HTTP logger
type Logger struct {
	file     *os.File // файл логирования HTTP вызовов
	cfg      *Config  // конфигурационные параметры
	fileName string   // наименование файл логирования
}

// Config represent аn HTTP logger config
type Config struct {
	Enable     bool // состояние логирования
	LogInReq   bool // логировать входящие запросы
	LogOutReq  bool // логировать исходящие запросы
	LogInResp  bool // логировать входящие ответы
	LogOutResp bool // логировать исходящие ответы
	LogBody    bool // логировать тело запроса
}

// New - создает новый Logger
func New(ctx context.Context, cfg *Config, fileName string) (*Logger, error) {

	log := &Logger{
		cfg:      cfg,
		fileName: fileName,
	}

	if cfg != nil && fileName != "" {
		// добавляем в имя лог файла дату и время
		if strings.Contains(fileName, "%s") {
			fileName = fmt.Sprintf(fileName, time.Now().Format("2006_01_02_150405"))
		}

		// Открываем файл для логирования
		file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return nil, myerror.WithCause("6020", "Error open HTTP log file: Filename", err, fileName).PrintfInfo()
		}

		log.file = file // сохраняем дескриптор файла логирования
	}

	return log, nil
}

// SetConfig set new logger config
func (log *Logger) SetConfig(cfg *Config) {
	if cfg != nil {
		mylog.PrintfInfoMsg("Set HTTP loger config: cfg", *cfg)
		log.cfg = cfg
	}
}

// Close Logger
func (log *Logger) Close() error {
	if log.file != nil {
		defer log.file.Close() // ошибку закрытия игнорируем

		// flushing write buffers out to disks
		err := log.file.Sync()

		if err != nil {
			return myerror.WithCause("6020", "Error sync HTTP log file before closing", err).PrintfInfo()
		}
	}
	return nil
}

// LogHTTPOutRequest process HTTP logging for Out request
func (log *Logger) LogHTTPOutRequest(ctx context.Context, req *http.Request) error {
	if log.cfg.Enable && log.file != nil {
		reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

		mylog.PrintfDebugMsg("Logging HTTP out request: reqID", reqID)

		if req != nil && log.cfg.LogOutReq {
			dump, err := httputil.DumpRequestOut(req, log.cfg.LogBody)
			if err != nil {
				return myerror.WithCause("8020", "Error dump HTTP Request: reqID", err, reqID).PrintfInfo()
			}
			fmt.Fprintf(log.file, "'%s' Out Request '%v' BEGIN ==================================================================== \n", mylog.GetTimestampStr(), reqID)
			fmt.Fprintf(log.file, "%+v\n", string(dump))
			fmt.Fprintf(log.file, "'%s' Out Request '%v' END ==================================================================== \n", mylog.GetTimestampStr(), reqID)
		}
	}
	return nil
}

// LogHTTPInResponse process HTTP logging for In response
func (log *Logger) LogHTTPInResponse(ctx context.Context, resp *http.Response) error {
	if log.cfg.Enable && log.file != nil {
		reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context
		mylog.PrintfDebugMsg("Logging HTTP in response: reqID", reqID)

		if resp != nil && log.cfg.LogInResp {
			dump, err := httputil.DumpResponse(resp, log.cfg.LogBody)
			if err != nil {
				return myerror.WithCause("8020", "Error dump HTTP Request: reqID", err, reqID).PrintfInfo()
			}
			fmt.Fprintf(log.file, "'%s' In Response '%v' BEGIN ==================================================================== \n", mylog.GetTimestampStr(), reqID)
			fmt.Fprintf(log.file, "%+v\n", string(dump))
			fmt.Fprintf(log.file, "'%s' In Response '%v' End ==================================================================== \n", mylog.GetTimestampStr(), reqID)
		}
	}
	return nil
}

// LogHTTPInRequest process HTTP logging for In request
func (log *Logger) LogHTTPInRequest(ctx context.Context, req *http.Request) error {
	if log.cfg.Enable && log.file != nil {
		reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context
		mylog.PrintfDebugMsg("Logging HTTP in request: reqID", reqID)

		if req != nil && log.cfg.LogInReq {
			dump, err := httputil.DumpRequest(req, log.cfg.LogBody)
			if err != nil {
				return myerror.WithCause("8020", "Error dump HTTP Request: reqID", err, reqID).PrintfInfo()
			}
			fmt.Fprintf(log.file, "'%s' In Request '%v' BEGIN ==================================================================== \n", mylog.GetTimestampStr(), reqID)
			fmt.Fprintf(log.file, "%+v\n", string(dump))
			fmt.Fprintf(log.file, "'%s' In Request '%v' End ==================================================================== \n", mylog.GetTimestampStr(), reqID)
		}
	}
	return nil
}

// LogHTTPOutResponse process HTTP logging for Out Response
func (log *Logger) LogHTTPOutResponse(ctx context.Context, header map[string]string, responseBuf []byte, status int) error {
	if log.cfg.Enable && log.file != nil && log.cfg.LogOutResp {
		reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context
		mylog.PrintfDebugMsg("Logging HTTP out response: reqID", reqID)

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

		// Добавим в буффер тело
		if log.cfg.LogBody && responseBuf != nil {
			dump = append(dump, []byte("\n")...)
			dump = append(dump, responseBuf...)
		}

		fmt.Fprintf(log.file, "'%s' Out Response '%v' BEGIN ==================================================================== \n", mylog.GetTimestampStr(), reqID)
		fmt.Fprintf(log.file, "%+v\n", string(dump))
		fmt.Fprintf(log.file, "'%s' Out Response '%v' End ==================================================================== \n", mylog.GetTimestampStr(), reqID)
	}
	return nil
}
