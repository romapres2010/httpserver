package db

import (
	"context"
	"database/sql"

	myctx "github.com/romapres2010/httpserver/ctx"
	myerror "github.com/romapres2010/httpserver/error"
	mylog "github.com/romapres2010/httpserver/log"
	model "github.com/romapres2010/httpserver/model"
	mysql "github.com/romapres2010/httpserver/sqlxx"
	"gopkg.in/guregu/null.v4"
)

// getDept return a Dept with a given id
func (s *Service) getDept(ctx context.Context, tx *mysql.Tx, out *model.Dept) (exists bool, myerr error) {
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

	if out != nil {
		mylog.PrintfDebugMsg("START: reqID, Deptno", reqID, out.Deptno)

		// Запросим основной объект
		if exists, myerr = s.db.Get(reqID, tx, "GetDept", out, out.Deptno); myerr != nil {
			return false, myerr
		}

		// Запросим вложенные объекты
		if exists {
			outEmps := model.GetEmpSlice() // Извлечем из pool срез для вложенных объектов
			if myerr = s.getEmpsByDept(ctx, tx, out, &outEmps); myerr != nil {
				return false, myerr
			}
			out.Emps = outEmps // Встроим срез в основной объект
		}

		return exists, nil
	}
	return false, myerror.New("4400", "Incorrect call 'out != nil': reqID", reqID).PrintfInfo()
}

// getDeptsPK return a PK for all Dept
func (s *Service) getDeptsPK(ctx context.Context, tx *mysql.Tx, out *model.DeptPKs) (myerr error) {
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

	if out != nil {
		mylog.PrintfDebugMsg("START: reqID", reqID)

		return s.db.Select(reqID, tx, "GetDeptsPK", out)
	}
	return myerror.New("4400", "Incorrect call 'out != nil': reqID", reqID).PrintfInfo()
}

// createDept create new Dept
func (s *Service) createDept(ctx context.Context, tx *mysql.Tx, in *model.Dept, out *model.Dept) (myerr error) {
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

	if in != nil && out != nil && tx != nil {
		mylog.PrintfDebugMsg("START: reqID, Deptno", reqID, in.Deptno)

		newDept := model.GetDept()         // Извлечем из pool структуру для нового экземпляра в БД
		defer model.PutDept(newDept, true) // Вернем структуру в pool

		{ // Проверим, существует ли строка по натуральному уникальному ключу UK
			mylog.PrintfDebugMsg("Check if row already exists: reqID, Deptno", reqID, in.Deptno)
			var foo int
			exists, myerr := s.db.Get(reqID, tx, "DeptExists", &foo, in.Deptno)
			if myerr != nil {
				return myerr
			}
			if exists {
				return myerror.New("4004", "Error create - row already exists: reqID, Deptno", reqID, in.Deptno).PrintfInfo()
			}
		} // Проверим, существует ли строка по натуральному уникальному ключу UK

		{ // Выполняем вставку и получим значение сурогатного PK
			rows, myerr := s.db.Exec(reqID, tx, "CreateDept", in)
			if myerr != nil {
				return myerr
			}
			// проверим количество обработанных строк
			if rows != 1 {
				return myerror.New("4004", "Error create: reqID, Deptno, rows", reqID, in.Deptno, rows).PrintfInfo()
			}

			// считаем созданный объект - в БД могли быть тригера, которые меняли данные
			// запрос делаем по UK, так как сурогатный PK мы еще не знаем
			exists, myerr := s.db.Get(reqID, tx, "GetDeptUK", newDept, in.Deptno)
			if myerr != nil {
				return myerr
			}
			if !exists {
				return myerror.New("4004", "Row does not exists after creating: reqID, Deptno", reqID, in.Deptno).PrintfInfo()
			}
		} // Выполняем вставку и получим значение сурогатного PK

		{ // Обработаем вложенные объекты в рамках текущей транзации
			if in.Emps != nil {
				for _, newEmp := range in.Emps {
					// Копируем сурогатный PK во внешний ключ вложенного объекта
					newEmp.Deptno = null.Int{sql.NullInt64{int64(newDept.Deptno), true}}

					// создаем вложенные объекты
					if myerr = s.createEmp(ctx, tx, newEmp, nil); myerr != nil {
						return myerr
					}
				}
			}
		} // Обработаем вложенные объекты в рамках текущей транзации

		// считаем обновленный объект из БД
		if out != nil {
			out.Deptno = newDept.Deptno // столбцы первичного ключа PK
			exists, myerr := s.getDept(ctx, tx, out)
			if myerr != nil {
				return myerr
			}
			// Проверка для отладки табличного API
			if !exists {
				return myerror.New("4004", "Row does not exists after updating: reqID, PK", reqID, in.Deptno).PrintfInfo()
			}
		}
		return nil
	}
	return myerror.New("4400", "Incorrect call 'in != nil && tx != nil': reqID", reqID).PrintfInfo()
}

// updateDept update the Dept
func (s *Service) updateDept(ctx context.Context, tx *mysql.Tx, in *model.Dept, out *model.Dept) (exists bool, myerr error) {
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

	if in != nil && tx != nil {
		mylog.PrintfDebugMsg("START: reqID, Deptno", reqID, in.Deptno)

		oldDept := model.GetDept()         // Извлечем из pool структуру для старого экземпляра в БД
		defer model.PutDept(oldDept, true) // Вернем структуру в pool

		newDept := model.GetDept()         // Извлечем из pool структуру для нового экземпляра в БД
		defer model.PutDept(newDept, true) // Вернем структуру в pool

		{ // Считаем состояние объекта до обновления и проверим его существование
			mylog.PrintfDebugMsg("Get row and check if it exists: reqID, PK", reqID, in.Deptno)
			oldDept.Deptno = in.Deptno // столбцы первичного ключа PK
			if exists, myerr = s.getDept(ctx, tx, oldDept); myerr != nil {
				return false, myerr
			}
			if !exists {
				mylog.PrintfDebugMsg("Row does not exists: reqID, PK", reqID, in.Deptno)
				return false, nil
			}
		} // Считаем состояние объекта до обновления и проверим его существование

		{ // выполняем проверки / действия на основании старых и новых значений атрибутов
			// Проверить изменение UK
			// для Dept PK и UK совпадают - эта проверка только для примера
			if oldDept.Deptno != in.Deptno {
				mylog.PrintfDebugMsg("Check if row already exists: reqID, Deptno", reqID, in.Deptno)
				var foo int
				exists, myerr := s.db.Get(reqID, tx, "DeptExists", &foo, in.Deptno)
				if myerr != nil {
					return false, myerr
				}
				if exists {
					return false, myerror.New("4004", "Error update - row with UK already exists: reqID, Deptno", reqID, in.Deptno).PrintfInfo()
				}
			}
		} // выполняем проверки / действия на основании старых и новых значений атрибутов

		{ // Выполняем обновление
			rows, myerr := s.db.Exec(reqID, tx, "UpdateDept", in)
			if myerr != nil {
				return false, myerr
			}
			// проверим количество обработанных строк
			if rows != 1 {
				return false, myerror.New("4004", "Error update: reqID, Deptno, rows", reqID, in.Deptno, rows).PrintfInfo()
			}

			// считаем объект по сурогатному PK
			exists, myerr := s.db.Get(reqID, tx, "GetDept", newDept, in.Deptno)
			if myerr != nil {
				return false, myerr
			}
			if !exists {
				return false, myerror.New("4004", "Row does not exists after creating: reqID, PK", reqID, in.Deptno).PrintfInfo()
			}
		} // Выполняем обновление

		{ // Обработаем вложенные объекты в рамках текущей транзации
			if in.Emps != nil {
				for _, inEmp := range in.Emps {
					// Копируем сурогатный PK во внешний ключ вложенного объекта
					inEmp.Deptno = null.Int{sql.NullInt64{int64(in.Deptno), true}}

					// обновляем вложенные объекты
					if exists, myerr = s.updateEmp(ctx, tx, inEmp, nil); myerr != nil {
						return false, myerr
					}
					// Если одного из вложенных объектов не существует - то создать его
					if !exists {
						if myerr = s.createEmp(ctx, tx, inEmp, nil); myerr != nil {
							return false, myerr
						}
					}
				}
			}
		} // Обработаем вложенные объекты в рамках текущей транзации

		// считаем обновленный объект из БД
		if out != nil {
			out.Deptno = in.Deptno // столбцы первичного ключа PK
			if exists, myerr = s.getDept(ctx, tx, out); myerr != nil {
				return false, myerr
			}
			// Проверка для отладки табличного API
			if !exists {
				return false, myerror.New("4004", "Row does not exists after updating: reqID, PK", reqID, in.Deptno).PrintfInfo()
			}
		}
		return true, nil
	}
	return false, myerror.New("4400", "Incorrect call 'in != nil  && tx != nil': reqID", reqID).PrintfInfo()
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
	var tx *mysql.Tx
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

	// Начинаем новую транзакцию
	if tx, myerr = s.db.Beginx(reqID); myerr != nil {
		return myerr
	}

	// Создаем объект в рамках транзации
	if myerr = s.createDept(ctx, tx, in, out); myerr != nil {
		_ = s.db.Rollback(reqID, tx)
		return myerr
	}

	// завершаем транзакцию
	return s.db.Commit(reqID, tx)
}

// UpdateDept update Dept
func (s *Service) UpdateDept(ctx context.Context, in *model.Dept, out *model.Dept) (exists bool, myerr error) {
	var tx *mysql.Tx
	reqID := myctx.FromContextRequestID(ctx) // RequestID передается через context

	// Начинаем новую транзакцию
	if tx, myerr = s.db.Beginx(reqID); myerr != nil {
		return false, myerr
	}

	// Создаем объект в рамках транзации
	if exists, myerr = s.updateDept(ctx, tx, in, out); myerr != nil {
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
