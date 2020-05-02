package postgres

import (
	"database/sql"
	"reflect"
	"sync/atomic"

	"github.com/jmoiron/sqlx"

	myerror "github.com/romapres2010/httpserver/error"
	mylog "github.com/romapres2010/httpserver/log"
)

// уникальный номер SQL
var sqlID uint64

// GetNextSQLID - запросить номер следующей SQL
func GetNextSQLID() uint64 {
	return atomic.AddUint64(&sqlID, 1)
}

// beginx - begin a new transaction
func beginx(reqID uint64, pqDb *sqlx.DB) (tx *sqlx.Tx, myerr error) {
	var err error

	// функция восстановления после паники
	defer func() {
		r := recover()
		if r != nil {
			msg := "PostgreSQL recover from panic: begin transaction: reqID"
			switch t := r.(type) {
			case string:
				myerr = myerror.New("4008", msg, reqID, t).PrintfInfo()
			case error:
				myerr = myerror.WithCause("4006", msg, t, reqID).PrintfInfo()
			default:
				myerr = myerror.New("4006", msg, reqID).PrintfInfo()
			}
		}
	}()

	if tx, err = pqDb.Beginx(); err != nil {
		return nil, myerror.WithCause("4006", "Error begin a new transaction", err).PrintfInfo()
	}
	return tx, nil
}

// rollback - rollback the transaction
func rollback(reqID uint64, tx *sqlx.Tx) (myerr error) {
	// Проверяем определен ли контекст транзакции
	if tx == nil {
		return myerror.New("4004", "Transaction is not defined: reqID", reqID).PrintfInfo()
	}

	// функция восстановления после паники
	defer func() {
		r := recover()
		if r != nil {
			msg := "PostgreSQL recover from panic: rollback transaction: reqID"
			switch t := r.(type) {
			case string:
				myerr = myerror.New("4008", msg, reqID, t).PrintfInfo()
			case error:
				myerr = myerror.WithCause("4006", msg, t, reqID).PrintfInfo()
			default:
				myerr = myerror.New("4006", msg, reqID).PrintfInfo()
			}
		}
	}()

	if err := tx.Rollback(); err != nil {
		return myerror.WithCause("4008", "Error rollback the transaction: reqID", err, reqID).PrintfInfo()
	}
	return nil
}

// commit - commit the transaction
func commit(reqID uint64, tx *sqlx.Tx) (myerr error) {
	// Проверяем определен ли контекст транзакции
	if tx == nil {
		return myerror.New("4004", "Transaction is not defined: reqID", reqID).PrintfInfo()
	}

	// функция восстановления после паники
	defer func() {
		r := recover()
		if r != nil {
			msg := "PostgreSQL recover from panic: commit transaction: reqID"
			switch t := r.(type) {
			case string:
				myerr = myerror.New("4008", msg, reqID, t).PrintfInfo()
			case error:
				myerr = myerror.WithCause("4006", msg, t, reqID).PrintfInfo()
			default:
				myerr = myerror.New("4006", msg, reqID).PrintfInfo()
			}
		}
	}()

	if err := tx.Commit(); err != nil {
		return myerror.WithCause("4008", "Error commit the transaction: reqID", err, reqID).PrintfInfo()
	}
	return nil
}

// selectx - represent common task in process SQL Select statement
func selectx(reqID uint64, tx *sqlx.Tx, sqlStm *SQLStm, dest interface{}, args ...interface{}) (myerr error) {
	if dest != nil && reflect.ValueOf(dest).Pointer() != 0 {
		if sqlStm == nil {
			return myerror.New("4100", "SQL statement is not defined: reqID", reqID).PrintfInfo()
		}

		// Получить уникальный номер SQL
		sqlID := GetNextSQLID()

		mylog.PrintfDebugMsg("reqID, sqlID, SQL", reqID, sqlID, sqlStm.sqlText)

		// функция восстановления после паники
		defer func() {
			r := recover()
			if r != nil {
				msg := "Recover from panic: reqID, sqlID, SQL"
				switch t := r.(type) {
				case string:
					myerr = myerror.New("4446", msg, t, reqID, sqlID, sqlStm.sqlText).PrintfInfo()
				case error:
					myerr = myerror.WithCause("4446", msg, t, reqID, sqlID, sqlStm.sqlText).PrintfInfo()
				default:
					myerr = myerror.New("4446", msg, reqID, sqlID, sqlStm.sqlText).PrintfInfo()
				}
			}
		}()

		stm := sqlStm.sqlStm
		// Помещаем запрос в рамки транзакции
		if tx != nil {
			stm = tx.Stmtx(sqlStm.sqlStm)
		}

		//Выполняем запрос
		if err := stm.Select(dest, args...); err != nil {
			return myerror.WithCause("4003", "Error Select SQL statement: reqID, sqlID, SQL", err, reqID, sqlID, sqlStm.sqlText).PrintfInfo()
		}
		return nil
	}
	return myerror.New("4400", "Incorrect call - nil dest interface{} pointer: reqID", reqID).PrintfInfo()
}

// get - represent common task in process SQL Select statement with only one rows
func get(reqID uint64, tx *sqlx.Tx, sqlStm *SQLStm, dest interface{}, args ...interface{}) (exists bool, myerr error) {

	if dest != nil && reflect.ValueOf(dest).Pointer() != 0 {
		if sqlStm == nil {
			return false, myerror.New("4100", "SQL statement is not defined: reqID", reqID).PrintfInfo()
		}

		// Получить уникальный номер SQL
		sqlID := GetNextSQLID()

		mylog.PrintfDebugMsg("reqID, sqlID, SQL", reqID, sqlID, sqlStm.sqlText)

		// функция восстановления после паники
		defer func() {
			r := recover()
			if r != nil {
				msg := "Recover from panic: reqID, sqlID, SQL"
				switch t := r.(type) {
				case string:
					myerr = myerror.New("4446", msg, t, reqID, sqlID, sqlStm.sqlText).PrintfInfo()
				case error:
					myerr = myerror.WithCause("4446", msg, t, reqID, sqlID, sqlStm.sqlText).PrintfInfo()
				default:
					myerr = myerror.New("4446", msg, reqID, sqlID, sqlStm.sqlText).PrintfInfo()
				}
			}
		}()

		stm := sqlStm.sqlStm
		// Помещаем запрос в рамки транзакции
		if tx != nil {
			stm = tx.Stmtx(sqlStm.sqlStm)
		}

		//Выполняем запрос
		if err := stm.Get(dest, args...); err != nil {
			// NO_DATA_FOUND - ошибкой не считаем
			if err == sql.ErrNoRows {
				return false, nil
			}
			return false, myerror.WithCause("4003", "Error Get SQL statement: reqID, sqlID, SQL", err, reqID, sqlID, sqlStm.sqlText).PrintfInfo()
		}
		return true, nil
	}
	return false, myerror.New("4400", "Incorrect call - nil dest interface{} pointer: reqID", reqID).PrintfInfo()
}

// exec - represent common task in process DML statement
func exec(reqID uint64, tx *sqlx.Tx, sqlStm *SQLStm, args interface{}) (rows int64, myerr error) {

	if args != nil && reflect.ValueOf(args).Pointer() != 0 {
		if sqlStm == nil {
			return 0, myerror.New("4100", "SQL statement is not defined: reqID", reqID).PrintfInfo()
		}

		// Получить уникальный номер SQL
		sqlID := GetNextSQLID()

		mylog.PrintfDebugMsg("reqID, sqlID, SQL", reqID, sqlID, sqlStm.sqlText)

		// функция восстановления после паники
		defer func() {
			r := recover()
			if r != nil {
				msg := "Recover from panic: reqID, sqlID, SQL"
				switch t := r.(type) {
				case string:
					myerr = myerror.New("4446", msg, t, reqID, sqlID, sqlStm.sqlText).PrintfInfo()
				case error:
					myerr = myerror.WithCause("4446", msg, t, reqID, sqlID, sqlStm.sqlText).PrintfInfo()
				default:
					myerr = myerror.New("4446", msg, reqID, sqlID, sqlStm.sqlText).PrintfInfo()
				}
			}
		}()

		// Проверяем определен ли контекст транзакции
		if tx == nil {
			return 0, myerror.New("4004", "Transaction is not defined: reqID, sqlID, SQL", reqID, sqlID, sqlStm.sqlText).PrintfInfo()
		}

		// Выполняем DML
		res, err := tx.NamedExec(sqlStm.sqlText, args)
		if err != nil {
			return 0, myerror.WithCause("4005", "Error Exec SQL statement: reqID, sqlID, SQL, args", err, reqID, sqlID, sqlStm.sqlText, args).PrintfInfo()
		}

		// Количество обработанных строк
		rows, err = res.RowsAffected()
		if err != nil {
			return 0, myerror.WithCause("4005", "Error count affected rows: reqID, sqlID, SQL, args", err, reqID, sqlID, sqlStm.sqlText, args).PrintfInfo()
		}
		return rows, nil
	}
	return 0, myerror.New("4400", "Incorrect call - nil args interface{} pointer: reqID", reqID).PrintfInfo()
}

// сreate - create a new row
func сreate(reqID uint64, tx *sqlx.Tx, getSQLStm *SQLStm, crtSQLStm *SQLStm, in interface{}, out interface{}, args ...interface{}) (myerr error) {

	if in != nil && reflect.ValueOf(in).Pointer() != 0 && getSQLStm != nil && crtSQLStm != nil {
		// Проверим, существует ли строка
		if out != nil && reflect.ValueOf(out).Pointer() != 0 {
			mylog.PrintfDebugMsg("Check if row already exists: reqID", reqID)
			exists, err := get(reqID, tx, getSQLStm, out, args...)
			if err != nil {
				return err
			}
			if exists {
				return myerror.New("4004", "Error create - row already exists: reqID", reqID).PrintfInfo()
			}
		}

		// Выполняем вставку
		rows, myerr := exec(reqID, tx, crtSQLStm, in)
		if myerr != nil {
			return myerr
		}
		// проверим количество обработанных строк
		if rows != 1 {
			return myerror.New("4004", "Error exec SQL statement: reqID, rows", reqID, rows).PrintfInfo()
		}

		return nil
	}
	return myerror.New("4400", "Incorrect call - nill interface{} pointer: reqID", reqID).PrintfInfo()
}

// update - update the row
func update(reqID uint64, tx *sqlx.Tx, getSQLStm *SQLStm, updSQLStm *SQLStm, in interface{}, out interface{}, args ...interface{}) (myerr error) {

	if in != nil && reflect.ValueOf(in).Pointer() != 0 && getSQLStm != nil && updSQLStm != nil {
		// Проверим, существует ли строка
		if out != nil && reflect.ValueOf(out).Pointer() != 0 {
			mylog.PrintfDebugMsg("Check if row already exists: reqID", reqID)
			exists, err := get(reqID, tx, getSQLStm, out, args...)
			if err != nil {
				return err
			}
			if !exists {
				return myerror.New("4004", "Error update - row does not exists: reqID", reqID).PrintfInfo()
			}
		}

		// Выполняем обновление
		rows, myerr := exec(reqID, tx, updSQLStm, in)
		if myerr != nil {
			return myerr
		}
		// проверим количество обработанных строк
		if rows != 1 {
			return myerror.New("4004", "Error exec SQL statement: reqID, rows", reqID, rows).PrintfInfo()
		}

		return nil
	}
	return myerror.New("4400", "Incorrect call - nill interface{} pointer: reqID", reqID).PrintfInfo()
}
