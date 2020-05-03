package db

import (
	"context"

	myerror "github.com/romapres2010/httpserver/error"
	mylog "github.com/romapres2010/httpserver/log"
	sql "github.com/romapres2010/httpserver/sqlxx"
)

// Service represent DB service
type Service struct {
	ctx    context.Context    // корневой контекст при инициации сервиса
	cancel context.CancelFunc // функция закрытия глобального контекста
	cfg    *Config            // конфигурационные параметры
	errCh  chan<- error       // канал ошибок
	stopCh chan struct{}      // канал подтверждения об успешном закрытии сервиса

	db      *sql.DB     // БД
	SQLStms sql.SQLStms // SQL команды

	// вложенные сервисы
}

// Config конфигурационные настройки
type Config struct {
	SQLCfg sql.Config
}

// New create DB service
func New(ctx context.Context, errCh chan<- error, cfg *Config) (*Service, error) {
	var err error

	mylog.PrintfInfoMsg("Creating new DB service")

	{ // входные проверки
		if cfg == nil {
			return nil, myerror.New("6030", "Empty DB service config").PrintfInfo()
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

	// Создадим подключение к БД
	if service.db, err = sql.Connect(&cfg.SQLCfg); err != nil {
		return nil, err
	}

	// Наполним список SQL команд
	service.SQLStms = map[string]*sql.SQLStm{
		"GetDept":         &sql.SQLStm{"SELECT deptno, dname, loc FROM dept WHERE deptno = $1", nil, true},
		"GetDepts":        &sql.SQLStm{"SELECT deptno, dname, loc FROM dept", nil, true},
		"GetDeptsPK":      &sql.SQLStm{"SELECT deptno FROM dept", nil, true},
		"CreateDept":      &sql.SQLStm{"INSERT INTO dept (deptno, dname, loc) VALUES (:deptno, :dname, :loc)", nil, false},
		"UpdateDept":      &sql.SQLStm{"UPDATE dept SET dname = :dname, loc = :loc WHERE deptno = :deptno", nil, false},
		"GetEmp":          &sql.SQLStm{"SELECT empno, ename, job, mgr, hiredate, sal, comm, deptno FROM emp WHERE empno = $1", nil, true},
		"GetEmpsByDept":   &sql.SQLStm{"SELECT empno, ename, job, mgr, hiredate, sal, comm, deptno FROM emp WHERE deptno = $1", nil, true},
		"GetEmpsPKByDept": &sql.SQLStm{"SELECT empno FROM emp WHERE deptno = $1", nil, true},
		"CreateEmp":       &sql.SQLStm{"INSERT INTO emp (empno, ename, job, mgr, hiredate, sal, comm, deptno) VALUES (:empno, :ename, :job, :mgr, :hiredate, :sal, :comm, :deptno)", nil, false},
		"UpdateEmp":       &sql.SQLStm{"UPDATE emp SET empno = :empno, ename = :ename, job = :job, mgr = :mgr, hiredate = :hiredate, sal = :sal, comm = :comm, deptno = :deptno WHERE empno = :empno", nil, false},
	}

	// Подготовим SQL команды
	if err = sql.Preparex(service.db, service.SQLStms); err != nil {
		return nil, err
	}

	mylog.PrintfInfoMsg("DB service is created")
	return service, nil
}

// Shutdown shutting down service
func (s *Service) Shutdown() (myerr error) {
	mylog.PrintfInfoMsg("Shutdowning DB service")

	defer s.cancel() // закрываем контекст

	{
		// закрываем вложенные сервисы
	}

	mylog.PrintfInfoMsg("DB service shutdown successfuly")
	return
}
