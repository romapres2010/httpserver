package httpservice

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	myctx "github.com/romapres2010/httpserver/ctx"
	myerror "github.com/romapres2010/httpserver/error"
	httplog "github.com/romapres2010/httpserver/httpserver/httplog"
	mylog "github.com/romapres2010/httpserver/log"
)

// HTTPLogHandler handle HTTP log mode
func (s *Service) HTTPLogHandler(w http.ResponseWriter, r *http.Request) {
	mylog.PrintfDebugMsg("START   ==================================================================================")

	// Запускаем обработчик, возврат ошибки игнорируем
	_ = s.process("POST", w, r, func(ctx context.Context, requestBuf []byte, buf []byte) ([]byte, Header, int, error) {
		reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

		mylog.PrintfDebugMsg("START: reqID", reqID)

		// новый конфиг для HTTP Loger
		logCfg := &httplog.Config{}

		{ // обрабатываем HTTP-log из заголовка
			HTTPLogStr := r.Header.Get("HTTP-Log")
			if HTTPLogStr != "" {
				switch strings.ToUpper(HTTPLogStr) {
				case "TRUE":
					logCfg.Enable = true
				case "FALSE":
					logCfg.Enable = false
				default:
					myerr := myerror.New("5014", "Incorrect boolean for 'HTTP-Log': reqID,  Value", reqID, HTTPLogStr)
					return nil, nil, http.StatusBadRequest, myerr
				}
			} else {
				logCfg.Enable = false
			}
		} // обрабатываем HTTP-log из заголовка

		{ // обрабатываем HTTP-log-type из заголовка
			HTTPLogTypeStr := r.Header.Get("HTTP-Log-Type")
			// логировать входящие запросы
			if strings.Index(HTTPLogTypeStr, "INREQ") >= 0 {
				logCfg.LogInReq = true
			}
			// логировать исходящие запросы
			if strings.Index(HTTPLogTypeStr, "OUTREQ") >= 0 {
				logCfg.LogOutReq = true
			}
			// логировать входящие ответы
			if strings.Index(HTTPLogTypeStr, "INRESP") >= 0 {
				logCfg.LogInResp = true
			}
			// логировать исходящие ответы
			if strings.Index(HTTPLogTypeStr, "OUTRESP") >= 0 {
				logCfg.LogOutResp = true
			}
			// логировать тело запроса
			if strings.Index(HTTPLogTypeStr, "BODY") >= 0 {
				logCfg.LogBody = true
			}
		} // обрабатываем HTTP-log-type из заголовка

		// формируем ответ
		header := Header{}
		header["Errcode"] = "0"
		header["RequestID"] = fmt.Sprintf("%v", reqID)

		// устанавливаем новый конфиг для HTTP Loger
		s.logger.SetConfig(logCfg)

		mylog.PrintfDebugMsg("SUCCESS", reqID)
		return requestBuf, header, http.StatusOK, nil
	})

	mylog.PrintfDebugMsg("SUCCESS ==================================================================================")
}

// HTTPErrorLogHandler handle loging error into HTTP response
func (s *Service) HTTPErrorLogHandler(w http.ResponseWriter, r *http.Request) {
	mylog.PrintfDebugMsg("START   ==================================================================================")

	// Запускаем обработчик, возврат ошибки игнорируем
	_ = s.process("POST", w, r, func(ctx context.Context, requestBuf []byte, buf []byte) ([]byte, Header, int, error) {
		reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

		mylog.PrintfDebugMsg("START: reqID", reqID)

		{ // обрабатываем HTTP-Err-Log из заголовка
			HTTPErrLogStr := r.Header.Get("HTTP-Err-Log")
			// логировать ошибку в заголовок ответа
			if strings.Index(HTTPErrLogStr, "HEADER") >= 0 {
				s.cfg.HTTPErrorLogHeader = true
			} else {
				s.cfg.HTTPErrorLogHeader = false
			}
			// логировать ошибку в тело ответа
			if strings.Index(HTTPErrLogStr, "BODY") >= 0 {
				s.cfg.HTTPErrorLogBody = true
			} else {
				s.cfg.HTTPErrorLogBody = false
			}
		} // обрабатываем HTTP-Err-Log из заголовка

		mylog.PrintfInfoMsg("Set HTTP Error: HTTPErrorLogHeader, HTTPErrorLogBody", s.cfg.HTTPErrorLogHeader, s.cfg.HTTPErrorLogBody)

		// формируем ответ
		header := Header{}
		header["Errcode"] = "0"
		header["RequestID"] = fmt.Sprintf("%v", reqID)

		mylog.PrintfDebugMsg("SUCCESS", reqID)
		return requestBuf, header, http.StatusOK, nil
	})

	mylog.PrintfDebugMsg("SUCCESS ==================================================================================")
}

// LogLevelHandler handle loging filter
func (s *Service) LogLevelHandler(w http.ResponseWriter, r *http.Request) {
	mylog.PrintfDebugMsg("START   ==================================================================================")

	// Запускаем обработчик, возврат ошибки игнорируем
	_ = s.process("POST", w, r, func(ctx context.Context, requestBuf []byte, buf []byte) ([]byte, Header, int, error) {
		reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

		mylog.PrintfDebugMsg("START: reqID", reqID)

		LogLevelStr := r.Header.Get("Log-Level-Filter")
		switch LogLevelStr {
		case "DEBUG", "ERROR", "INFO":
			mylog.SetFilter(LogLevelStr)
		default:
			myerr := myerror.New("9001", "Incorrect log level. Only avaliable: DEBUG, INFO, ERROR.", LogLevelStr)
			mylog.PrintfErrorMsg(fmt.Sprintf("%+v", myerr))
			return nil, nil, http.StatusBadRequest, myerr
		}

		mylog.PrintfInfoMsg("Set log level", s.cfg.HTTPErrorLogHeader, s.cfg.HTTPErrorLogBody)

		// формируем ответ
		header := Header{}
		header["Errcode"] = "0"
		header["RequestID"] = fmt.Sprintf("%v", reqID)

		mylog.PrintfDebugMsg("SUCCESS", reqID)
		return requestBuf, header, http.StatusOK, nil
	})

	mylog.PrintfDebugMsg("SUCCESS ==================================================================================")
}
