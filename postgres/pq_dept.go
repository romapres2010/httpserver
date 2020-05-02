package postgres

import (
	"context"

	"github.com/jmoiron/sqlx"
	myctx "github.com/romapres2010/httpserver/ctx"
	myerror "github.com/romapres2010/httpserver/error"
	mylog "github.com/romapres2010/httpserver/log"
	model "github.com/romapres2010/httpserver/model"
)

// getDept return a Dept with a given id
func (s *Service) getDept(ctx context.Context, tx *sqlx.Tx, out *model.Dept) (exists bool, myerr error) {
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context
	mylog.PrintfDebugMsg("START: reqID", reqID)

	if out != nil {
		{ // Считаем вложенные объекты
			vOutEmps := model.GetEmpSlice() // Извлечем из pool срез для вложенных объектов
			if myerr = s.getEmpsByDept(ctx, tx, out, &vOutEmps); myerr != nil {
				return false, myerr
			}
			out.Emps = vOutEmps // Встроим срез в основной объект
		} // Считаем вложенные объекты

		// Считаем основной объект
		return get(reqID, tx, s.SQLStms["GetDept"], out, out.Deptno)
	}
	return false, myerror.New("4400", "Incorrect call: reqID", reqID).PrintfInfo()
}

// getDeptsPK return a PK for all Dept
func (s *Service) getDeptsPK(ctx context.Context, tx *sqlx.Tx, out *model.DeptPKs) (myerr error) {
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context
	mylog.PrintfDebugMsg("START: reqID", reqID)

	if out != nil {
		return selectx(reqID, tx, s.SQLStms["GetDeptsPK"], out)
	}
	return myerror.New("4400", "Incorrect call: reqID", reqID).PrintfInfo()
}

// createDept create new Dept
func (s *Service) createDept(ctx context.Context, tx *sqlx.Tx, in *model.Dept, out *model.Dept) (myerr error) {
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context
	mylog.PrintfDebugMsg("START: reqID", reqID)

	if in != nil && tx != nil {
		// Создаем объект в рамках транзации
		if myerr = сreate(reqID, tx, s.SQLStms["GetDept"], s.SQLStms["CreateDept"], in, out, in.Deptno); myerr != nil {
			return myerr
		}

		{ // Обработаем вложенные объекты в рамках текущей транзации
			if in.Emps != nil {
				for _, v := range in.Emps {
					// создаем вложенные объекты без проверки существования строк
					if myerr = s.createEmp(ctx, tx, v, nil); myerr != nil {
						return myerr
					}
				}
			}
		} // Обработаем вложенные объекты в рамках текущей транзации

		// считаем объект - в БД могли быть тригера, которые меняли данные
		if out != nil {
			out.Deptno = in.Deptno
			if _, myerr := s.getDept(ctx, tx, out); myerr != nil {
				return myerr
			}
		}
		return nil
	}
	return myerror.New("4400", "Incorrect call: reqID", reqID).PrintfInfo()
}

// updateDept update the Dept
func (s *Service) updateDept(ctx context.Context, tx *sqlx.Tx, in *model.Dept, out *model.Dept) (myerr error) {
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context
	mylog.PrintfDebugMsg("START: reqID", reqID)

	if in != nil && tx != nil {
		// Обновляем объект в рамках транзации
		if myerr = update(reqID, tx, s.SQLStms["GetDept"], s.SQLStms["UpdateDept"], in, out, in.Deptno); myerr != nil {
			return myerr
		}

		{ // Обработаем вложенные объекты в рамках текущей транзации
			if in.Emps != nil {
				for _, v := range in.Emps {
					// создаем вложенные объекты без проверки существования строк
					if myerr = s.updateEmp(ctx, tx, v, nil); myerr != nil {
						return myerr
					}
				}
			}
		} // Обработаем вложенные объекты в рамках текущей транзации

		// считаем объект - в БД могли быть тригера, которые меняли данные
		if out != nil {
			out.Deptno = in.Deptno
			if _, myerr := s.getDept(ctx, tx, out); myerr != nil {
				return myerr
			}
		}
		return nil
	}
	return myerror.New("4400", "Incorrect call: reqID", reqID).PrintfInfo()
}

// GetDept return a Dept with a given id
func (s *Service) GetDept(ctx context.Context, out *model.Dept) (exists bool, myerr error) {
	return s.getDept(ctx, nil, out)
}

// GetDeptsPK return a PK for all Dept
func (s *Service) GetDeptsPK(ctx context.Context, out *model.DeptPKs) (myerr error) {
	return s.getDeptsPK(ctx, nil, out)
}

// CreateDept create new Dept
func (s *Service) CreateDept(ctx context.Context, in *model.Dept, out *model.Dept) (myerr error) {
	var tx *sqlx.Tx
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

	// Начинаем новую транзакцию
	if tx, myerr = beginx(reqID, s.pqDb); myerr != nil {
		return myerr
	}

	// Создаем объект в рамках транзации
	if myerr = s.createDept(ctx, tx, in, out); myerr != nil {
		_ = rollback(reqID, tx)
		return myerr
	}

	// завершаем транзакцию
	return commit(reqID, tx)
}

// UpdateDept update Dept
func (s *Service) UpdateDept(ctx context.Context, in *model.Dept, out *model.Dept) (myerr error) {
	var tx *sqlx.Tx
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

	// Начинаем новую транзакцию
	if tx, myerr = beginx(reqID, s.pqDb); myerr != nil {
		return myerr
	}

	// Создаем объект в рамках транзации
	if myerr = s.updateDept(ctx, tx, in, out); myerr != nil {
		_ = rollback(reqID, tx)
		return myerr
	}

	// завершаем транзакцию
	return commit(reqID, tx)
}
