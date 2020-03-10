package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	neturl "net/url"
	"time"

	myerror "github.com/romapres2010/httpserver/error"
	httplog "github.com/romapres2010/httpserver/httpserver/httplog"
	"github.com/romapres2010/httpserver/httpserver/httpservice"
	mylog "github.com/romapres2010/httpserver/log"
)

// Header represent temporary HTTP header for save
type Header map[string]string

// Call represent а parameters for client requests
type Call struct {
	URL                string          // URL
	UserID             string          // UserID для аутентификации
	UserPwd            string          // UserPwd для аутентификации
	CallMethod         string          // HTTP метод для вызова
	CallTimeout        int             // полный Timeout вызова
	ContentType        string          // тип контента в ответе
	InsecureSkipVerify bool            // игнорировать проверку сертификатов
	ReCallRepeat       int             // количество попыток вызова при недоступности сервиса - 0 не ограничено
	ReCallWaitTimeout  int             // Timeout между вызовами при недоступности сервиса
	HTTPLogger         *httplog.Logger // сервис логирования HTTP
}

// Process - represent client common task in process outcoming HTTP request
func (c *Call) Process(ctx context.Context, cnt string, header Header, body []byte) (int, []byte, http.Header, uint64, error) {
	var err error
	var myerr error
	var errM string
	var req *http.Request
	var resp *http.Response
	var responseBuf []byte

	// Получить уникальный номер HTTP запроса
	reqID := httpservice.GetNextRequestID() // уникальный ID Request

	{ // входные проверки
		if ctx == nil {
			myerr := myerror.New("6030", "Empty context")
			mylog.PrintfErrorInfo(myerr)
			return 0, nil, nil, reqID, myerr
		}
		if c.URL == "" {
			myerr := myerror.New("6031", "Empty URL for client call")
			mylog.PrintfErrorInfo(myerr)
			return 0, nil, nil, reqID, myerr
		}
	} // входные проверки

	// Создаем новый запрос c контекстом для возможности отмены
	// В тело передаем буфер для передачи в составе запроса
	if body != nil {
		req, err = http.NewRequestWithContext(ctx, c.CallMethod, c.URL, bytes.NewReader(body))
	} else {
		req, err = http.NewRequestWithContext(ctx, c.CallMethod, c.URL, nil)
	}
	if err != nil {
		myerr := myerror.WithCause("8012", "Failed to create new HTTP reques: reqID, Method, URL, len(body)", err, reqID, c.CallMethod, c.URL, len(body))
		mylog.PrintfErrorInfo(myerr)
		return 0, nil, nil, reqID, myerr
	}

	// запишем заголовок в запрос
	if header != nil {
		for key, h := range header {
			req.Header.Add(key, h)
		}
	}

	// добавим HTTP Basic Authentication
	if c.UserID != "" && c.UserPwd != "" {
		req.SetBasicAuth(c.UserID, c.UserPwd)
	} else {
		mylog.PrintfInfoMsg("UserID is null or UserPwd is null. Do call without HTTP Basic Authentication: reqID, Method, URL, len(body)", err, reqID, c.CallMethod, c.URL, len(body))
	}

	// Скопируем дефолтный транспорт
	tr := http.DefaultTransport.(*http.Transport)

	// переопределим проверку невалидных сертификатов при использовании SSL
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: c.InsecureSkipVerify}

	// создадим HTTP клиента с переопределенным транспортом
	client := &http.Client{
		Transport: tr,
		Timeout:   time.Duration(c.CallTimeout * int(time.Second)), // полный таймаут ожидания
	}

	// Если сервис не доступен - то цикл с задержкой
	tryCount := 1
	for {
		// проверим, не получен ли сигнал закрытия контекст - останавливаем обработку
		select {
		case <-ctx.Done():
			myerr := myerror.WithCause("6666", "Context was closed: reqID, Method, URL, len(body)", ctx.Err(), err, reqID, c.CallMethod, c.URL, len(body))
			mylog.PrintfErrorInfo(myerr)
			return 0, nil, nil, reqID, myerr
		default:

			// логируем исходящий HTTP запрос
			if c.HTTPLogger != nil {
				_ = c.HTTPLogger.LogHTTPOutRequest(ctx, req, reqID)
			}

			// выполним запрос
			resp, err = client.Do(req)

			// логируем исходящий HTTP ответ
			if c.HTTPLogger != nil {
				_ = c.HTTPLogger.LogHTTPInResponse(ctx, resp, reqID)
			}

			// обработаем ошибки
			if err != nil {
				if httperr, ok := err.(*neturl.Error); ok {
					// если прервано по Timeout или произошло закрытие контекста
					if httperr.Timeout() {
						myerr = myerror.WithCause("8013", "Failed to do HTTP request - timeout exceeded: reqID, Method, URL, len(body), CallTimeout", err, reqID, c.CallMethod, c.URL, len(body), c.CallTimeout)
					} else {
						myerr = myerror.WithCause("8014", "UNKNOWN neturl.Error - Failed to do HTTP request: reqID, Method, URL, len(body)", err, reqID, c.CallMethod, c.URL, len(body))
					}
				} else {
					myerr = myerror.WithCause("8014", "UNKNOWN ERROR - Failed to do HTTP request: reqID, Method, URL, len(body)", err, reqID, c.CallMethod, c.URL, len(body))
				}
				mylog.PrintfErrorInfo(myerr)
				return 0, nil, nil, reqID, myerr
			}

			// считаем тело ответа
			if resp.Body != nil {
				responseBuf, err = ioutil.ReadAll(resp.Body)
				defer resp.Body.Close()
				if err != nil {
					myerr = myerror.WithCause("8001", "Failed to read HTTP body: reqID, Method, URL", err, reqID, c.CallMethod, c.URL)
					mylog.PrintfErrorInfo(myerr)
					return resp.StatusCode, nil, resp.Header, reqID, myerr
				}
			}

			mylog.PrintfDebugMsg("Process HTTP call: reqID, Method, URL, len(reqBuf), resp.StatusCode, len(resp.Buf)", reqID, c.CallMethod, c.URL, len(body), resp.Status, len(responseBuf))

			// частичный анализ статуса ответа
			if resp.StatusCode == http.StatusNotFound {
				// Если превышено количество попыток то на выход
				if c.ReCallRepeat != 0 && tryCount >= c.ReCallRepeat {
					myerr = myerror.New("8016", "URL was not found. Exceeded limit of attemts to call: reqID, Method, URL, ReCallRepeat", reqID, c.CallMethod, c.URL, c.ReCallRepeat)
					mylog.PrintfErrorInfo(myerr)
					return 0, nil, nil, reqID, myerr
				}
				// Если URL не доступен - продолжаем в цикле
				mylog.PrintfInfoMsg("URL was not found, wait and try again: : reqID, Method, URL, ReCallWaitTimeout, tryCount", reqID, c.CallMethod, c.URL, c.ReCallWaitTimeout, tryCount)
				// делаем задержку
				time.Sleep(time.Duration(c.ReCallWaitTimeout * int(time.Second)))
				tryCount++
				break // выходим на начало цикла
			} else if resp.StatusCode == http.StatusMethodNotAllowed {
				myerr = myerror.New("8017", "URL Method Not Allowed: reqID, Method, URL, HTTP Status", reqID, c.CallMethod, c.URL, resp.Status)
				mylog.PrintfInfoMsg(errM)
				return http.StatusMethodNotAllowed, nil, resp.Header, reqID, myerr
			}
			// все успешно
			return resp.StatusCode, responseBuf, resp.Header, reqID, nil
		}
	}
}
