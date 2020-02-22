package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	neturl "net/url"
	"time"

	myerror "github.com/romapres2010/httpserver/error"
	httplog "github.com/romapres2010/httpserver/http/httplog"
	mylog "github.com/romapres2010/httpserver/log"
)

// Header represent temporary HTTP header for save
// =====================================================================
type Header map[string]string

// ClientCall represent а parameters for client requests
type ClientCall struct {
	URL                string              // URL
	UserID             string              // UserID для аутентификации
	UserPwd            string              // UserPwd для аутентификации
	CallMethod         string              // HTTP метод для вызова
	CallTimeout        int                 // полный Timeout вызова
	ContentType        string              // тип контента в ответе
	InsecureSkipVerify bool                // игнорировать проверку сертификатов
	ReCallRepeat       int                 // количество попыток вызова при недоступности сервиса - 0 не ограничено
	ReCallWaitTimeout  int                 // Timeout между вызовами при недоступности сервиса
	HTTPLogger         *httplog.HTTPLogger // сервис логирования HTTP
}

// Process - represent client common task in process outcoming HTTP request
// =====================================================================
func (c *ClientCall) Process(ctx context.Context, cnt string, header Header, body []byte) (int, []byte, http.Header, uint64, error) {
	var err error
	var myerr error
	var errM string
	var reqID uint64 // Идентификатор исходящего Request
	var req *http.Request
	var resp *http.Response
	var responseBuf []byte

	{ // входные проверки
		if ctx == nil {
			errM := fmt.Sprintf("Empty context")
			mylog.PrintfErrorStd(errM)
			return 0, nil, nil, 0, myerror.New("6030", errM, cnt, "")
		}
		if c.URL == "" {
			errM := fmt.Sprintf("Empty URL for client call")
			mylog.PrintfErrorStd(errM)
			return 0, nil, nil, 0, myerror.New("6031", errM, cnt, "")
		}
	} // входные проверки

	// Создаем новый запрос c контекстом  для возможности отмены
	// В тело передаем буфер для передачи в составе запроса
	if body != nil {
		req, err = http.NewRequestWithContext(ctx, c.CallMethod, c.URL, bytes.NewReader(body))
	} else {
		req, err = http.NewRequestWithContext(ctx, c.CallMethod, c.URL, nil)
	}
	if err != nil {
		errM := fmt.Sprintf("Failed to create new HTTP request")
		myerr = myerror.WithCause("8012", errM, "http.NewRequestWithContext()", fmt.Sprintf("method='%s', url='%s', len(body)='%v'", c.CallMethod, c.URL, len(body)), "", err.Error())
		mylog.PrintfErrorStd(errM)
		return 0, nil, nil, 0, myerr
	}

	// запишем заголовок в запрос - докускается пустой заголовок
	if header != nil {
		for key, h := range header {
			req.Header.Add(key, h)
		}
	}

	// добавим HTTP Basic Authentication
	if c.UserID != "" && c.UserPwd != "" {
		req.SetBasicAuth(c.UserID, c.UserPwd)
	} else {
		errM = fmt.Sprintf("UserID is null or UserPwd is null. Do call without HTTP Basic Authentication. URL '%s'", c.URL)
		mylog.PrintfInfoStd(errM)
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
			errM = fmt.Sprintf("Context was closed")
			mylog.PrintfErrorStd(errM)
			return 0, nil, nil, 0, myerror.WithCause("6666", errM, cnt, "", "", fmt.Sprintf("Err='%s'", ctx.Err()))
		default:

			// логируем исходящий HTTP запрос
			if c.HTTPLogger != nil {
				reqID, _ = c.HTTPLogger.LogHTTPOutRequest(ctx, req)
			}

			// выполним запрос
			resp, err = client.Do(req)

			// логируем исходящий HTTP ответ
			if c.HTTPLogger != nil {
				c.HTTPLogger.LogHTTPOutResponse(ctx, resp, reqID)
			}

			if err != nil {
				// анализ ошибок
				if httperr, ok := err.(*neturl.Error); ok {
					// если прервано по Timeout или произошло закрытие контекста
					if httperr.Timeout() {
						errM = fmt.Sprintf("Failed to do HTTP request - timeout '%v' exceeded", c.CallTimeout)
						myerr = myerror.WithCause("8013", errM, "client.Do()", fmt.Sprintf("method='%s', url='%s', len(body)='%v', timeout='%v'", c.CallMethod, c.URL, len(body), c.CallTimeout), "", err.Error())
					} else {
						errM = fmt.Sprintf("UNKNOWN ERROR - Failed to do HTTP request")
						myerr = myerror.WithCause("8014", errM, "client.Do()", fmt.Sprintf("method='%s', url='%s', len(body)='%v', timeout='%v'", c.CallMethod, c.URL, len(body), c.CallTimeout), "", err.Error())
					}
				} else {
					errM = fmt.Sprintf("UNKNOWN ERROR - Failed to do HTTP request")
					myerr = myerror.WithCause("8014", errM, "client.Do()", fmt.Sprintf("method='%s', url='%s', len(body)='%v', timeout='%v'", c.CallMethod, c.URL, len(body), c.CallTimeout), "", err.Error())
				}
				mylog.PrintfErrorStd(errM)
				return 0, nil, nil, reqID, myerr
			}

			// считаем тело запроса
			if resp.Body != nil {
				responseBuf, err = ioutil.ReadAll(resp.Body)
				defer resp.Body.Close()
				if err != nil {
					errM = fmt.Sprintf("Failed to read HTTP body")
					myerr = myerror.WithCause("8001", errM, "ioutil.ReadAll()", "", "", err.Error())
					mylog.PrintfErrorStd(errM)
					return resp.StatusCode, nil, resp.Header, reqID, myerr
				}
			}

			mylog.PrintfDebugStd(fmt.Sprintf("Process HTTP call, url='%s':'%s', len(req.Buf)='%v', resp.StatusCode='%s', len(resp.Buf)='%v'", c.CallMethod, c.URL, len(body), resp.Status, len(responseBuf)))

			// частичный анализ статуса ответа
			if resp.StatusCode == http.StatusNotFound {
				// Если превышено количество попыток то на выход
				if c.ReCallRepeat != 0 && tryCount >= c.ReCallRepeat {
					errM = fmt.Sprintf("URL '%s' was not found. Exceeded limit '%v' of attemts to call.", c.URL, c.ReCallRepeat)
					mylog.PrintfInfoStd(errM)
					myerr = myerror.New("8016", errM, "client.Do()", "")
					return 0, nil, nil, reqID, myerr
				}
				// Если URL не доступен - продолжаем в цикле
				errM = fmt.Sprintf("URL '%s' was not found. Wait '%v' Second and try again. Try '%v' times.", c.URL, c.ReCallWaitTimeout, tryCount)
				mylog.PrintfInfoStd(errM)
				// делаем задержку
				time.Sleep(time.Duration(c.ReCallWaitTimeout * int(time.Second)))
				tryCount++
				break // выходим на начало цикла
			} else if resp.StatusCode == http.StatusMethodNotAllowed {
				errM = fmt.Sprintf("URL Method Not Allowed, HTTP Status='%s'", resp.Status)
				myerr = myerror.New("8017", errM, "client.Do()", fmt.Sprintf("method='%s', url='%s', len(resp.Buf)='%v', timeout='%v'", c.CallMethod, c.URL, len(body), c.CallTimeout))
				mylog.PrintfErrorStd(errM)
				return http.StatusMethodNotAllowed, nil, resp.Header, reqID, myerr
			}
			// все успешно
			return resp.StatusCode, responseBuf, resp.Header, reqID, nil
		}
	}
}
