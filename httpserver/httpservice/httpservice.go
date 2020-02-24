package httpservice

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	myerror "github.com/romapres2010/httpserver/error"
	httplog "github.com/romapres2010/httpserver/httpserver/httplog"
	mylog "github.com/romapres2010/httpserver/log"
)

// Header represent temporary HTTP header
type Header map[string]string

// Handler represent HTTP handler
type Handler struct {
	Path        string
	HundlerFunc func(http.ResponseWriter, *http.Request)
	Method      string
}

// Handlers represent HTTP handlers map
type Handlers map[string]Handler

// Service represent HTTP service
type Service struct {
	сtx      context.Context    // корневой контекст при инициации сервиса
	cancel   context.CancelFunc // функция закрытия глобального контекста
	cfg      *Config            // Конфигурационные параметры
	Handlers Handlers           // список обработчиков

	// вложенные сервисы
	logger *httplog.Logger // сервис логирования HTTP
}

// Config repsent HTTP Service configurations
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

	// конфигурация вложенных сервисов
	LogCfg httplog.Config // конфигурация HTTP логирования
}

// New create new HTTP service
func New(ctx context.Context, cfg *Config) (*Service, *httplog.Logger, error) {
	var err error

	service := &Service{
		cfg: cfg,
	}

	// создаем контекст с отменой
	if ctx == nil {
		service.сtx, service.cancel = context.WithCancel(context.Background())
	} else {
		service.сtx, service.cancel = context.WithCancel(ctx)
	}

	// создаем обработчик для логирования HTTP
	if service.logger, err = httplog.NewLogger(service.сtx, &cfg.LogCfg); err != nil {
		return nil, nil, err
	}

	// Наполним список обрабочиков
	service.Handlers = map[string]Handler{
		"/echo":    Handler{"/echo", service.RecoverWrap(service.EchoHandler), "POST"},
		"/signin":  Handler{"/signin", service.RecoverWrap(service.SinginHandler), "POST"},
		"/refresh": Handler{"/refresh", service.RecoverWrap(service.JWTRefreshHandler), "POST"},
	}

	return service, service.logger, nil
}

// Shutdown shutting down service
func (s *Service) Shutdown() {
	// Закрываем Logger для корректного закрытия лог файла
	if s.logger != nil {
		s.logger.Close()
	}

	defer s.cancel()
}

// RecoverWrap cover handler functions with panic recoverer
func (s *Service) RecoverWrap(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// объявляем функцию восстановления после паники
		defer func() {
			var err error
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
				// формируем текст ошибки для логирования
				myerr := myerror.New("8888", fmt.Sprintf("UNKNOWN ERROR - recover from panic \n %+v", err.Error()), "RecoverWrap", "")
				// кастомное логирование ошибки
				s.LogError(myerr, w, http.StatusInternalServerError, 0)
			}
		}()

		// вызываем обработчик
		if handlerFunc != nil {
			handlerFunc(w, r)
		}
	})
}

// Process - represent server common task in process incoming HTTP request
func (s *Service) Process(method string, w http.ResponseWriter, r *http.Request, fn func(requestBuf []byte, reqID uint64) ([]byte, Header, int, error)) {
	var err error
	var reqID uint64 // уникальный номер Request

	// логируем входящий HTTP запрос, одновременно получаем ID Request
	if s.logger != nil {
		reqID, _ = s.logger.LogHTTPInRequest(s.сtx, r)
		mylog.PrintfDebugStd("Logging HTTP request", reqID)
	}

	// проверим разрешенный метод
	mylog.PrintfDebugStd("Check allowed HTTP method", reqID, r.Method, method)
	if r.Method != method {
		myerr := myerror.New("8000", fmt.Sprintf("Not allowed method '%s', reqID '%v'", r.Method, reqID), "", "")
		s.LogError(myerr, w, http.StatusMethodNotAllowed, reqID)
		return
	}

	// Если включен режим аутентификации без использования JWT токена, то проверять пользователя и пароль каждый раз
	mylog.PrintfDebugStd("Check authentication method", reqID, s.cfg.AuthType)
	if (s.cfg.AuthType == "INTERNAL" || s.cfg.AuthType == "MSAD") && !s.cfg.UseJWT {
		mylog.PrintfDebugStd(fmt.Sprintf("JWT is of. Need Authentication, reqID '%v'", reqID))
		if _, err = s._checkBasicAuthentication(r); err != nil {
			s.LogError(err, w, http.StatusUnauthorized, reqID)
			return
		}
	}

	// Если используем JWT - проверим токен
	if s.cfg.UseJWT {
		mylog.PrintfDebugStd("JWT is on. Check JSON web token", reqID)
		if _, err = s._checkJWTFromCookie(r); err != nil {
			s.LogError(err, w, http.StatusUnauthorized, reqID)
			return
		}
	}

	// считаем тело запроса
	mylog.PrintfDebugStd("Reading request body", reqID)
	requestBuf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		myerr := myerror.WithCause("8001", fmt.Sprintf("Failed to read HTTP body, reqID '%v'", reqID), "ioutil.ReadAll()", "", "", err.Error())
		s.LogError(myerr, w, http.StatusInternalServerError, reqID)
		return
	}
	mylog.PrintfDebugStd("Read request body", reqID, len(requestBuf))

	// вызываем обработчик
	mylog.PrintfDebugStd("Calling external function handler", reqID)
	responseBuf, header, status, err := fn(requestBuf, reqID)
	if err != nil {
		s.LogError(err, w, status, reqID)
		return
	}

	// use HSTS Strict-Transport-Security
	if s.cfg.UseHSTS {
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
	if s.logger != nil {
		mylog.PrintfDebugStd("Logging HTTP response", reqID)
		s.logger.LogHTTPInResponse(s.сtx, header, responseBuf, status, reqID)
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
			s.LogError(myerr, w, http.StatusInternalServerError, reqID)
			return
		}
		mylog.PrintfDebugStd("Written HTTP response", reqID, respWrittenLen)
	}
}

// LogError - log error into header, body and log
// =====================================================================
func (s *Service) LogError(err error, w http.ResponseWriter, status int, reqID uint64) {
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
