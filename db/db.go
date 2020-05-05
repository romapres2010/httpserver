package db

import (
	"context"

	myerror "github.com/romapres2010/httpserver/error"
	mylog "github.com/romapres2010/httpserver/log"
	mysql "github.com/romapres2010/httpserver/sqlxx"
)

// Service represent DB service
type Service struct {
	ctx    context.Context    // корневой контекст при инициации сервиса
	cancel context.CancelFunc // функция закрытия глобального контекста
	cfg    *Config            // конфигурационные параметры
	errCh  chan<- error       // канал ошибок
	stopCh chan struct{}      // канал подтверждения об успешном закрытии сервиса

	db      *mysql.DB     // БД
	SQLStms mysql.SQLStms // SQL команды

	// вложенные сервисы
}

// Config конфигурационные настройки
type Config struct {
	SQLCfg mysql.Config
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

	// Наполним список SQL команд
	sqlStms := map[string]*mysql.SQLStm{
		"GetDept":         &mysql.SQLStm{"SELECT deptno, dname, loc FROM dept WHERE deptno = $1", nil, true},
		"GetDeptUK":       &mysql.SQLStm{"SELECT deptno, dname, loc FROM dept WHERE deptno = $1", nil, true},
		"DeptExists":      &mysql.SQLStm{"SELECT 1 FROM dept WHERE deptno = $1", nil, true},
		"GetDepts":        &mysql.SQLStm{"SELECT deptno, dname, loc FROM dept", nil, true},
		"GetDeptsPK":      &mysql.SQLStm{"SELECT deptno FROM dept", nil, true},
		"CreateDept":      &mysql.SQLStm{"INSERT INTO dept (deptno, dname, loc) VALUES (:deptno, :dname, :loc)", nil, false},
		"UpdateDept":      &mysql.SQLStm{"UPDATE dept SET dname = :dname, loc = :loc WHERE deptno = :deptno", nil, false},
		"EmpExists":       &mysql.SQLStm{"SELECT 1 FROM emp WHERE empno = $1", nil, true},
		"GetEmp":          &mysql.SQLStm{"SELECT empno, ename, job, mgr, hiredate, sal, comm, deptno FROM emp WHERE empno = $1", nil, true},
		"GetEmpUK":        &mysql.SQLStm{"SELECT empno, ename, job, mgr, hiredate, sal, comm, deptno FROM emp WHERE empno = $1", nil, true},
		"GetEmpsByDept":   &mysql.SQLStm{"SELECT empno, ename, job, mgr, hiredate, sal, comm, deptno FROM emp WHERE deptno = $1", nil, true},
		"GetEmpsPKByDept": &mysql.SQLStm{"SELECT empno FROM emp WHERE deptno = $1", nil, true},
		"CreateEmp":       &mysql.SQLStm{"INSERT INTO emp (empno, ename, job, mgr, hiredate, sal, comm, deptno) VALUES (:empno, :ename, :job, :mgr, :hiredate, :sal, :comm, :deptno)", nil, false},
		"UpdateEmp":       &mysql.SQLStm{"UPDATE emp SET empno = :empno, ename = :ename, job = :job, mgr = :mgr, hiredate = :hiredate, sal = :sal, comm = :comm, deptno = :deptno WHERE empno = :empno", nil, false},
	}

	// Создадим подключение к БД
	if service.db, err = mysql.New(&cfg.SQLCfg, sqlStms); err != nil {
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
