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

// Config конфигурационные настройки БД
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
	DriverName      string // имя драйвера "postgres" | "pgx" | "godror"
}

// DB is a wrapper around sqlx.DB
type DB struct {
	*sqlx.DB

	cfg     *Config
	sqlStms SQLStms // SQL команды
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

// New - create new connect to DB
func New(cfg *Config, sqlStms SQLStms) (db *DB, myerr error) {
	{ // входные проверки
		if cfg == nil {
			return nil, myerror.New("6030", "Empty SQLxx config").PrintfInfo()
		}
	} // входные проверки

	// Сформировать строку подключения
	cfg.ConnectString = fmt.Sprintf("host=%s port=%s dbname=%s sslmode=%s user=%s password=%s ", cfg.Host, cfg.Port, cfg.Dbname, cfg.SslMode, cfg.User, cfg.Pass)

	// Создаем новый сервис
	db = &DB{
		cfg:     cfg,
		sqlStms: sqlStms,
	}

	// открываем соединение с БД
	mylog.PrintfInfoMsg("Testing connect to PostgreSQL server: host, port, dbname, sslmode, user", cfg.Host, cfg.Port, cfg.Dbname, cfg.SslMode, cfg.User)

	sqlxDb, err := sqlx.Connect(cfg.DriverName, cfg.ConnectString)
	if err != nil {
		return nil, myerror.WithCause("4001", "Error connecting to PostgreSQL server: host, port, dbname, sslmode, user", err, cfg.Host, cfg.Port, cfg.Dbname, cfg.SslMode, cfg.User).PrintfInfo()
	}
	db.DB = sqlxDb

	{ // Устанавливаем параметры пула подключений
		db.DB.SetMaxOpenConns(cfg.MaxOpenConns)
		db.DB.SetMaxIdleConns(cfg.MaxIdleConns)
		db.DB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime * int(time.Millisecond)))
	} // Устанавливаем параметры пула подключений

	// Подготовим SQL команды
	if err = db.Preparex(sqlStms); err != nil {
		return nil, err
	}

	mylog.PrintfInfoMsg("Success connect to PostgreSQL server")
	return db, nil
}

// Preparex - prepare SQL statements
func (db *DB) Preparex(SQLStms SQLStms) (myerr error) {
	var err error

	if db.DB == nil {
		return myerror.New("4004", "DB is not defined").PrintfInfo()
	}

	// Подготовим SQL команды
	for _, h := range SQLStms {
		if h.IsPrepare {
			if h.Stmt, err = db.DB.Preparex(h.Text); err != nil {
				return myerror.WithCause("4002", "Error prepare SQL stament: SQL", err, h.Text).PrintfInfo()
			}
			mylog.PrintfInfoMsg("SQL stament is prepared: SQL", h.Text)
		}
	}
	return nil
}

// Beginx - begin a new transaction
func (db *DB) Beginx(reqID uint64) (tx *Tx, myerr error) {
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

	if db.DB == nil {
		return nil, myerror.New("4004", "DB is not defined").PrintfInfo()
	}

	sqlxTx, err := db.DB.Beginx()
	if err != nil {
		return nil, myerror.WithCause("4006", "Error begin a new transaction", err).PrintfInfo()
	}
	return &Tx{sqlxTx}, nil
}

// Rollback - rollback the transaction
func (db *DB) Rollback(reqID uint64, tx *Tx) (myerr error) {
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

	// Проверяем определен ли контекст транзакции
	if tx == nil {
		return myerror.New("4004", "Transaction is not defined: reqID", reqID).PrintfInfo()
	}

	if err := tx.Rollback(); err != nil {
		return myerror.WithCause("4008", "Error rollback the transaction: reqID", err, reqID).PrintfInfo()
	}
	mylog.PrintfDebugMsgDepth("Transaction rollbacked", 1, reqID)
	return nil
}

// Commit - commit the transaction
func (db *DB) Commit(reqID uint64, tx *Tx) (myerr error) {
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

	// Проверяем определен ли контекст транзакции
	if tx == nil {
		return myerror.New("4004", "Transaction is not defined: reqID", reqID).PrintfInfo()
	}

	if err := tx.Commit(); err != nil {
		return myerror.WithCause("4008", "Error commit the transaction: reqID", err, reqID).PrintfInfo()
	}
	mylog.PrintfDebugMsgDepth("Transaction commited", 1, reqID)
	return nil
}

// Select - represent common task in process SQL Select statement
func (db *DB) Select(reqID uint64, tx *Tx, sqlT string, dest interface{}, args ...interface{}) (myerr error) {
	sqlStm, ok := db.sqlStms[sqlT]
	if !ok {
		return myerror.New("4100", "SQL statement is not defined: reqID, sql", reqID, sqlT).PrintfInfo()
	}

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

	if dest != nil && !reflect.ValueOf(dest).IsNil() {
		// Получить уникальный номер SQL
		sqlID := GetNextSQLID()

		mylog.PrintfDebugMsg("reqID, sqlID, SQL", reqID, sqlID, sqlStm.Text)

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
	return myerror.New("4400", "Incorrect call - nil dest interface{} pointer: reqID, sql", reqID, sqlT).PrintfInfo()
}

// Get - represent common task in process SQL Select statement with only one rows
func (db *DB) Get(reqID uint64, tx *Tx, sqlT string, dest interface{}, args ...interface{}) (exists bool, myerr error) {
	sqlStm, ok := db.sqlStms[sqlT]
	if !ok {
		return false, myerror.New("4100", "SQL statement is not defined: reqID, sql", reqID, sqlT).PrintfInfo()
	}

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

	if dest != nil && !reflect.ValueOf(dest).IsNil() {
		// Получить уникальный номер SQL
		sqlID := GetNextSQLID()

		mylog.PrintfDebugMsg("reqID, sqlID, SQL", reqID, sqlID, sqlStm.Text)

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
	return false, myerror.New("4400", "Incorrect call - nil dest interface{} pointer: reqID, sql", reqID, sqlT).PrintfInfo()
}

// Exec - represent common task in process DML statement
func (db *DB) Exec(reqID uint64, tx *Tx, sqlT string, args interface{}) (rows int64, myerr error) {
	sqlStm, ok := db.sqlStms[sqlT]
	if !ok {
		return 0, myerror.New("4100", "SQL statement is not defined: reqID, sql", reqID, sqlT).PrintfInfo()
	}

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

	if args != nil && !reflect.ValueOf(args).IsNil() {
		// Получить уникальный номер SQL
		sqlID := GetNextSQLID()

		mylog.PrintfDebugMsg("reqID, sqlID, SQL", reqID, sqlID, sqlStm.Text)

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
	return 0, myerror.New("4400", "Incorrect call - nil args interface{} pointer: reqID, sql", reqID, sqlT).PrintfInfo()
}
