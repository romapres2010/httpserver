package json

import (
	"context"

	myerror "github.com/romapres2010/httpserver/error"
	mylog "github.com/romapres2010/httpserver/log"
	"github.com/romapres2010/httpserver/model"
)

// Config конфигурационные настройки
type Config struct {
}

// Service represent a JSON service
type Service struct {
	ctx    context.Context    // корневой контекст при инициации сервиса
	cancel context.CancelFunc // функция закрытия глобального контекста
	cfg    *Config            // конфигурационные параметры
	errCh  chan<- error       // канал ошибок
	stopCh chan struct{}      // канал подтверждения об успешном закрытии сервиса

	// вложенные сервисы
	empService  model.EmpService
	deptService model.DeptService
}

// New returns a new Service
func New(ctx context.Context, errCh chan<- error, cfg *Config, empService model.EmpService, deptService model.DeptService) (*Service, error) {
	//var err error

	mylog.PrintfInfoMsg("Creating new JSON service")

	{ // входные проверки
		if cfg == nil {
			return nil, myerror.New("6030", "Empty JSON service config").PrintfInfo()
		}
		if empService == nil {
			return nil, myerror.New("6030", "Empty EmpService service").PrintfInfo()
		}
		if deptService == nil {
			return nil, myerror.New("6030", "Empty DeptService service").PrintfInfo()
		}
	} // входные проверки

	// Создаем новый сервис
	service := &Service{
		cfg:         cfg,
		errCh:       errCh,
		stopCh:      make(chan struct{}, 1), // канал подтверждения об успешном закрытии сервиса
		empService:  empService,
		deptService: deptService,
	}

	// создаем контекст с отменой
	if ctx == nil {
		service.ctx, service.cancel = context.WithCancel(context.Background())
	} else {
		service.ctx, service.cancel = context.WithCancel(ctx)
	}

	mylog.PrintfInfoMsg("JSON service is created")
	return service, nil
}

// Shutdown shutting down service
func (s *Service) Shutdown() (myerr error) {
	mylog.PrintfInfoMsg("Shutdowning JSON service")

	defer s.cancel() // закрываем контекст

	// ...

	// Print statistics about bytes pool
	model.PrintModelPoolStats()

	mylog.PrintfInfoMsg("JSON service shutdown successfuly")
	return
}
