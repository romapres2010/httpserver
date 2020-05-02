package json

import (
	"context"

	jwriter "github.com/mailru/easyjson/jwriter"
	myctx "github.com/romapres2010/httpserver/ctx"
	myerror "github.com/romapres2010/httpserver/error"
	mylog "github.com/romapres2010/httpserver/log"
	model "github.com/romapres2010/httpserver/model"
)

// GetDept return a JSON for a given id
func (s *Service) GetDept(ctx context.Context, id int, buf []byte) ([]byte, error) {
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context
	mylog.PrintfDebugMsg("START: reqID", reqID)

	vOut := model.GetDept()         // Извлечем из pool новую структуру
	defer model.PutDept(vOut, true) // возвращаем в pool струкуру со всеми вложенными объектами
	vOut.Deptno = id                // параметры для запроса передаются в структуре

	// вызываем сервис обработки
	exists, err := s.deptService.GetDept(ctx, vOut)
	if err != nil {
		return nil, err
	}

	// сформируем json
	if exists {
		mylog.PrintfDebugMsg("Marshal with EasyJSON: reqID", reqID)
		w := jwriter.Writer{}    // подготовим EasyJSON Writer
		vOut.MarshalEasyJSON(&w) // сформируем JSON во внутренний буфер EasyJSON Writer
		if w.Error != nil {
			return nil, myerror.WithCause("6001", "Error Marshal: reqID", w.Error, reqID).PrintfInfo()
		}
		// Скопируем из внутреннего буфера EasyJSON Writer во внешний буфер
		// Если размер внещнего буфера будет мал - то он исопльзован не будет
		mylog.PrintfDebugMsg("SUCCESS: reqID", reqID)
		return w.Buffer.BuildBytes(buf), nil
	}

	mylog.PrintfDebugMsg("SUCCESS: reqID", reqID)
	return nil, nil // возврат пустого буфера - признак, что объекта не найдено
}

// CreateDept create dept and return a JSON
func (s *Service) CreateDept(ctx context.Context, inBuf []byte, buf []byte) ([]byte, error) {
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context
	mylog.PrintfDebugMsg("START: reqID", reqID)

	vIn := model.GetDept()         // Извлечем из pool новую структуру
	defer model.PutDept(vIn, true) // возвращаем в pool струкуру

	vOut := model.GetDept()         // Извлечем из pool новую структуру
	defer model.PutDept(vOut, true) // возвращаем в pool струкуру

	// Парсим JSON в структуру
	//err := json.Unmarshal(inBuf, vIn)
	mylog.PrintfDebugMsg("Unmarshal with EasyJSON: reqID", reqID)
	if err := vIn.UnmarshalJSON(inBuf); err != nil {
		return nil, myerror.WithCause("6001", "Error Unmarshal: reqID, buf", err, reqID, string(inBuf)).PrintfInfo()
	}

	// вызываем сервис обработки
	if err := s.deptService.CreateDept(ctx, vIn, vOut); err != nil {
		return nil, err
	}

	{ // сформируем json
		mylog.PrintfDebugMsg("Marshal with EasyJSON: reqID", reqID)
		w := jwriter.Writer{}    // подготовим EasyJSON Writer
		vOut.MarshalEasyJSON(&w) // сформируем JSON во внутренний буфер EasyJSON Writer
		if w.Error != nil {
			return nil, myerror.WithCause("6001", "Error Marshal: reqID", w.Error, reqID).PrintfInfo()
		}
		// Скопируем из внутреннего буфера EasyJSON Writer во внешний буфер
		// Если размер внещнего буфера будет мал - то он исопльзован не будет
		mylog.PrintfDebugMsg("SUCCESS: reqID", reqID)
		return w.Buffer.BuildBytes(buf), nil
	} // сформируем json
}

// UpdateDept update dept and return a JSON
func (s *Service) UpdateDept(ctx context.Context, id int, inBuf []byte, buf []byte) ([]byte, error) {
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context
	mylog.PrintfDebugMsg("START: reqID", reqID)

	vIn := model.GetDept()         // Извлечем из pool новую структуру
	defer model.PutDept(vIn, true) // возвращаем в pool струкуру

	vOut := model.GetDept()         // Извлечем из pool новую структуру
	defer model.PutDept(vOut, true) // возвращаем в pool струкуру

	// Парсим JSON в структуру
	//err := json.Unmarshal(inBuf, vIn)
	mylog.PrintfDebugMsg("Unmarshal with EasyJSON: reqID", reqID)
	if err := vIn.UnmarshalJSON(inBuf); err != nil {
		return nil, myerror.WithCause("6001", "Error Unmarshal: reqID, buf", err, reqID, string(inBuf)).PrintfInfo()
	}

	// проверим ID объекта
	if id != vIn.Deptno {
		return nil, myerror.New("6001", "Resource ID does not corespond to JSON: resource.id, json.Deptno", id, vIn.Deptno).PrintfInfo()
	}

	// вызываем сервис обработки
	if err := s.deptService.UpdateDept(ctx, vIn, vOut); err != nil {
		return nil, err
	}

	{ // сформируем json
		mylog.PrintfDebugMsg("Marshal with EasyJSON: reqID", reqID)
		w := jwriter.Writer{}    // подготовим EasyJSON Writer
		vOut.MarshalEasyJSON(&w) // сформируем JSON во внутренний буфер EasyJSON Writer
		if w.Error != nil {
			return nil, myerror.WithCause("6001", "Error Marshal: reqID", w.Error, reqID).PrintfInfo()
		}
		// Скопируем из внутреннего буфера EasyJSON Writer во внешний буфер
		// Если размер внещнего буфера будет мал - то он исопльзован не будет
		mylog.PrintfDebugMsg("SUCCESS: reqID", reqID)
		return w.Buffer.BuildBytes(buf), nil
	} // сформируем json
}
