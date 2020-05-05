package db

import (
	"context"

	myctx "github.com/romapres2010/httpserver/ctx"
	myerror "github.com/romapres2010/httpserver/error"
	mylog "github.com/romapres2010/httpserver/log"
	"github.com/romapres2010/httpserver/model"
	mysql "github.com/romapres2010/httpserver/sqlxx"
)

// getEmp return a row for a given id
func (s *Service) getEmp(ctx context.Context, tx *mysql.Tx, out *model.Emp) (exists bool, myerr error) {
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

	if out != nil {
		mylog.PrintfDebugMsg("START: reqID, Empno", reqID, out.Empno)

		// Запросим основной объект
		if exists, myerr = s.db.Get(reqID, tx, "GetEmp", out, out.Empno); myerr != nil {
			return false, myerr
		}

		// Запросим вложенные объекты
		if exists {
			// ...
		}

		return exists, nil
	}
	return false, myerror.New("4400", "Incorrect call 'out != nil': reqID", reqID).PrintfInfo()
}

// getEmpsByDept return a rows for a given dept
func (s *Service) getEmpsByDept(ctx context.Context, tx *mysql.Tx, in *model.Dept, out *model.EmpSlice) (myerr error) {
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

	if in != nil && out != nil {
		mylog.PrintfDebugMsg("START: reqID, Deptno", reqID, in.Deptno)

		return s.db.Select(reqID, tx, "GetEmpsByDept", out, in.Deptno)
	}
	return myerror.New("4400", "Incorrect call 'in != nil && out != nil': reqID", reqID).PrintfInfo()
}

// createEmp create new Emp
func (s *Service) createEmp(ctx context.Context, tx *mysql.Tx, in *model.Emp, out *model.Emp) (myerr error) {
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

	if in != nil && tx != nil {
		mylog.PrintfDebugMsg("START: reqID, Empno", reqID, in.Empno)

		newEmp := model.GetEmp()   // Извлечем из pool структуру для нового экземпляра в БД
		defer model.PutEmp(newEmp) // Вернем структуру в pool

		{ // Проверим, существует ли строка по натуральному уникальному ключу UK
			mylog.PrintfDebugMsg("Check if row already exists: reqID, Empno", reqID, in.Empno)
			exists, myerr := s.db.Get(reqID, tx, "EmpExists", new(int), in.Empno)
			if myerr != nil {
				return myerr
			}
			if exists {
				return myerror.New("4004", "Error create - row already exists: reqID, Empno", reqID, in.Empno).PrintfInfo()
			}
		} // Проверим, существует ли строка по натуральному уникальному ключу UK

		{ // Выполняем вставку и получим значение сурогатного PK
			rows, myerr := s.db.Exec(reqID, tx, "CreateEmp", in)
			if myerr != nil {
				return myerr
			}
			// проверим количество обработанных строк
			if rows != 1 {
				return myerror.New("4004", "Error create: reqID, Empno, rows", reqID, in.Empno, rows).PrintfInfo()
			}

			// считаем созданный объект - в БД могли быть тригера, которые меняли данные
			// запрос делаем по UK, так как сурогатный PK мы еще не знаем
			exists, myerr := s.db.Get(reqID, tx, "GetEmpUK", newEmp, in.Empno)
			if myerr != nil {
				return myerr
			}
			if !exists {
				return myerror.New("4004", "Row does not exists after creating: reqID, Empno", reqID, in.Empno).PrintfInfo()
			}
		} // Выполняем вставку и получим значение сурогатного PK

		{ // Обработаем вложенные объекты в рамках текущей транзации
			// ...
		} // Обработаем вложенные объекты в рамках текущей транзации

		// считаем созданный объект из БД
		if out != nil {
			out.Empno = newEmp.Empno // столбцы первичного ключа PK
			exists, myerr := s.getEmp(ctx, tx, out)
			if myerr != nil {
				return myerr
			}
			// Проверка для отладки табличного API
			if !exists {
				return myerror.New("4004", "Row does not exists after creating: reqID, PK", reqID, in.Empno).PrintfInfo()
			}
		}
		return nil
	}
	return myerror.New("4400", "Incorrect call 'in != nil && tx != nil': reqID", reqID).PrintfInfo()
}

// updateEmp update the Emp
func (s *Service) updateEmp(ctx context.Context, tx *mysql.Tx, in *model.Emp, out *model.Emp) (exists bool, myerr error) {
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

	if in != nil && tx != nil {
		mylog.PrintfDebugMsg("START: reqID, Empno", reqID, in.Empno)

		oldEmp := model.GetEmp()   // Извлечем из pool структуру для старого экземпляра в БД
		defer model.PutEmp(oldEmp) // Вернем структуру в pool

		newEmp := model.GetEmp()   // Извлечем из pool структуру для нового экземпляра в БД
		defer model.PutEmp(newEmp) // Вернем структуру в pool

		{ // Считаем состояние объекта до обновления и проверим его существование
			mylog.PrintfDebugMsg("Get row and check if it exists: reqID, PK", reqID, in.Empno)
			oldEmp.Empno = in.Empno // столбцы первичного ключа PK
			if exists, myerr = s.getEmp(ctx, tx, oldEmp); myerr != nil {
				return false, myerr
			}
			if !exists {
				mylog.PrintfDebugMsg("Row does not exists before updating: reqID, PK", reqID, in.Empno)
				return false, nil
			}
		} // Считаем состояние объекта до обновления и проверим его существование

		{ // выполняем проверки / действия на основании старых и новых значений атрибутов
			// ...
		} // выполняем проверки / действия на основании старых и новых значений атрибутов

		{ // Выполняем обновление
			rows, myerr := s.db.Exec(reqID, tx, "UpdateEmp", in)
			if myerr != nil {
				return false, myerr
			}
			// проверим количество обработанных строк
			if rows != 1 {
				return false, myerror.New("4004", "Error update: reqID, Empno, rows", reqID, in.Empno, rows).PrintfInfo()
			}

			// считаем объект по сурогатному PK
			exists, myerr := s.db.Get(reqID, tx, "GetEmp", newEmp, in.Empno)
			if myerr != nil {
				return false, myerr
			}
			if !exists {
				return false, myerror.New("4004", "Row does not exists after creating: reqID, PK", reqID, in.Deptno).PrintfInfo()
			}
		} // Выполняем обновление

		{ // Обработаем вложенные объекты в рамках текущей транзации
			// ...
		} // Обработаем вложенные объекты в рамках текущей транзации

		// считаем обновленный объект из БД
		if out != nil {
			out.Empno = in.Empno // столбцы первичного ключа PK
			if exists, myerr = s.getEmp(ctx, tx, out); myerr != nil {
				return false, myerr
			}
			// Проверка для отладки табличного API
			if !exists {
				return false, myerror.New("4004", "Row does not exists after updating: reqID, PK", reqID, in.Empno).PrintfInfo()
			}
		}
		return true, nil
	}
	return false, myerror.New("4400", "Incorrect call 'in != nil && tx != nil': reqID", reqID).PrintfInfo()
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
	var tx *mysql.Tx
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

	// Начинаем новую транзакцию
	if tx, myerr = s.db.Beginx(reqID); myerr != nil {
		return myerr
	}

	// Создаем объект в рамках транзации
	if myerr = s.createEmp(ctx, tx, in, out); myerr != nil {
		_ = s.db.Rollback(reqID, tx)
		return myerr
	}

	// завершаем транзакцию
	return s.db.Commit(reqID, tx)
}

// UpdateEmp update the Emp
func (s *Service) UpdateEmp(ctx context.Context, in *model.Emp, out *model.Emp) (exists bool, myerr error) {
	var tx *mysql.Tx
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

	// Начинаем новую транзакцию
	if tx, myerr = s.db.Beginx(reqID); myerr != nil {
		return false, myerr
	}

	// Создаем объект в рамках транзации
	if exists, myerr = s.updateEmp(ctx, tx, in, out); myerr != nil {
		_ = s.db.Rollback(reqID, tx)
		return false, myerr
	}

	// Если объект или один из вложенных подобъектов не был найден при обновлении, то откат
	if !exists {
		_ = s.db.Rollback(reqID, tx)
		return false, nil
	}

	// завершаем транзакцию
	if myerr = s.db.Commit(reqID, tx); myerr != nil {
		return false, myerr
	}
	return exists, nil
}
