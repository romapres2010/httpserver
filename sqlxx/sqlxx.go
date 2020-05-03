package sqlxx

import (
	"database/sql"
	"fmt"
	"reflect"
	"sync/atomic"
	"time"

	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	myerror "github.com/romapres2010/httpserver/error"
	mylog "github.com/romapres2010/httpserver/log"
)

// Config конфигурационные настройки
type Config struct {
	ConnectString   string // строка подключения к БД
	Host            string // host БД
	Port            string // порт листенера БД
	Dbname          string // имя БД
	SslMode         string // режим SSL
	User            string // пользователь для подключения к БД
	Pass            string // пароль пользователя
	ConnMaxLifetime int    // время жизни подключения в милисекундах
	MaxOpenConns    int    // максимальное количество открытых подключений
	MaxIdleConns    int    // максимальное количество простаивающих подключений
	DriverName      string // имя драйвера "postgres" | "pgx"
}

// DB is a wrapper around sqlx.DB
type DB struct {
	*sqlx.DB
}

// Tx is an sqlx wrapper around sqlx.Tx
type Tx struct {
	*sqlx.Tx
}

// SQLStm represent SQL text and sqlStm
type SQLStm struct {
	Text      string     // текст SQL команды
	Stmt      *sqlx.Stmt // подготовленная SQL команда
	IsPrepare bool       // признак, нужно ли предварительно готовить SQL команду
}

// SQLStms represent SQLStm map
type SQLStms map[string]*SQLStm

// уникальный номер SQL
var sqlID uint64

// GetNextSQLID - запросить номер следующей SQL
func GetNextSQLID() uint64 {
	return atomic.AddUint64(&sqlID, 1)
}

// Connect - create new connect to DB
func Connect(cfg *Config) (db *DB, myerr error) {
	// Сформировать строку подключения
	cfg.ConnectString = fmt.Sprintf("host=%s port=%s dbname=%s sslmode=%s user=%s password=%s ", cfg.Host, cfg.Port, cfg.Dbname, cfg.SslMode, cfg.User, cfg.Pass)

	// открываем соединение с БД
	mylog.PrintfInfoMsg("Testing connect to PostgreSQL server: host, port, dbname, sslmode, user", cfg.Host, cfg.Port, cfg.Dbname, cfg.SslMode, cfg.User)

	sqlxDb, err := sqlx.Connect(cfg.DriverName, cfg.ConnectString)
	if err != nil {
		return nil, myerror.WithCause("4001", "Error connecting to PostgreSQL server: host, port, dbname, sslmode, user", err, cfg.Host, cfg.Port, cfg.Dbname, cfg.SslMode, cfg.User).PrintfInfo()
	}

	{ // Устанавливаем параметры пула подключений
		sqlxDb.SetMaxOpenConns(cfg.MaxOpenConns)
		sqlxDb.SetMaxIdleConns(cfg.MaxIdleConns)
		sqlxDb.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime * int(time.Millisecond)))
	} // Устанавливаем параметры пула подключений

	mylog.PrintfInfoMsg("Success connect to PostgreSQL server")
	return &DB{sqlxDb}, nil
}

// Preparex - prepare SQL statements
func Preparex(db *DB, SQLStms SQLStms) (myerr error) {
	var err error

	if db == nil || db.DB == nil {
		return myerror.New("4004", "DB is not defined").PrintfInfo()
	}

	// Подготовим SQL команды
	for _, h := range SQLStms {
		if h.IsPrepare {
			if h.Stmt, err = db.Preparex(h.Text); err != nil {
				return myerror.WithCause("4002", "Error prepare SQL stament: SQL", err, h.Text).PrintfInfo()
			}
			mylog.PrintfInfoMsg("SQL stament is prepared: SQL", h.Text)
		}
	}
	return nil
}

// Beginx - begin a new transaction
func Beginx(reqID uint64, db *DB) (tx *Tx, myerr error) {
	if db == nil || db.DB == nil {
		return nil, myerror.New("4004", "DB is not defined").PrintfInfo()
	}

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

	sqlxTx, err := db.Beginx()
	if err != nil {
		return nil, myerror.WithCause("4006", "Error begin a new transaction", err).PrintfInfo()
	}
	return &Tx{sqlxTx}, nil
}

// Rollback - rollback the transaction
func Rollback(reqID uint64, tx *Tx) (myerr error) {
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

// Commit - commit the transaction
func Commit(reqID uint64, tx *Tx) (myerr error) {
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

// Select - represent common task in process SQL Select statement
func Select(reqID uint64, tx *Tx, sqlStm *SQLStm, dest interface{}, args ...interface{}) (myerr error) {
	if dest != nil && !reflect.ValueOf(dest).IsNil() {
		if sqlStm == nil {
			return myerror.New("4100", "SQL statement is not defined: reqID", reqID).PrintfInfo()
		}

		// Получить уникальный номер SQL
		sqlID := GetNextSQLID()

		mylog.PrintfDebugMsg("reqID, sqlID, SQL", reqID, sqlID, sqlStm.Text)

		// функция восстановления после паники
		defer func() {
			r := recover()
			if r != nil {
				msg := "Recover from panic: reqID, sqlID, SQL"
				switch t := r.(type) {
				case string:
					myerr = myerror.New("4446", msg, t, reqID, sqlID, sqlStm.Text).PrintfInfo()
				case error:
					myerr = myerror.WithCause("4446", msg, t, reqID, sqlID, sqlStm.Text).PrintfInfo()
				default:
					myerr = myerror.New("4446", msg, reqID, sqlID, sqlStm.Text).PrintfInfo()
				}
			}
		}()

		stm := sqlStm.Stmt
		// Помещаем запрос в рамки транзакции
		if tx != nil {
			stm = tx.Stmtx(sqlStm.Stmt)
		}

		//Выполняем запрос
		if err := stm.Select(dest, args...); err != nil {
			return myerror.WithCause("4003", "Error Select SQL statement: reqID, sqlID, SQL", err, reqID, sqlID, sqlStm.Text).PrintfInfo()
		}
		return nil
	}
	return myerror.New("4400", "Incorrect call - nil dest interface{} pointer: reqID", reqID).PrintfInfo()
}

// Get - represent common task in process SQL Select statement with only one rows
func Get(reqID uint64, tx *Tx, sqlStm *SQLStm, dest interface{}, args ...interface{}) (exists bool, myerr error) {

	if dest != nil && !reflect.ValueOf(dest).IsNil() {
		if sqlStm == nil {
			return false, myerror.New("4100", "SQL statement is not defined: reqID", reqID).PrintfInfo()
		}

		// Получить уникальный номер SQL
		sqlID := GetNextSQLID()

		mylog.PrintfDebugMsg("reqID, sqlID, SQL", reqID, sqlID, sqlStm.Text)

		// функция восстановления после паники
		defer func() {
			r := recover()
			if r != nil {
				msg := "Recover from panic: reqID, sqlID, SQL"
				switch t := r.(type) {
				case string:
					myerr = myerror.New("4446", msg, t, reqID, sqlID, sqlStm.Text).PrintfInfo()
				case error:
					myerr = myerror.WithCause("4446", msg, t, reqID, sqlID, sqlStm.Text).PrintfInfo()
				default:
					myerr = myerror.New("4446", msg, reqID, sqlID, sqlStm.Text).PrintfInfo()
				}
			}
		}()

		stm := sqlStm.Stmt
		// Помещаем запрос в рамки транзакции
		if tx != nil {
			stm = tx.Stmtx(sqlStm.Stmt)
		}

		//Выполняем запрос
		if err := stm.Get(dest, args...); err != nil {
			// NO_DATA_FOUND - ошибкой не считаем
			if err == sql.ErrNoRows {
				return false, nil
			}
			return false, myerror.WithCause("4003", "Error Get SQL statement: reqID, sqlID, SQL", err, reqID, sqlID, sqlStm.Text).PrintfInfo()
		}
		return true, nil
	}
	return false, myerror.New("4400", "Incorrect call - nil dest interface{} pointer: reqID", reqID).PrintfInfo()
}

// Exec - represent common task in process DML statement
func Exec(reqID uint64, tx *Tx, sqlStm *SQLStm, args interface{}) (rows int64, myerr error) {

	if args != nil && !reflect.ValueOf(args).IsNil() {
		if sqlStm == nil {
			return 0, myerror.New("4100", "SQL statement is not defined: reqID", reqID).PrintfInfo()
		}

		// Получить уникальный номер SQL
		sqlID := GetNextSQLID()

		mylog.PrintfDebugMsg("reqID, sqlID, SQL", reqID, sqlID, sqlStm.Text)

		// функция восстановления после паники
		defer func() {
			r := recover()
			if r != nil {
				msg := "Recover from panic: reqID, sqlID, SQL"
				switch t := r.(type) {
				case string:
					myerr = myerror.New("4446", msg, t, reqID, sqlID, sqlStm.Text).PrintfInfo()
				case error:
					myerr = myerror.WithCause("4446", msg, t, reqID, sqlID, sqlStm.Text).PrintfInfo()
				default:
					myerr = myerror.New("4446", msg, reqID, sqlID, sqlStm.Text).PrintfInfo()
				}
			}
		}()

		// Проверяем определен ли контекст транзакции
		if tx == nil {
			return 0, myerror.New("4004", "Transaction is not defined: reqID, sqlID, SQL", reqID, sqlID, sqlStm.Text).PrintfInfo()
		}

		// Выполняем DML
		res, err := tx.NamedExec(sqlStm.Text, args)
		if err != nil {
			return 0, myerror.WithCause("4005", "Error Exec SQL statement: reqID, sqlID, SQL, args", err, reqID, sqlID, sqlStm.Text, args).PrintfInfo()
		}

		// Количество обработанных строк
		rows, err = res.RowsAffected()
		if err != nil {
			return 0, myerror.WithCause("4005", "Error count affected rows: reqID, sqlID, SQL, args", err, reqID, sqlID, sqlStm.Text, args).PrintfInfo()
		}
		return rows, nil
	}
	return 0, myerror.New("4400", "Incorrect call - nil args interface{} pointer: reqID", reqID).PrintfInfo()
}

// Create - create a new row
func Create(reqID uint64, tx *Tx, getSQLStm *SQLStm, crtSQLStm *SQLStm, in interface{}, out interface{}, args ...interface{}) (myerr error) {

	if in != nil && !reflect.ValueOf(in).IsNil() && getSQLStm != nil && crtSQLStm != nil {
		// Проверим, существует ли строка
		if out != nil && reflect.ValueOf(out).Pointer() != 0 {
			mylog.PrintfDebugMsg("Check if row already exists: reqID", reqID)
			exists, err := Get(reqID, tx, getSQLStm, out, args...)
			if err != nil {
				return err
			}
			if exists {
				return myerror.New("4004", "Error create - row already exists: reqID", reqID).PrintfInfo()
			}
		}

		// Выполняем вставку
		rows, myerr := Exec(reqID, tx, crtSQLStm, in)
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

// Update - update the row
func Update(reqID uint64, tx *Tx, getSQLStm *SQLStm, updSQLStm *SQLStm, in interface{}, out interface{}, args ...interface{}) (myerr error) {

	if in != nil && !reflect.ValueOf(in).IsNil() && getSQLStm != nil && updSQLStm != nil {
		// Проверим, существует ли строка
		if out != nil && reflect.ValueOf(out).Pointer() != 0 {
			mylog.PrintfDebugMsg("Check if row already exists: reqID", reqID)
			exists, err := Get(reqID, tx, getSQLStm, out, args...)
			if err != nil {
				return err
			}
			if !exists {
				return myerror.New("4004", "Error update - row does not exists: reqID", reqID).PrintfInfo()
			}
		}

		// Выполняем обновление
		rows, myerr := Exec(reqID, tx, updSQLStm, in)
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
