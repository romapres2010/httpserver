package postgres

import (
	"context"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx" // https://jmoiron.github.io/sqlx/
	_ "github.com/lib/pq"

	myerror "github.com/romapres2010/httpserver/error"
	mylog "github.com/romapres2010/httpserver/log"
)

// SQLStm represent SQL text and sqlStm
type SQLStm struct {
	sqlText   string     // текст SQL команды
	sqlStm    *sqlx.Stmt // подготовленный запрос к БД
	isPrepare bool       // признак, нужно ли предварительно готовить запрос (кроме DML)
}

// SQLStms represent SQLStm map
type SQLStms map[string]*SQLStm

// Service represent PostgreSQL service
type Service struct {
	ctx    context.Context    // корневой контекст при инициации сервиса
	cancel context.CancelFunc // функция закрытия глобального контекста
	cfg    *Config            // конфигурационные параметры
	errCh  chan<- error       // канал ошибок
	stopCh chan struct{}      // канал подтверждения об успешном закрытии сервиса

	pqDb    *sqlx.DB // БД PostgreSQL
	SQLStms SQLStms  // map SQL команд

	// вложенные сервисы
}

// Config конфигурационные настройки
type Config struct {
	ConnectString     string // строка подключения к БД
	PqHost            string
	PqPort            string
	PqDbname          string
	PqSslMode         string
	PqUser            string
	PqPass            string
	PqConnMaxLifetime int    // время жизни подключения в милисекундах
	PqMaxOpenConns    int    // максимальное количество открытых подключений
	PqMaxIdleConns    int    // максимальное количество простаивающих подключений
	PqDriverName      string // имя драйвера "postgres" | "pgx"
}

// New create PostgreSQL service
func New(ctx context.Context, errCh chan<- error, cfg *Config) (*Service, error) {
	var err error

	mylog.PrintfInfoMsg("Creating new PostgreSQL service")

	{ // входные проверки
		if cfg == nil {
			return nil, myerror.New("6030", "Empty PostgreSQL service config").PrintfInfo()
		}
		if cfg.PqHost == "" {
			return nil, myerror.New("6030", "Empty PostgreSQL Host parameter").PrintfInfo()
		}
	} // входные проверки

	// Создаем новый сервис
	service := &Service{
		cfg:    cfg,
		errCh:  errCh,
		stopCh: make(chan struct{}, 1), // канал подтверждения об успешном закрытии сервиса
	}

	// создаем контекст с отменой
	if ctx == nil {
		service.ctx, service.cancel = context.WithCancel(context.Background())
	} else {
		service.ctx, service.cancel = context.WithCancel(ctx)
	}

	// Сформировать строку подключения
	cfg.ConnectString = fmt.Sprintf("host=%s port=%s dbname=%s sslmode=%s user=%s password=%s ", cfg.PqHost, cfg.PqPort, cfg.PqDbname, cfg.PqSslMode, cfg.PqUser, cfg.PqPass)

	// открываем соединение с БД
	mylog.PrintfInfoMsg("Testing connect to PostgreSQL server: host, port, dbname, sslmode, user", cfg.PqHost, cfg.PqPort, cfg.PqDbname, cfg.PqSslMode, cfg.PqUser)
	if service.pqDb, err = sqlx.Connect(cfg.PqDriverName, cfg.ConnectString); err != nil {
		return nil, myerror.WithCause("4001", "Error connecting to PostgreSQL server: host, port, dbname, sslmode, user", err, cfg.PqHost, cfg.PqPort, cfg.PqDbname, cfg.PqSslMode, cfg.PqUser).PrintfInfo()
	}
	mylog.PrintfInfoMsg("Success connect to PostgreSQL server")

	// Наполним список SQL команд
	service.SQLStms = map[string]*SQLStm{
		"GetDept":         &SQLStm{"SELECT deptno, dname, loc FROM dept WHERE deptno = $1", nil, true},
		"GetDepts":        &SQLStm{"SELECT deptno, dname, loc FROM dept", nil, true},
		"GetDeptsPK":      &SQLStm{"SELECT deptno FROM dept", nil, true},
		"CreateDept":      &SQLStm{"INSERT INTO dept (deptno, dname, loc) VALUES (:deptno, :dname, :loc)", nil, false},
		"UpdateDept":      &SQLStm{"UPDATE dept SET dname = :dname, loc = :loc WHERE deptno = :deptno", nil, false},
		"GetEmp":          &SQLStm{"SELECT empno, ename, job, mgr, hiredate, sal, comm, deptno FROM emp WHERE empno = $1", nil, true},
		"GetEmpsByDept":   &SQLStm{"SELECT empno, ename, job, mgr, hiredate, sal, comm, deptno FROM emp WHERE deptno = $1", nil, true},
		"GetEmpsPKByDept": &SQLStm{"SELECT empno FROM emp WHERE deptno = $1", nil, true},
		"CreateEmp":       &SQLStm{"INSERT INTO emp (empno, ename, job, mgr, hiredate, sal, comm, deptno) VALUES (:empno, :ename, :job, :mgr, :hiredate, :sal, :comm, :deptno)", nil, false},
		"UpdateEmp":       &SQLStm{"UPDATE emp SET empno = :empno, ename = :ename, job = :job, mgr = :mgr, hiredate = :hiredate, sal = :sal, comm = :comm, deptno = :deptno WHERE empno = :empno", nil, false},
	}

	// Распарсим SQL команды
	for _, h := range service.SQLStms {
		if h.isPrepare {
			h.sqlStm, err = service.pqDb.Preparex(h.sqlText)
			if err != nil {
				return nil, myerror.WithCause("4002", "Error prepare SQL stament: SQL", err, h.sqlText).PrintfInfo()
			}
			mylog.PrintfInfoMsg("SQL stament is prepared: SQL", h.sqlText)
		}
	}

	{ // Устанавливаем параметры пула подключений
		service.pqDb.SetMaxOpenConns(cfg.PqMaxOpenConns)
		service.pqDb.SetMaxIdleConns(cfg.PqMaxIdleConns)
		service.pqDb.SetConnMaxLifetime(time.Duration(cfg.PqConnMaxLifetime * int(time.Millisecond)))
	} // Устанавливаем параметры пула подключений

	mylog.PrintfInfoMsg("PostgreSQL service is created")
	return service, nil
}

// Shutdown shutting down service
func (s *Service) Shutdown() (myerr error) {
	mylog.PrintfInfoMsg("Shutdowning PostgreSQL service")

	defer s.cancel() // закрываем контекст

	{
		// закрываем вложенные сервисы
	}

	mylog.PrintfInfoMsg("PostgreSQL service shutdown successfuly")
	return
}
