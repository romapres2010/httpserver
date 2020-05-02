package postgres

import (
	"context"

	"github.com/jmoiron/sqlx"
	myctx "github.com/romapres2010/httpserver/ctx"
	myerror "github.com/romapres2010/httpserver/error"
	mylog "github.com/romapres2010/httpserver/log"
	"github.com/romapres2010/httpserver/model"
)

// getEmp return a row for a given id
func (s *Service) getEmp(ctx context.Context, tx *sqlx.Tx, out *model.Emp) (exists bool, myerr error) {
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context
	mylog.PrintfDebugMsg("START: reqID", reqID)

	if out != nil {
		return get(reqID, tx, s.SQLStms["GetEmp"], out, out.Empno)
	}
	return false, myerror.New("4400", "Incorrect call: reqID", reqID).PrintfInfo()
}

// getEmpsByDept return a rows for a given dept
func (s *Service) getEmpsByDept(ctx context.Context, tx *sqlx.Tx, in *model.Dept, out *model.EmpSlice) (myerr error) {
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

	mylog.PrintfDebugMsg("START: reqID", reqID)

	if in != nil && out != nil {
		return selectx(reqID, tx, s.SQLStms["GetEmpsByDept"], out, in.Deptno)
	}
	return myerror.New("4400", "Incorrect call: reqID", reqID).PrintfInfo()
}

// createEmp create new Emp
func (s *Service) createEmp(ctx context.Context, tx *sqlx.Tx, in *model.Emp, out *model.Emp) (myerr error) {
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context
	mylog.PrintfDebugMsg("START: reqID", reqID)

	if in != nil {
		// Создаем объект в рамках транзации
		if myerr = сreate(reqID, tx, s.SQLStms["GetEmp"], s.SQLStms["CreateEmp"], in, out, in.Empno); myerr != nil {
			return myerr
		}

		// считаем обработанную строку - в БД могли быть тригера, которые меняли данные
		if out != nil {
			out.Empno = in.Empno
			if _, myerr := s.getEmp(ctx, tx, out); myerr != nil {
				return myerr
			}
		}
		return nil
	}
	return myerror.New("4400", "Incorrect call: reqID", reqID).PrintfInfo()
}

// updateEmp update the Emp
func (s *Service) updateEmp(ctx context.Context, tx *sqlx.Tx, in *model.Emp, out *model.Emp) (myerr error) {
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context
	mylog.PrintfDebugMsg("START: reqID", reqID)

	if in != nil {
		// Обновляем объект в рамках транзации
		if myerr = update(reqID, tx, s.SQLStms["GetEmp"], s.SQLStms["UpdateEmp"], in, out, in.Empno); myerr != nil {
			return myerr
		}

		// считаем обработанную строку - в БД могли быть тригера, которые меняли данные
		if out != nil {
			out.Empno = in.Empno
			if _, myerr := s.getEmp(ctx, tx, out); myerr != nil {
				return myerr
			}
		}
		return nil
	}
	return myerror.New("4400", "Incorrect call: reqID", reqID).PrintfInfo()
}

// GetEmp return a row for a given id
func (s *Service) GetEmp(ctx context.Context, out *model.Emp) (exists bool, myerr error) {
	return s.getEmp(ctx, nil, out)
}

// GetEmpsByDept return a rows for a given dept
func (s *Service) GetEmpsByDept(ctx context.Context, in *model.Dept, out *model.EmpSlice) (myerr error) {
	return s.getEmpsByDept(ctx, nil, in, out)
}

// CreateEmp create new Emp
func (s *Service) CreateEmp(ctx context.Context, in *model.Emp, out *model.Emp) (myerr error) {
	var tx *sqlx.Tx
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

	// Начинаем новую транзакцию
	if tx, myerr = beginx(reqID, s.pqDb); myerr != nil {
		return myerr
	}

	// Создаем объект в рамках транзации
	if myerr = s.createEmp(ctx, tx, in, out); myerr != nil {
		_ = rollback(reqID, tx)
		return myerr
	}

	// завершаем транзакцию
	return commit(reqID, tx)
}

// UpdateEmp update the Emp
func (s *Service) UpdateEmp(ctx context.Context, in *model.Emp, out *model.Emp) (myerr error) {
	var tx *sqlx.Tx
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

	// Начинаем новую транзакцию
	if tx, myerr = beginx(reqID, s.pqDb); myerr != nil {
		return myerr
	}

	// Создаем объект в рамках транзации
	if myerr = s.updateEmp(ctx, tx, in, out); myerr != nil {
		_ = rollback(reqID, tx)
		return myerr
	}

	// завершаем транзакцию
	return commit(reqID, tx)
}
