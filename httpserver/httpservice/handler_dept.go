package httpservice

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	myctx "github.com/romapres2010/httpserver/ctx"
	myerror "github.com/romapres2010/httpserver/error"
	mylog "github.com/romapres2010/httpserver/log"
)

// GetDeptHandler handle JSON for GetDept
func (s *Service) GetDeptHandler(w http.ResponseWriter, r *http.Request) {
	mylog.PrintfDebugMsg("START   ==================================================================================")

	// Запускаем типовой process, возврат ошибки игнорируем
	_ = s.process("GET", w, r, func(ctx context.Context, requestBuf []byte, buf []byte) ([]byte, Header, int, error) {
		reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

		mylog.PrintfDebugMsg("START: reqID", reqID)

		// Считаем PK из URL запроса и проверим на число
		vars := mux.Vars(r)
		idStr := vars["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return nil, nil, http.StatusBadRequest, myerror.WithCause("8001", "Failed to process parameter 'id' invalid number: reqID, id", err, reqID, idStr).PrintfInfo()
		}

		// вызываем JSON сервис, передаем ему буфер для копирования
		responseBuf, err := s.jsonService.GetDept(ctx, id, buf)
		if err != nil {
			return nil, nil, http.StatusInternalServerError, err
		}

		// Если данные не найдены
		if responseBuf == nil {
			return nil, nil, http.StatusNotFound, nil
		}

		// формируем ответ
		header := Header{}
		header["Content-Type"] = "application/json; charset=utf-8"
		header["Errcode"] = "0"
		header["RequestID"] = fmt.Sprintf("%v", reqID)

		mylog.PrintfDebugMsg("SUCCESS: reqID", reqID)
		return responseBuf, header, http.StatusOK, nil
	})

	mylog.PrintfDebugMsg("SUCCESS ==================================================================================")
}

// CreateDeptHandler handle JSON for CreateDept
func (s *Service) CreateDeptHandler(w http.ResponseWriter, r *http.Request) {
	mylog.PrintfDebugMsg("START   ==================================================================================")

	// Запускаем типовой process, возврат ошибки игнорируем
	_ = s.process("POST", w, r, func(ctx context.Context, requestBuf []byte, buf []byte) ([]byte, Header, int, error) {
		reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

		mylog.PrintfDebugMsg("START: reqID", reqID)

		// вызываем JSON сервис
		id, responseBuf, err := s.jsonService.CreateDept(ctx, requestBuf, buf)
		if err != nil {
			return nil, nil, http.StatusInternalServerError, err
		}

		// формируем ответ
		header := Header{}
		header["Content-Type"] = "application/json; charset=utf-8"
		header["Errcode"] = "0"
		header["Id"] = fmt.Sprintf("%v", id)
		header["RequestID"] = fmt.Sprintf("%v", reqID)

		mylog.PrintfDebugMsg("SUCCESS: reqID", reqID)
		return responseBuf, header, http.StatusOK, nil
	})

	mylog.PrintfDebugMsg("SUCCESS ==================================================================================")
}

// UpdateDeptHandler handle JSON for UpdateDept
func (s *Service) UpdateDeptHandler(w http.ResponseWriter, r *http.Request) {
	mylog.PrintfDebugMsg("START   ==================================================================================")

	// Запускаем типовой process, возврат ошибки игнорируем
	_ = s.process("PUT", w, r, func(ctx context.Context, requestBuf []byte, buf []byte) ([]byte, Header, int, error) {
		reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

		mylog.PrintfDebugMsg("START: reqID", reqID)

		// Считаем параметры и проверим на число
		vars := mux.Vars(r)
		idStr := vars["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return nil, nil, http.StatusBadRequest, myerror.WithCause("8001", "Failed to process parameter 'id' invalid number: reqID, id", err, reqID, idStr).PrintfInfo()
		}

		// вызываем JSON сервис
		responseBuf, err := s.jsonService.UpdateDept(ctx, id, requestBuf, buf)
		if err != nil {
			return nil, nil, http.StatusInternalServerError, err
		}

		// Если данные не найдены
		if responseBuf == nil {
			return nil, nil, http.StatusNotFound, nil
		}

		// формируем ответ
		header := Header{}
		header["Content-Type"] = "application/json; charset=utf-8"
		header["Errcode"] = "0"
		header["RequestID"] = fmt.Sprintf("%v", reqID)

		mylog.PrintfDebugMsg("SUCCESS: reqID", reqID)
		return responseBuf, header, http.StatusOK, nil
	})

	mylog.PrintfDebugMsg("SUCCESS ==================================================================================")
}
