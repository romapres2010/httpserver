package httpservice

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"

	"github.com/romapres2010/httpserver/bytespool"
	myctx "github.com/romapres2010/httpserver/ctx"
	myerror "github.com/romapres2010/httpserver/error"
	httplog "github.com/romapres2010/httpserver/httpserver/httplog"
	"github.com/romapres2010/httpserver/json"
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
	ctx      context.Context    // корневой контекст при инициации сервиса
	cancel   context.CancelFunc // функция закрытия глобального контекста
	cfg      *Config            // конфигурационные параметры
	Handlers Handlers           // список обработчиков

	// вложенные сервисы
	logger      *httplog.Logger // сервис логирования HTTP
	jsonService *json.Service   // реализация JSON сервиса
	bytesPool   *bytespool.Pool // represent pooling of []byte
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
	UseBufPool         bool   // use []byte poolling
	BufPooledSize      int    // recomended size of []byte for poolling
	BufPooledMaxSize   int    // max size of []byte for poolling

	// конфигурация вложенных сервисов
	LogCfg       httplog.Config   // конфигурация HTTP логирования
	bytesPoolCfg bytespool.Config // конфигурация bytesPool
}

// New create new HTTP service
func New(ctx context.Context, cfg *Config, jsonService *json.Service) (*Service, *httplog.Logger, error) {
	var err error

	mylog.PrintfInfoMsg("Creating new HTTP service")

	{ // входные проверки
		if jsonService == nil {
			return nil, nil, myerror.New("6030", "Empty JSON service").PrintfInfo()
		}
	} // входные проверки

	service := &Service{
		cfg:         cfg,
		jsonService: jsonService,
	}

	// создаем контекст с отменой
	if ctx == nil {
		service.ctx, service.cancel = context.WithCancel(context.Background())
	} else {
		service.ctx, service.cancel = context.WithCancel(ctx)
	}

	// создаем обработчик для логирования HTTP
	if service.logger, err = httplog.New(service.ctx, &cfg.LogCfg, cfg.HTTPLogFileName); err != nil {
		return nil, nil, err
	}

	// Наполним список обрабочиков
	service.Handlers = map[string]Handler{
		// Типовые обработчики
		"EchoHandler":         Handler{"/echo", service.recoverWrap(service.EchoHandler), "POST"},
		"SinginHandler":       Handler{"/signin", service.recoverWrap(service.SinginHandler), "POST"},
		"JWTRefreshHandler":   Handler{"/refresh", service.recoverWrap(service.JWTRefreshHandler), "POST"},
		"HTTPLogHandler":      Handler{"/httplog", service.recoverWrap(service.HTTPLogHandler), "POST"},
		"HTTPErrorLogHandler": Handler{"/httperrlog", service.recoverWrap(service.HTTPErrorLogHandler), "POST"},
		"LogLevelHandler":     Handler{"/loglevel", service.recoverWrap(service.LogLevelHandler), "POST"},

		// JSON обработчики
		"CreateDeptHandler": Handler{"/depts", service.recoverWrap(service.CreateDeptHandler), "POST"},
		"GetDeptHandler":    Handler{"/depts/{id:[0-9]+}", service.recoverWrap(service.GetDeptHandler), "GET"},
		"UpdateDeptHandler": Handler{"/depts/{id:[0-9]+}", service.recoverWrap(service.UpdateDeptHandler), "PUT"},
	}

	// создаем BytesPool
	if service.cfg.UseBufPool {
		service.cfg.bytesPoolCfg.PooledSize = service.cfg.BufPooledSize
		service.bytesPool = bytespool.New(&service.cfg.bytesPoolCfg)
	}

	mylog.PrintfInfoMsg("HTTP service is created")
	return service, service.logger, nil
}

// GetNextRequestID - запросить номер следующего HTTP запроса
func GetNextRequestID() uint64 {
	return atomic.AddUint64(&requestID, 1)
}

// Shutdown shutting down service
func (s *Service) Shutdown() (myerr error) {
	defer s.cancel() // закрываем контекст

	// Закрываем Logger для корректного закрытия лог файла
	if s.logger != nil {
		myerr = s.logger.Close()
	}

	// Print statistics about bytes pool
	if s.cfg.UseBufPool && s.bytesPool != nil {
		s.bytesPool.PrintBytesPoolStats()
	}

	return
}

// recoverWrap cover handler functions with panic recoverer
func (s *Service) recoverWrap(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// функция восстановления после паники
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

// process - represent server common task in process incoming HTTP request
func (s *Service) process(method string, w http.ResponseWriter, r *http.Request, fn func(ctx context.Context, requestBuf []byte, buf []byte) ([]byte, Header, int, error)) (myerr error) {

	// Получить уникальный номер HTTP запроса
	reqID := GetNextRequestID()

	// для каждого запроса поздаем новый контекст, сохраняем в нем уникальный номер HTTP запроса
	ctx := myctx.NewContextRequestID(s.ctx, reqID)

	// Логируем входящий HTTP запрос
	if s.logger != nil {
		_ = s.logger.LogHTTPInRequest(ctx, r) // При сбое HTTP логирования, делаем системное логирование, но работу не останавливаем
	}

	// Проверим разрешенный метод
	mylog.PrintfDebugMsg("Check allowed HTTP method: reqID, request.Method, method", reqID, r.Method, method)
	if r.Method != method {
		myerr = myerror.New("8000", "HTTP method is not allowed: reqID, request.Method, method", reqID, r.Method, method).PrintfInfo()
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
			myerr := myerror.New("8004", "Header 'Authorization' is not set").PrintfInfo()
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
			myerr := myerror.WithCause("8005", "JWT token does not present in Cookie. You have to authorize first.", err).PrintfInfo()
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
		myerr = myerror.WithCause("8001", "Failed to read HTTP body: reqID", err, reqID).PrintfInfo()
		s.processError(myerr, w, http.StatusInternalServerError, reqID) // расширенное логирование ошибки в контексте HTTP
		return myerr
	}
	mylog.PrintfDebugMsg("Read request body: reqID, len(body)", reqID, len(requestBuf))

	// Выделяем новый буфер из pool, он может использоваться для копирования JSON / XML
	// Если буфер будет недостаточного размера, то он не будет использован
	var buf []byte
	if s.cfg.UseBufPool && s.bytesPool != nil {
		buf = s.bytesPool.GetBuf()
		mylog.PrintfDebugMsg("Got []byte buffer from pool: size", cap(buf))
	}

	// вызываем обработчик
	mylog.PrintfDebugMsg("Calling external function handler: reqID, function", reqID, fn)
	responseBuf, header, status, myerr := fn(ctx, requestBuf, buf)
	if myerr != nil {
		mylog.PrintfErrorInfo(myerr)
		s.processError(myerr, w, status, reqID) // расширенное логирование ошибки в контексте HTTP
		return myerr
	}

	// Если переданного буфера не хватило, то мог быть создан новый буфер. Вернем его в pool
	if responseBuf != nil && buf != nil && s.cfg.UseBufPool && s.bytesPool != nil {
		// Если новый буфер подходит по размерам для хранения в pool
		if cap(responseBuf) >= s.cfg.BufPooledSize && cap(responseBuf) <= s.cfg.BufPooledMaxSize {
			defer s.bytesPool.PutBuf(responseBuf)
		}

		if reflect.ValueOf(buf).Pointer() != reflect.ValueOf(responseBuf).Pointer() {
			mylog.PrintfInfoMsg("[]byte buffer: poolBufSize, responseBufSize", cap(buf), cap(responseBuf))
		}
	}

	// Логируем ответ в файл
	if s.logger != nil {
		_ = s.logger.LogHTTPOutResponse(ctx, header, responseBuf, status) // При сбое HTTP логирования, делаем системное логирование, но работу не останавливаем
	}

	// Записываем заголовок ответа
	mylog.PrintfDebugMsg("Set HTTP response headers: reqID", reqID)
	if header != nil {
		for key, h := range header {
			w.Header().Set(key, h)
		}
	}

	// Устанвливаем HSTS Strict-Transport-Security
	if s.cfg.UseHSTS {
		mylog.PrintfDebugMsg("Set HSTS Strict-Transport-Security header")
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	}

	// Записываем HTTP статус ответа
	mylog.PrintfDebugMsg("Set HTTP response status: reqID, Status", reqID, http.StatusText(status))
	w.WriteHeader(status)

	// Записываем тело ответа
	if responseBuf != nil && len(responseBuf) > 0 {
		mylog.PrintfDebugMsg("Writing HTTP response body: reqID, len(body)", reqID, len(responseBuf))
		respWrittenLen, err := w.Write(responseBuf)
		if err != nil {
			myerr = myerror.WithCause("8002", "Failed to write HTTP repsonse: reqID", err).PrintfInfo()
			s.processError(myerr, w, http.StatusInternalServerError, reqID) // расширенное логирование ошибки в контексте HTTP
			return myerr
		}
		mylog.PrintfDebugMsg("Written HTTP response: reqID, len(body)", reqID, respWrittenLen)
	} else {
		mylog.PrintfDebugMsg("HTTP response body is empty")
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
