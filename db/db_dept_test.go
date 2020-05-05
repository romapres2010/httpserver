package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"testing"

	"github.com/hashicorp/logutils"
	mylog "github.com/romapres2010/httpserver/log"
	model "github.com/romapres2010/httpserver/model"
	mysql "github.com/romapres2010/httpserver/sqlxx"
	"gopkg.in/guregu/null.v4"
)

var testDB *Service

func InitDbTest() *Service {
	var err error

	// Фильтр логирования
	var logFilter = &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "WARN", "INFO", "ERROR"},
		MinLevel: logutils.LogLevel("INFO"),
		Writer:   os.Stderr,
	}

	// Переопределяем глобальный логер на кастомный
	log.SetOutput(logFilter)

	// конфигурационный файл для БД
	pqServiceCfg := Config{mysql.Config{
		Host:            "130.61.117.149",
		Port:            "5432",
		Dbname:          "test_database",
		SslMode:         "disable",
		User:            "test_user",
		Pass:            "qwerty",
		ConnMaxLifetime: 1000,
		MaxOpenConns:    16,
		MaxIdleConns:    8,
		DriverName:      "pgx",
	},
	}

	// канал ошибок для PostgreSQL сервиса
	pqServiceErrCh := make(chan error, 1)

	// создаем БД
	if testDB, err = New(nil, pqServiceErrCh, &pqServiceCfg); err != nil {
		return nil
	}

	return testDB
}

func BenchmarkPgDb_GetDept(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping test in short mode.")
	}

	// Инициализируем БД для теста
	if testDB == nil {
		testDB = InitDbTest()
	}

	// создадим корневой контекст
	ctx, _ := context.WithCancel(context.Background())

	deptsPK := model.DeptPKs(make([]*model.DeptPK, 0))

	// считаем все PK для dept
	err := testDB.GetDeptsPK(ctx, &deptsPK)
	if err != nil {
		b.Errorf("\n PgDb.GetDept() - db.GetDeptsPK() error = %v", err)
		return
	}
	deptsPKlen := len(deptsPK)
	mylog.PrintfInfoMsg("len(deptsPK)", deptsPKlen)

	// готовим структуру для тестирования
	tests := struct {
		name string
		p    *Service
		args model.DeptPKs
		//		want    *model.Dept
		//		wantErr bool
	}{
		name: "test",
		p:    testDB,
		args: deptsPK,
	}

	b.ResetTimer()
	b.SetBytes(200000)
	//b.SetParallelism(2)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() { // The loop body is executed b.N times total across all goroutines
			b.StopTimer()

			b.StartTimer()
			vOut := model.GetDept()                                // Извлечем из pool новую структуру
			vOut.Deptno = tests.args[rand.Intn(deptsPKlen)].Deptno // случайным образом выбираем PK
			//vOut := &model.Dept{}

			_, err := tests.p.GetDept(ctx, vOut)
			if err != nil {
				b.Errorf("\n PgDb.GetDept() - error GetDept(d.Deptno), %v", fmt.Sprintf("%+v", err))
				return
			}

			model.PutDept(vOut, true) // возвращаем в pool струкуру
		}
	})
}

func BenchmarkPgDb_CreateDept(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping test in short mode.")
	}

	// Инициализируем БД для теста
	if testDB == nil {
		testDB = InitDbTest()
	}

	// создадим корневой контекст
	ctx, _ := context.WithCancel(context.Background())

	//готовим фиктивные данные двя вставки
	var emp = model.Emp{
		Empno: 0,
		Ename: null.String{sql.NullString{
			String: "Ename - insertBenchmark",
			Valid:  true,
		}},
		Job: null.String{sql.NullString{
			String: "Job - insertBenchmark",
			Valid:  true,
		}},
		Mgr: null.Int{sql.NullInt64{
			Int64: 0,
			Valid: false,
		}},
		Hiredate: null.String{sql.NullString{
			String: "1981-06-09T00:00:00Z",
			Valid:  true,
		}},
		Sal: null.Int{sql.NullInt64{
			Int64: 99,
			Valid: true,
		}},
		Comm: null.Int{sql.NullInt64{
			Int64: 99,
			Valid: true,
		}},
		Deptno: null.Int{sql.NullInt64{
			Int64: 0,
			Valid: true,
		}},
	}

	var dept = model.Dept{
		Deptno: 0,
		Dname:  "Dname - insertBenchmark",
		Loc: null.String{sql.NullString{
			String: "Loc - insertBenchmark",
			Valid:  true,
		}},
		Emps: nil,
	}

	// готовим структуру для тестирования
	tests := struct {
		name string
		p    *Service
		args []*model.Dept
		//		want    *model.Dept
		//		wantErr bool
	}{
		name: "test",
		p:    testDB,
		args: nil,
	}

	b.ResetTimer()
	b.SetBytes(200000)
	//b.SetParallelism(2)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() { // The loop body is executed b.N times total across all goroutines
			b.StopTimer()

			var depnoPK model.DeptPK
			var deptOut model.Dept
			var empnoPK model.EmpPK
			var deptNew = dept
			deptNew.Emps = make([]*model.Emp, 10)

			// считываем последовательность для dept PK
			err := tests.p.db.DB.Get(&depnoPK, "select nextval('dept_deptno_seq') deptno")
			if err != nil {
				b.Errorf("\n PgDb.CreateDept() - error tests.p.Db.Get(&depnoPK, select nextval('dept_deptno_seq')) %v", fmt.Sprintf("%+v", err))
				return
			}

			// Подменяем PK в тестовых данных dept
			deptNew.Deptno = depnoPK.Deptno

			// Подменяем PK в тестовых данных emp
			for i := range deptNew.Emps {
				// создаем новую переменную
				var e = emp
				// записываем номер dept
				e.Deptno.Int64 = int64(depnoPK.Deptno)
				e.Deptno.Valid = true
				// считываем последовательность для emp PK
				err := tests.p.db.DB.Get(&empnoPK, "select nextval('dept_deptno_seq') empno")
				if err != nil {
					b.Errorf("\n PgDb.CreateDept() - error tests.p.Db.Get(&depnoPK, select nextval('dept_deptno_seq')) %v", fmt.Sprintf("%+v", err))
					return
				}
				// записываем номер emp
				e.Empno = empnoPK.Empno
				deptNew.Emps[i] = &e
			}

			b.StartTimer()

			err = tests.p.CreateDept(ctx, &deptNew, &deptOut)
			if err != nil {
				b.Errorf("\n PgDb.CreateDept() - error tests.p.CreateDept(&deptNew), param: %v/n %v", deptNew, fmt.Sprintf("%+v", err))
				return
			}
		}
	})

}
