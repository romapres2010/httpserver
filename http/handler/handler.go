package handler

// Обертка над стандартным пакетом http используется для изоляции HTTP кода и обработчиков

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	myerror "github.com/romapres2010/httpserver/error"
	httplog "github.com/romapres2010/httpserver/http/httplog"
	mylog "github.com/romapres2010/httpserver/log"
)

// Header represent temporary HTTP header
type Header map[string]string

// Handler - представляет обертку над обработчиком для реализации всех HTTP методов
type Handler struct {
	Ctx        context.Context     // корневой контекст при инициации сервиса
	Cancel     context.CancelFunc  // функция закрытия глобального контекста
	Cfg        *Config             // Конфигурационные параметры
	HTTPLogger *httplog.HTTPLogger // сервис логирования HTTP
}

// Config repsent HTTP Handler configurations
type Config struct {
	MaxBodyBytes int    // максимальный размер тела сообщения - 0 не ограничено
	UseTLS       bool   // признак использования SSL
	UseHSTS      bool   // использовать HTTP Strict Transport Security
	UseJWT       bool   // use JSON web token (JWT)
	JWTExpiresAt int    // JWT expiry time in seconds - 0 without restriction
	JwtKey       []byte // JWT secret key
	AuthType     string // тип аутентификации NONE, INTERNAL, MSAD
	HTTPUserID   string // пользователь для HTTP Basic Authentication передается через командую строку
	HTTPUserPwd  string // пароль для HTTP Basic Authentication передается через командую строку
	MSADServer   string // MS Active Directory server
	MSADPort     int    // MS Active Directory Port
	MSADBaseDN   string // MS Active Directory BaseDN
	MSADSecurity int    // MS Active Directory Security: SecurityNone, SecurityTLS, SecurityStartTLS
}

/*
// NewHandler - создает новый handler
// =====================================================================
func NewHandler(ctx context.Context, queue *queue.Service, xml *xml.Service, eventLogger *EventLogger, cfg *HandlerConfig) *Handler {

	h := &Handler{
		queue:       queue,
		xml:         xml,
		cfg:         cfg,
		eventLogger: eventLogger,
	}

	// создаем контекст с отменой
	// h.cancel используется при остановке сервера для остановки всех обработчиков и текущих запросов
	if ctx == nil {
		h.Ctx, h.Cancel = context.WithCancel(context.Background())
	} else {
		h.Ctx, h.Cancel = context.WithCancel(ctx)
	}

	return h
}
*/

// RecoverWrap cover handler functions with panic recoverer
// =====================================================================
func (h *Handler) RecoverWrap(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var err error
		defer func() {
			r := recover()
			if r != nil {
				switch t := r.(type) {
				case string:
					err = errors.New(t)
				case error:
					err = t
				default:
					err = errors.New("UNKNOWN ERROR")
				}

				myerr := myerror.New("8888", fmt.Sprintf("UNKNOWN ERROR - recover from panic \n %+v", err.Error()), "RecoverWrap", "")
				h.LogError(myerr, w, http.StatusInternalServerError, 0)
			}
		}()
		handlerFunc.ServeHTTP(w, r)
	})
}

// Process - represent server common task in process incoming HTTP request
// =====================================================================
func (h *Handler) Process(method string, w http.ResponseWriter, r *http.Request, fn func(requestBuf []byte, reqID uint64) ([]byte, Header, int, error)) {
	var err error
	var reqID uint64 // уникальный номер Request

	// логируем входящий HTTP запрос, одновременно получаем ID Request
	if h.HTTPLogger != nil {
		reqID, _ = h.HTTPLogger.LogHTTPInRequest(h.Ctx, r)
		mylog.PrintfDebugStd("Logging HTTP request", reqID)
	}

	// проверим разрешенный метод
	mylog.PrintfDebugStd("Check allowed HTTP method", reqID, r.Method, method)
	if r.Method != method {
		myerr := myerror.New("8000", fmt.Sprintf("Not allowed method '%s', reqID '%v'", r.Method, reqID), "", "")
		h.LogError(myerr, w, http.StatusMethodNotAllowed, reqID)
		return
	}

	// Если включен режим аутентификации без использования JWT токена, то проверять пользователя и пароль каждый раз
	mylog.PrintfDebugStd("Check authentication method", reqID, h.Cfg.AuthType)
	if (h.Cfg.AuthType == "INTERNAL" || h.Cfg.AuthType == "MSAD") && !h.Cfg.UseJWT {
		mylog.PrintfDebugStd(fmt.Sprintf("JWT is of. Need Authentication, reqID '%v'", reqID))
		if _, err = h._checkBasicAuthentication(r); err != nil {
			h.LogError(err, w, http.StatusUnauthorized, reqID)
			return
		}
	}

	// Если используем JWT - проверим токен
	if h.Cfg.UseJWT {
		mylog.PrintfDebugStd("JWT is on. Check JSON web token", reqID)
		if _, err = h._checkJWTFromCookie(r); err != nil {
			h.LogError(err, w, http.StatusUnauthorized, reqID)
			return
		}
	}

	// считаем тело запроса
	mylog.PrintfDebugStd("Reading request body", reqID)
	requestBuf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		myerr := myerror.WithCause("8001", fmt.Sprintf("Failed to read HTTP body, reqID '%v'", reqID), "ioutil.ReadAll()", "", "", err.Error())
		h.LogError(myerr, w, http.StatusInternalServerError, reqID)
		return
	}
	mylog.PrintfDebugStd("Read request body", reqID, len(requestBuf))

	// вызываем обработчик
	mylog.PrintfDebugStd("Calling external function handler", reqID)
	responseBuf, header, status, err := fn(requestBuf, reqID)
	if err != nil {
		h.LogError(err, w, status, reqID)
		return
	}

	// use HSTS Strict-Transport-Security
	if h.Cfg.UseHSTS {
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	}

	// Content-Type - требуемый тип контента в ответе
	responseContentType := r.Header.Get("Content-Type-Response")
	// Если не задан Content-Type-Response то берем его из запроса
	if responseContentType == "" {
		responseContentType = r.Header.Get("Content-Type")
	}
	header["Content-Type"] = responseContentType

	// Логируем ответ в файл
	if h.HTTPLogger != nil {
		mylog.PrintfDebugStd("Logging HTTP response", reqID)
		h.HTTPLogger.LogHTTPInResponse(h.Ctx, header, responseBuf, status, reqID)
	}

	// запишем ответ в заголовок
	mylog.PrintfDebugStd("Set HTTP response headers", reqID)
	if header != nil {
		for key, h := range header {
			w.Header().Set(key, h)
		}
	}

	// запишем статус
	mylog.PrintfDebugStd("Set HTTP response status", reqID, http.StatusText(status))
	w.WriteHeader(status)

	// записываем буфер в ответ
	if responseBuf != nil {
		mylog.PrintfDebugStd("Writing HTTP response body", reqID, len(responseBuf))
		respWrittenLen, err := w.Write(responseBuf)
		if err != nil {
			myerr := myerror.WithCause("8002", fmt.Sprintf("Failed to write HTTP repsonse, reqID '%v'", reqID), "http.Write()", "", "", err.Error())
			h.LogError(myerr, w, http.StatusInternalServerError, reqID)
			return
		}
		mylog.PrintfDebugStd("Written HTTP response", reqID, respWrittenLen)
	}
}

// LogError - log error into header, body and log
// =====================================================================
func (h *Handler) LogError(err error, w http.ResponseWriter, status int, reqID uint64) {
	// записываем в лог кроме статусов StatusNotFound
	if status != http.StatusNotFound {

		// полная ошибка со стеком вызова
		errM := fmt.Sprintf("RequestID '%v', err %+v", reqID, err)

		mylog.PrintfErrorStd1(errM)

		// дополнительно записываем в заголовок ответа
		if w != nil {
			// если тип ошибки myerror.Error, то возьмем коды из нее
			if myerr, ok := err.(*myerror.Error); ok {
				w.Header().Set("Errcode", myerr.Code)
				w.Header().Set("Errmes", myerr.Msg)
				w.Header().Set("Causeerrcode", fmt.Sprintf("%v", myerr.CauseCode))
				w.Header().Set("Causeerrmes", fmt.Sprintf("%v", myerr.CauseMes))
			} else {
				w.Header().Set("Errcode", "")
				w.Header().Set("Errmes", err.Error())
			}
			w.Header().Set("Errtrace", errM)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("RequestID", fmt.Sprintf("%v", reqID))
			w.WriteHeader(status)

			fmt.Fprintln(w, errM)
		}
	} else { // http.StatusNotFound
		// дополнительно записываем в заголовок ответа
		if w != nil {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("RequestID", fmt.Sprintf("%v", reqID))
			w.WriteHeader(status)
		}
	}
}
