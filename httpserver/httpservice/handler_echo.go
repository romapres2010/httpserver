package httpservice

import (
	"fmt"
	"net/http"

	mylog "github.com/romapres2010/httpserver/log"
)

// EchoHandler handle echo page with request header and body
// =====================================================================
func (s *Service) EchoHandler(w http.ResponseWriter, r *http.Request) {
	mylog.PrintfDebugStd("START   ==================================================================================")

	// Запускаем обработчик
	s.Process("GET", w, r, func(requestBuf []byte, reqID uint64) ([]byte, Header, int, error) {
		mylog.PrintfDebugStd("START", reqID)

		// формируем ответ
		header := make(map[string]string)
		header["Errcode"] = "0"
		header["RequestID"] = fmt.Sprintf("%v", reqID)

		// Считаем параметры из заголовка сообщения
		for key := range r.Header {
			header[key] = r.Header.Get(key)
		}

		mylog.PrintfDebugStd("SUCCESS", reqID)
		return requestBuf, header, http.StatusOK, nil
	})

	mylog.PrintfDebugStd("SUCCESS ==================================================================================")
}
