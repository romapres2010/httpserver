package json

import (
	"context"

	jwriter "github.com/mailru/easyjson/jwriter"
	myctx "github.com/romapres2010/httpserver/ctx"
	myerror "github.com/romapres2010/httpserver/error"
	mylog "github.com/romapres2010/httpserver/log"
	model "github.com/romapres2010/httpserver/model"
)

func deptMarshal(reqID uint64, v *model.Dept, buf []byte) (outBuf []byte, myerr error) {
	mylog.PrintfDebugMsgDepth("Marshal with EasyJSON: reqID", 1, reqID)

	w := jwriter.Writer{} // подготовим EasyJSON Writer
	v.MarshalEasyJSON(&w) // сформируем JSON во внутренний буфер EasyJSON Writer

	if w.Error != nil {
		return nil, myerror.WithCause("6001", "Error Marshal: reqID", w.Error, reqID).PrintfInfo(1)
	}

	// Скопируем из внутреннего буфера EasyJSON Writer во внешний буфер
	// Если размер внешнего буфера будет мал - то он использован не будет
	mylog.PrintfDebugMsgDepth("SUCCESS: reqID", 1, reqID)
	return w.Buffer.BuildBytes(buf), nil
}

// GetDept return a JSON for a given PK
func (s *Service) GetDept(ctx context.Context, id int, buf []byte) (outBuf []byte, myerr error) {
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context
	mylog.PrintfDebugMsg("START: reqID", reqID)

	vOut := model.GetDept()         // Извлечем из pool новую структуру
	defer model.PutDept(vOut, true) // возвращаем в pool струкуру со всеми вложенными объектами
	vOut.Deptno = id                // параметры для запроса передаются в структуре

	// вызываем сервис обработки
	exists, myerr := s.deptService.GetDept(ctx, vOut)
	if myerr != nil {
		return nil, myerr
	}

	// сформируем json
	if exists {
		return deptMarshal(reqID, vOut, buf)
	}

	return nil, nil // возврат пустого буфера - признак, что объекта не найдено
}

// CreateDept create dept and return a JSON
func (s *Service) CreateDept(ctx context.Context, inBuf []byte, buf []byte) (id int, outBuf []byte, myerr error) {
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context
	mylog.PrintfDebugMsg("START: reqID", reqID)

	vIn := model.GetDept()         // Извлечем из pool новую структуру
	defer model.PutDept(vIn, true) // возвращаем в pool струкуру

	vOut := model.GetDept()         // Извлечем из pool новую структуру
	defer model.PutDept(vOut, true) // возвращаем в pool струкуру

	// Парсим JSON в структуру
	mylog.PrintfDebugMsg("Unmarshal with EasyJSON: reqID", reqID)
	if err := vIn.UnmarshalJSON(inBuf); err != nil {
		return 0, nil, myerror.WithCause("6001", "Error Unmarshal: reqID, buf", err, reqID, string(inBuf)).PrintfInfo()
	}

	// вызываем сервис обработки
	if myerr = s.deptService.CreateDept(ctx, vIn, vOut); myerr != nil {
		return 0, nil, myerr
	}

	// сформируем json
	if outBuf, myerr = deptMarshal(reqID, vOut, buf); myerr != nil {
		return 0, nil, myerr
	}

	return vOut.Deptno, outBuf, myerr
}

// UpdateDept update dept and return a JSON
func (s *Service) UpdateDept(ctx context.Context, id int, inBuf []byte, buf []byte) (outBuf []byte, myerr error) {
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context
	mylog.PrintfDebugMsg("START: reqID", reqID)

	vIn := model.GetDept()         // Извлечем из pool новую структуру
	defer model.PutDept(vIn, true) // возвращаем в pool струкуру

	vOut := model.GetDept()         // Извлечем из pool новую структуру
	defer model.PutDept(vOut, true) // возвращаем в pool струкуру

	// Парсим JSON в структуру
	mylog.PrintfDebugMsg("Unmarshal with EasyJSON: reqID", reqID)
	if err := vIn.UnmarshalJSON(inBuf); err != nil {
		return nil, myerror.WithCause("6001", "Error Unmarshal: reqID, buf", err, reqID, string(inBuf)).PrintfInfo()
	}

	// проверим ID объекта
	if id != vIn.Deptno {
		return nil, myerror.New("6001", "Resource ID does not corespond to JSON: resource.id, json.Deptno", id, vIn.Deptno).PrintfInfo()
	}

	// вызываем сервис обработки
	exists, myerr := s.deptService.UpdateDept(ctx, vIn, vOut)
	if myerr != nil {
		return nil, myerr
	}

	// сформируем json
	if exists {
		return deptMarshal(reqID, vOut, buf)
	}

	return nil, nil // возврат пустого буфера - признак, что объекта не найдено
}
