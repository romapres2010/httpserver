package httpservice

import (
	"fmt"
	"net/http"

	myerror "github.com/romapres2010/httpserver/error"
	mylog "github.com/romapres2010/httpserver/log"
)

// EchoHandler handle echo page with request header and body
func (s *Service) EchoHandler(w http.ResponseWriter, r *http.Request) {
	mylog.PrintfDebugMsg("START   ==================================================================================")

	// Запускаем обработчик, возврат ошибки игнорируем
	_ = s.Process("POST", w, r, func(requestBuf []byte, reqID uint64) ([]byte, Header, int, error) {
		mylog.PrintfDebugMsg("START: reqID", reqID)

		// формируем ответ
		header := Header{}
		header["Errcode"] = "0"
		header["RequestID"] = fmt.Sprintf("%v", reqID)

		// Считаем параметры из заголовка сообщения
		for key := range r.Header {
			header[key] = r.Header.Get(key)
		}

		// time.Sleep(time.Second * 120)

		myerr := myerror.New("8004", "Error message", "arg1", "arg2", "arg3")
		mylog.PrintfErrorInfo(myerr)
		return nil, nil, http.StatusInternalServerError, myerr

		mylog.PrintfDebugMsg("SUCCESS", reqID)
		return requestBuf, header, http.StatusOK, nil
	})

	mylog.PrintfDebugMsg("SUCCESS ==================================================================================")
}
