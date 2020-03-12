package httpservice

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync/atomic"

	myerror "github.com/romapres2010/httpserver/error"
	httplog "github.com/romapres2010/httpserver/httpserver/httplog"
	myjwt "github.com/romapres2010/httpserver/jwt"
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

// уникальный номер HTTP запроса
var requestID uint64

// Service represent HTTP service
type Service struct {
	сtx      context.Context    // корневой контекст при инициации сервиса
	cancel   context.CancelFunc // функция закрытия глобального контекста
	cfg      *Config            // конфигурационные параметры
	Handlers Handlers           // список обработчиков

	// вложенные сервисы
	logger *httplog.Logger // сервис логирования HTTP
}

// Config repsent HTTP Service configurations
type Config struct {
	MaxBodyBytes       int    // HTTP max body bytes - default 0 - unlimited
	UseTLS             bool   // use SSL
	UseHSTS            bool   // use HTTP Strict Transport Security
	UseJWT             bool   // use JSON web token (JWT)
	JWTExpiresAt       int    // JWT expiry time in seconds - 0 without restriction
	JwtKey             []byte // JWT secret key
	AuthType           string // тип аутентификации NONE, INTERNAL, MSAD
	HTTPUserID         string // пользователь для HTTP Basic Authentication передается через командую строку
	HTTPUserPwd        string // пароль для HTTP Basic Authentication передается через командую строку
	MSADServer         string // MS Active Directory server
	MSADPort           int    // MS Active Directory Port
	MSADBaseDN         string // MS Active Directory BaseDN
	MSADSecurity       int    // MS Active Directory Security: SecurityNone, SecurityTLS, SecurityStartTLS
	HTTPErrorLogHeader bool   // логирование ошибок в заголовок HTTP ответа
	HTTPErrorLogBody   bool   // логирование ошибок в тело HTTP ответа
	HTTPLog            bool   // логирование HTTP трафика в файл
	HTTPLogFileName    string // файл логирование HTTP трафика

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
	if service.logger, err = httplog.New(service.сtx, &cfg.LogCfg, cfg.HTTPLogFileName); err != nil {
		return nil, nil, err
	}

	// Наполним список обрабочиков
	service.Handlers = map[string]Handler{
		"/echo":       Handler{"/echo", service.RecoverWrap(service.EchoHandler), "POST"},
		"/signin":     Handler{"/signin", service.RecoverWrap(service.SinginHandler), "POST"},
		"/refresh":    Handler{"/refresh", service.RecoverWrap(service.JWTRefreshHandler), "POST"},
		"/httplog":    Handler{"/httplog", service.RecoverWrap(service.HTTPLogHandler), "POST"},
		"/httperrlog": Handler{"/httperrlog", service.RecoverWrap(service.HTTPErrorLogHandler), "POST"},
		"/loglevel":   Handler{"/loglevel", service.RecoverWrap(service.LogLevelHandler), "POST"},
	}

	return service, service.logger, nil
}

// GetNextRequestID - запросить номер следующего HTTP запроса
func GetNextRequestID() uint64 {
	return atomic.AddUint64(&requestID, 1)
}

// Shutdown shutting down service
func (s *Service) Shutdown() {
	// Закрываем Logger для корректного закрытия лог файла
	if s.logger != nil {
		s.logger.Close()
	}
	defer s.cancel() // вызываем функцию закрытия контекста
}

// RecoverWrap cover handler functions with panic recoverer
func (s *Service) RecoverWrap(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// объявляем функцию восстановления после паники
		defer func() {
			var myerr error
			r := recover()
			if r != nil {
				msg := "HTTP Handler recover from panic"
				switch t := r.(type) {
				case string:
					myerr = myerror.New("8888", msg, t)
				case error:
					myerr = myerror.WithCause("8888", msg, t)
				default:
					myerr = myerror.New("8888", msg)
				}
				// расширенное логирование ошибки в контексте HTTP
				s.processError(myerr, w, http.StatusInternalServerError, 0)
			}
		}()

		// вызываем обработчик
		if handlerFunc != nil {
			handlerFunc(w, r)
		}
	})
}

// Process - represent server common task in process incoming HTTP request
func (s *Service) Process(method string, w http.ResponseWriter, r *http.Request, fn func(requestBuf []byte, reqID uint64) ([]byte, Header, int, error)) error {
	var myerr error

	// Получить уникальный номер HTTP запроса
	reqID := GetNextRequestID()

	// Логируем входящий HTTP запрос
	if s.logger != nil {
		_ = s.logger.LogHTTPInRequest(s.сtx, r, reqID) // При сбое HTTP логирования, делаем системное логирование, но работу не останавливаем
		mylog.PrintfDebugMsg("Logging HTTP in request: reqID", reqID)
	}

	// Проверим разрешенный метод
	mylog.PrintfDebugMsg("Check allowed HTTP method: reqID, request.Method, method", reqID, r.Method, method)
	if r.Method != method {
		myerr = myerror.New("8000", "HTTP method is not allowed: reqID, request.Method, method", reqID, r.Method, method)
		mylog.PrintfErrorInfo(myerr)
		s.processError(myerr, w, http.StatusMethodNotAllowed, reqID) // расширенное логирование ошибки в контексте HTTP
		return myerr
	}

	// Если включен режим аутентификации без использования JWT токена, то проверять пользователя и пароль каждый раз
	mylog.PrintfDebugMsg("Check authentication method: reqID, AuthType", reqID, s.cfg.AuthType)
	if (s.cfg.AuthType == "INTERNAL" || s.cfg.AuthType == "MSAD") && !s.cfg.UseJWT {
		mylog.PrintfDebugMsg("JWT is of. Need Authentication: reqID", reqID)

		// Считаем из заголовка HTTP Basic Authentication
		username, password, ok := r.BasicAuth()
		if !ok {
			myerr := myerror.New("8004", "Header 'Authorization' is not set")
			mylog.PrintfErrorInfo(myerr)
			s.processError(myerr, w, http.StatusUnauthorized, reqID)
			return myerr
		}
		mylog.PrintfDebugMsg("Get Authorization header: username", username)

		// Выполняем аутентификацию
		if myerr = s.checkAuthentication(username, password); myerr != nil {
			mylog.PrintfErrorInfo(myerr)
			s.processError(myerr, w, http.StatusUnauthorized, reqID)
			return myerr
		}
	}

	// Если используем JWT - проверим токен
	if s.cfg.UseJWT {
		mylog.PrintfDebugMsg("JWT is on. Check JSON web token: reqID", reqID)

		// Считаем token из requests cookies
		cookie, err := r.Cookie("token")
		if err != nil {
			myerr := myerror.WithCause("8005", "JWT token does not present in Cookie. You have to authorize first.", err)
			mylog.PrintfErrorInfo(myerr)
			s.processError(myerr, w, http.StatusUnauthorized, reqID) // расширенное логирование ошибки в контексте HTTP
			return myerr
		}

		// Проверим JWT в token
		if myerr = myjwt.CheckJWTFromCookie(cookie, s.cfg.JwtKey); myerr != nil {
			mylog.PrintfErrorInfo(myerr)
			s.processError(myerr, w, http.StatusUnauthorized, reqID) // расширенное логирование ошибки в контексте HTTP
			return myerr
		}
	}

	// Считаем тело запроса
	mylog.PrintfDebugMsg("Reading request body: reqID", reqID)
	requestBuf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		myerr = myerror.WithCause("8001", "Failed to read HTTP body: reqID", err, reqID)
		mylog.PrintfErrorInfo(myerr)
		s.processError(myerr, w, http.StatusInternalServerError, reqID) // расширенное логирование ошибки в контексте HTTP
		return myerr
	}
	mylog.PrintfDebugMsg("Read request body: reqID, len(body)", reqID, len(requestBuf))

	// вызываем обработчик
	mylog.PrintfDebugMsg("Calling external function handler: reqID, function", reqID, fn)
	responseBuf, header, status, myerr := fn(requestBuf, reqID)
	if myerr != nil {
		mylog.PrintfErrorInfo(myerr)
		s.processError(myerr, w, status, reqID) // расширенное логирование ошибки в контексте HTTP
		return myerr
	}

	// use HSTS Strict-Transport-Security
	if s.cfg.UseHSTS {
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	}

	// Логируем ответ в файл
	if s.logger != nil {
		mylog.PrintfDebugMsg("Logging HTTP out response: reqID", reqID)
		_ = s.logger.LogHTTPOutResponse(s.сtx, header, responseBuf, status, reqID) // При сбое HTTP логирования, делаем системное логирование, но работу не останавливаем
	}

	// Записываем заголовок ответа
	mylog.PrintfDebugMsg("Set HTTP response headers: reqID", reqID)
	if header != nil {
		for key, h := range header {
			w.Header().Set(key, h)
		}
	}

	// Записываем HTTP статус ответа
	mylog.PrintfDebugMsg("Set HTTP response status: reqID, Status", reqID, http.StatusText(status))
	w.WriteHeader(status)

	// Записываем тело ответа
	if responseBuf != nil && len(responseBuf) > 0 {
		mylog.PrintfDebugMsg("Writing HTTP response body: reqID, len(body)", reqID, len(responseBuf))
		respWrittenLen, err := w.Write(responseBuf)
		if err != nil {
			myerr = myerror.WithCause("8002", "Failed to write HTTP repsonse: reqID", err)
			mylog.PrintfErrorInfo(myerr)
			s.processError(myerr, w, http.StatusInternalServerError, reqID) // расширенное логирование ошибки в контексте HTTP
			return myerr
		}
		mylog.PrintfDebugMsg("Written HTTP response: reqID, len(body)", reqID, respWrittenLen)
	}

	return nil
}

// processError - log error into header and body
func (s *Service) processError(err error, w http.ResponseWriter, status int, reqID uint64) {

	// логируем в файл с полной трассировкой
	mylog.PrintfErrorMsg(fmt.Sprintf("reqID:['%v'], %+v", reqID, err))

	if w != nil && err != nil {
		// Запишем базовые заголовки
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Request-ID", fmt.Sprintf("%v", reqID))

		if s.cfg.HTTPErrorLogHeader {
			// Заменим в заголовке запрещенные символы на пробел
			// carriage return (CR, ASCII 0xd), line feed (LF, ASCII 0xa), and the zero character (NUL, ASCII 0x0)
			headerReplacer := strings.NewReplacer("\x0a", " ", "\x0d", " ", "\x00", " ")

			// Запишем текст ошибки в заголовок ответа
			if myerr, ok := err.(*myerror.Error); ok {
				// если тип ошибки myerror.Error, то возьмем коды из нее
				w.Header().Set("Err-Code", headerReplacer.Replace(myerr.Code))
				w.Header().Set("Err-Message", headerReplacer.Replace(fmt.Sprintf("%v", myerr)))
				w.Header().Set("Err-Cause-Message", headerReplacer.Replace(myerr.CauseMsg))
				w.Header().Set("Err-Trace", headerReplacer.Replace(myerr.Trace))
			} else {
				w.Header().Set("Err-Message", headerReplacer.Replace(fmt.Sprintf("%+v", err)))
			}
		}

		w.WriteHeader(status) // Запишем статус ответа

		if s.cfg.HTTPErrorLogBody {
			// Запишем ошибку в тело ответа
			fmt.Fprintln(w, fmt.Sprintf("reqID:['%v'], %+v", reqID, err))
		}
	}
}
