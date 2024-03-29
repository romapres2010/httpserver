[Новая версия репозиторий проекта](https://github.com/romapres2010/goapp)
 
 # Шаблон backend сервера на Golang - часть 1 (HTTP сервер)

Представленный ниже шаблон сервера на Golang был подготовлен для передачи знаний внутри нашей команды. Основная цель шаблона, кроме обучения - это снизить время на прототипирование небольших серверных задач на Go.

Шаблон включает:

- Передачу параметров для запуска HTTP сервера через командную строку [github.com/urfave/cli](https://github.com/urfave/cli)
- Настройка параметров сервера через конфигурационный файл [github.com/sasbury/mini](https://github.com/sasbury/mini)
- Настройка параметров TLS  HTTP сервера
- Настройка роутера и регистрация HTTP и prof-обработчиков [github.com/gorilla/mux](https://github.com/gorilla/mux)
- Настройка уровней логирования без остановки сервера [github.com/hashicorp/logutils](https://github.com/hashicorp/logutils)
- Настройка логирования HTTP трафика без остановки сервера
- Настройка логирования ошибок в HTTP response без остановки сервера
- HTTP Basic аутентификация
- MS AD аутентификация [gopkg.in/korylprince/go-ad-auth.v2](https://github.com/korylprince/go-ad-auth/tree/v2.2.0)
- JSON Web Token [github.com/dgrijalva/jwt-go](https://github.com/dgrijalva/jwt-go)
- Запуск сервера с ожиданием возврата в канал ошибок
- Использование контекста для корректной остановки сервера и связанных сервисов
- Настройка кастомной обработки ошибок [github.com/pkg/errors](https://github.com/pkg/errors)
- Настройка кастомного логирования
- Сборка с внедрением версии, даты сборки и commit

Ссылка на [репозиторий проекта](https://github.com/romapres2010/httpserver). 

В состав шаблона включено несколько HTTP обработчиков:  

- POST /[echo](https://github.com/romapres2010/httpserver/blob/master/httpserver/httpservice/handler_echo.go#L11:19) - трансляция request HTTP и body в response
- POST /[signin](https://github.com/romapres2010/httpserver/blob/master/httpserver/httpservice/handler_auth.go#L57:19) - аутентификация и получение JSON Web Token в Cookie
- POST /[refresh](https://github.com/romapres2010/httpserver/blob/master/httpserver/httpservice/handler_auth.go#L110:19) - обновление времени жизни JSON Web Token в Cookie
- POST /[httplog](https://github.com/romapres2010/httpserver/blob/master/httpserver/httpservice/handler_log.go#L14:19) - настройка логирования HTTP трафика
- POST /[httperrlog](https://github.com/romapres2010/httpserver/blob/master/httpserver/httpservice/handler_log.go#L81:19) - настройка логирования ошибок в HTTP response
- POST /[loglevel](https://github.com/romapres2010/httpserver/blob/master/httpserver/httpservice/handler_log.go#L119:19) - настройка уровней логирования DEBUG, INFO, ERROR

Подход к упрощению написания HTTP обработчиков для этого шаблона описан в статье [Упрощаем написание HTTP обработчиков на Golang](https://habr.com/ru/post/489740)
<cut />

## Содержание статьи

1. Предыстория
2. Передача параметров серверу  
 2.2. Командная строка  
 2.3. Конфигурационный файл  
3. Создание, запуск и остановка сервера  
  3.1. Создание daemon и сервисов  
  3.2. Запуск daemon и сервисов  
  3.3. Остановка daemon и сервисов
4. Обработка ошибок  
  4.1. Кастомная структура ошибки  
  4.2. Форматирование печати ошибки  
  4.3. Регистрация ошибок  
  4.4. Логирование и обработка ошибок  
5. Логирование  
  5.1. Куда логируем  
  5.2. Формат логирования  
  5.3. Как логируем  
  5.4. Дополнительное логирование HTTP трафика  
6. Аутентификация  
7. Организация кода и сборка  
  7.1. Использование go mod  
  7.2. Сборка кода  
  
## 1. Предыстория

В ходе внедрения 1С:ERP, появилась интересная задача - интеграция 1С с шиной IBM MQ.
Ключевыми требованиями в части взаимодействия с IBM MQ были:

- управление пулом подключений к IBM MQ (минимальное, максимальное, время простоя)
- автоматическое переключение на резервный узел IBM MQ при сбое основного узла
- использование транзакционного режима при работе с IBM MQ (SYNCPOINT)

Дополнительно были выдвинуты требования к обработке XML сообщений:

- нормализация (канонизация) сообщений по стандарту [RFC 3076](http://www.ietf.org/rfc/rfc3076.txt)
- использование меток целостности для верификации сообщений по кастомному алгоритму HMAC с хэш-функцией ГОСТ-34.11.94
- управление оперативным кэшем секретных ключей для вычисления HMAC

Стандартного адаптера в 1C к IBM MQ не было. Существующий [REST API к IBM MQ](https://www.ibm.com/support/knowledgecenter/en/SSFKSJ_9.1.0/com.ibm.mq.dev.doc/q130940_.htm) не подходил под требования.

Адаптер 1C к IBM MQ  был успешно разработан на Go с использованием официальной библиотеки [IBM MQ](https://github.com/ibm-messaging/mq-golang). Библиотека отличается неплохой стабильностью, что и не удивительно, так как она написана в виде "обертки" над стандартной C библиотекой. За полгода работы с ней было зафиксировано всего 2 бага с обработкой слайсов [].

Представленный в статье шаблон backend сервера, является обобщением полученного опыта.

Архитектура адаптера 1С к IBM MQ укрупнено показана на следующем рисунке.  

![REST_IBMMQ](https://raw.githubusercontent.com/romapres2010/httpserver/master/img/REST_IBMMQ_v.0.3.5.jpg)

## 2. Передача параметров серверу

### 2.1. Командная строка

Все чувствительные, с точки зрения безопасности, параметры сервера передаются через командную строку. Для этого используется библиотека [github.com/urfave/cli](https://github.com/urfave/cli). Список основных параметров:

```
   --httpconfig value, --httpcfg value    HTTP Config file name
   --listenstring value, -l value         Listen string in format <host>:<port>
   --httpuser value, --httpu value        User name for access to HTTP server
   --httppassword value, --httppwd value  User password for access to HTTP server
   --jwtkey value, --jwtk value           JSON web token secret key
   --debug value, -d value                Debug mode: DEBUG, INFO, ERROR
   --logfile value, --log value           Log file name
```

### 2.2. Конфигурационный файл

Для обработки конфигурационного файла используется библиотека [github.com/sasbury/mini](https://github.com/sasbury/mini). Список типовых параметров, включенных в шаблон:

```
[HTTP_SERVER]
ReadTimeout = 6000          // HTTP read timeout duration in sec - default 60 sec
WriteTimeout = 6000         // HTTP write timeout duration in sec - default 60 sec
IdleTimeout = 6000          // HTTP idle timeout duration in sec - default 60 sec
MaxHeaderBytes = 262144     // HTTP max header bytes - default 1 MB
MaxBodyBytes = 1048576      // HTTP max body bytes - default 0 - unlimited
UseProfile = false          // use Go profiling
ShutdownTimeout = 30        // service shutdown timeout in sec - default 30 sec

[TLS]
UseTLS = false                  // use SSL
UseHSTS = false                 // use HTTP Strict Transport Security
TLSСertFile = certs/server.pem  // TLS Certificate file name
TLSKeyFile = certs/server.key   // TLS Private key file name
TLSMinVersion = VersionTLS10    // TLS min version VersionTLS13, VersionTLS12, VersionTLS11, VersionTLS10, VersionSSL30
TLSMaxVersion = VersionTLS12    // TLS max version VersionTLS13, VersionTLS12, VersionTLS11, VersionTLS10, VersionSSL30

[JWT]
UseJWT = false                  // use JSON web token (JWT)
JWTExpiresAt = 20000            // JWT expiry time in seconds - 0 without restriction

[AUTHENTIFICATION]
AuthType = INTERNAL             // Autehtification type NONE | INTERNAL | MSAD
MSADServer = company.com        // MS Active Directory server
MSADPort = 389                  // MS Active Directory Port
MSADBaseDN = OU=, DC=, DC=      // MS Active Directory BaseDN
MSADSecurity = SecurityNone     // MS Active Directory Security: SecurityNone, SecurityTLS, SecurityStartTLS

[LOG]
HTTPLog = false                         // Log HTTP traffic
HTTPLogType = INREQ                     // HTTP trafic log mode INREQ | OUTREQ | INRESP | OUTRESP | BODY
HTTPLogFileName = ./httplog/http%s.log  // HTTP log file
HTTPErrLog = HEADER | BODY              // Log error into HTTP response header and body
```

## 3. Создание, запуск и остановка сервера

На следующем рисунке показана упрощенная UML диаграмма последовательности запуска и остановки сервера.

![http_server_run_stop](https://raw.githubusercontent.com/romapres2010/httpserver/master/img/http_server_run_stop.png)

Для координации создания, запуска и остановки сервера используется [daemon](https://github.com/romapres2010/httpserver/blob/master/daemon/daemon.go). В его задачи входит:  

- считывание конфигурационного файла
- настройка конфигурации сервисов
- создание контекста context.Context
- создание каналов ошибок для обратной связи с сервисами
- создание зависимых сервисов
- корректный запуск сервисов
- ожидание системных прерываний и/или ошибок от сервисов
- корректная остановка сервисов

### 3.1. Создание daemon и сервисов

В общем случае, сервисы создаются и настраиваются при создании [daemon](https://github.com/romapres2010/httpserver/blob/master/daemon/daemon.go).  
Если есть ошибки при создании отдельных сервисах, то daemon не создается.  
Если сервис предназначен для работы в фоне, то в daemon для него создается отдельный канал ошибок:

``` go
httpserverErrCh: make(chan error, 1), // канал ошибок HTTP сервера
```

При создание сервиса, ему передаются параметры:

- контекст daemon - используется для передачи в сервис информации о закрытии
- канал ошибок - используется для возврата из сервиса в daemon информации о критичной ошибке
- структуру с конфигурационными параметрами сервиса

Примеры задач при создании сервисов:

- Для сервиса работы с IBM MQ:  
  - проверяются входные параметры
  - создается контекст сервиса
  - делается тестовое подключение к кластеру IBM MQ, определяется какой из узлов кластера является рабочим, а какой находится в резерве
  - открывается минимальный пул подключений к IBM MQ
- Для сервиса работы с PostgreSQL:
  - проверяются входные параметры
  - создается контекст сервиса
  - делается тестовое подключение
  - парсятся, предварительно определенные, SQL команды
- Для сервиса кеширования JSON в BoltDB:
  - проверяются входные параметры
  - создается контекст сервиса
  - открывается файл BoltDB на запись
  - происходит считывание закешированных ключей и проверяются валидность кэша (данные могли поменяться в БД PostgreSQL). Эта операция может быть перенесена в отдельных фоновый процесс, чтобы сократить время старта сервера (в ходе теста на обработку BoltDB размером 150 Гбайт уходит примерно 2 минуты в 64 потока при условии, что BoltDB размещена на NVMe диске со средним временем отклика 0.03 ms).
- Для [HTTP сервера](https://github.com/romapres2010/httpserver/blob/master/httpserver/httpserver.go):
  - проверяются входные параметры
  - создается контекст сервиса
  - создается и настраивается http.server
  - создается TCP листенер
  - настраиваются параметры TLS
  - создается роутер  
  - регистрируются HTTP обработчики
  - регистрируются pprof обработчики

### 3.2. Запуск daemon и сервисов

Запуск [daemon](https://github.com/romapres2010/httpserver/blob/master/daemon/daemon.go) заключается в скоординированном запуске сервисов.  
Так как все сервисы уже были созданы ранее, то запуск сервиса - это, обычно, включение листенера или установление флага, разрешающего начать обработку.  
Для запуска сервисов в фоне, используется анонимная функция с возвратом в канал ошибок. Пример, запуска HTTP сервера

``` go
go func() { httpserverErrCh <- d.httpserver.Run() }()
```

После запуска сервисов, daemon подписывается на основные системные прерывания и переходит в режим ожидания сигналов или возврата в каналы ошибок от сервисов.  
Получении daemon ошибки от сервиса, означает, что какой-то из сервисов не может продолжить работу. Здесь логика обработки может существенно отличаться, от полной остановки (как в примере ниже) до перезапуска сбойного сервиса.  
Например, если основной сервер IBM MQ становится недоступен, то daemon пробует пересоздать сервис на резервном сервере IBM MQ и продолжить обработку.

``` go
syscalCh := make(chan os.Signal, 1) // канал системных прерываний
signal.Notify(syscalCh, syscall.SIGINT, syscall.SIGTERM)

// ожидаем прерывания или возврат в канал ошибок
select {
case s := <-syscalCh: // системное прерывание
    mylog.PrintfInfoMsg("Exiting, got signal", s)
    d.Shutdown() // останавливаем daemon
    return nil
case err := <-d.httpserverErrCh: // возврат от HTTP сервера в канал ошибок
    mylog.PrintfErrorInfo(err) // логируем ошибку
    d.Shutdown() // останавливаем daemon
    return err
}
```

В запуск сервисов, работающих в фоне, добавляется анонимная функция восстановления после паники (пример ниже). При обработке паники, ошибка возвращается в канал ошибок для уведомления daemon.

``` go
func (s *Server) Run() error {
    defer func() {
        var myerr error
        r := recover()
        if r != nil {
            msg := "Recover from panic"
            switch t := r.(type) {
            case string:
                myerr = myerror.New("8888", msg, t)
            case error:
                myerr = myerror.WithCause("8888", msg, t)
            default:
                myerr = myerror.New("8888", msg)
            }
            mylog.PrintfErrorInfo(myerr) // логируем ошибку
            s.errCh <- myerr             // передаем ошибку в канал для уведомления daemon
        }
    }()

    // Запуск сервера
}
```

### 3.3. Остановка daemon и сервисов

Остановка daemon заключается в скоординированной остановке сервисов и последующем закрытии корневого контекста.

Остановка сервисов, работающих в фоне, осуществляется по следующему сценарию:

- устанавливается таймер ожидания успешной остановки (параметр ShutdownTimeout в конфигурационном файле)
- закрывается контекст сервиса
- в задачу всех сервисов входит корректная остановка активной работы при закрытии их контекста. На примере сервиса IBM MQ это:
  - ожидание обработки текущих сообщений
  - завершение открытых транзакций
  - возвращение активных подключений в пул
  - закрытие открытых очередей
  - закрытие пула активных подключений к IBM MQ
- после успешной остановки сервис отправляет подтверждение в канал stopCh

Для корректной остановки сервисов при закрытии контекста, используется такой подход:

- в корневых циклах добавляется проверка состояния контекста. Если контекст закрыт, то очередную итерацию не начинать и освободить ресурсы
- в обработчиках, в безопасных местах, добавляется проверка на состояние контекста, если контекст закрыт, то не начинать обработку

``` go
for {
    select {
    case <-ctx.Done(): // получен сигнал закрытия контекста

        // Освободить ресурсы

        s.stopCh <- struct{}{} // отправить подтверждение об успешном закрытии
        return
    default:
        // Обработка очередной итерации
    }
}
```

Для остановки [HTTP сервера](https://github.com/romapres2010/httpserver/blob/master/httpserver/httpserver.go) использовался несколько другой подход:

``` go
// создаем новый контекст с отменой и отсрочкой ShutdownTimeout
cancelCtx, cancel := context.WithTimeout(s.ctx, time.Duration(s.cfg.ShutdownTimeout*int(time.Second)))
defer cancel()

// ожидаем закрытия активных подключений в течении ShutdownTimeout
if err := s.httpServer.Shutdown(cancelCtx); err != nil {
    return err
}

s.httpService.Shutdown() // Останавливаем служебные сервисы

// подтверждение об успешном закрытии HTTP сервера
s.stopCh <- struct{}{}
```

## 4. Обработка ошибок

### 4.1. Кастомная структура ошибки

Один из наиболее удачных пакетов для обработки ошибок [github.com/pkg/errors](https://github.com/pkg/errors).  
Первоначально использовал его, но со временем стало не хватать структурности ошибки, поэтому перешел на простой [кастомный пакет](https://github.com/romapres2010/httpserver/blob/master/error/error.go).

Структура для хранения ошибки:

``` go
type Error struct {
    ID       uint64 // уникальный номер ошибки
    Code     string // код ошибки
    Msg      string // текст ошибки
    Caller   string // файл, строка и наименование метода в месте регистрации ошибки
    Args     string // строка аргументов
    CauseErr error  // ошибка - причина
    CauseMsg string // текст ошибки - причины
    Trace    string // стек вызова в месте регистрации ошибки
}
```

Мне удобно работать с типизированными ошибками, поэтому код ошибки выделен отдельным атрибутом. Например, в адаптере 1С к IBM MQ, использовался простой 4 символьный числовой код. Например, ошибки начинающиеся с "8ххх" относились к HTTP, с "7ххх" - к IBM MQ.

Caller - файл, строка и наименование метода в месте регистрации ошибки. Удобно использовать, если нет необходимости выводить полный стек. Пример вывода:

```
httpserver.go:[209] - (*Server).Run()
```

Caller вычисляется функцией

``` go
func caller(depth int) string {
    pc := make([]uintptr, 15)
    n := runtime.Callers(depth+1, pc)
    frame, _ := runtime.CallersFrames(pc[:n]).Next()
    idxFile := strings.LastIndexByte(frame.File, '/')
    idx := strings.LastIndexByte(frame.Function, '/')
    idxName := strings.IndexByte(frame.Function[idx+1:], '.') + idx + 1

    return frame.File[idxFile+1:] + ":[" + strconv.Itoa(frame.Line) + "] - " + frame.Function[idxName+1:] + "()"
}
```

Args - отдельная строка аргументов, которые можно добавить к сообщению при регистрации ошибки. Используется для целей отладки.

CauseErr и CauseMsg - исходная ошибка и сообщение. Используется, если оборачиваем чужую ошибку в свою структуру.

Trace - стандартный трейс стека. Для его получения использовал несколько своеобразный подход. При регистрации ошибки создавал дополнительно пустую ошибку из пакета [github.com/pkg/errors](https://github.com/pkg/errors) и печатал ее с ключом '%+v'. В этом режиме она выводит стек.

``` go
fmt.Sprintf("'%+v'", pkgerr.New(""))
```

### 4.2. Форматирование печати ошибки

Для соответствия интерфейсу Error, используется вывод в сокращенном формате.

``` go
func (e *Error) Error() string {
    mes := fmt.Sprintf("ID=[%v], code=[%s], mes=[%s]", e.ID, e.Code, e.Msg)
    if e.Args != "" {
        mes = fmt.Sprintf("%s, args=[%s]", mes, e.Args)
    }
    if e.CauseMsg != "" {
        mes = fmt.Sprintf("%s, causemes=[%s]", mes, e.CauseMsg)
    }
    return mes
}
```

Пример вывода в сокращенном формате

```
ID=[1], code=[8004], mes=[Error message], args=['arg1', 'arg2', 'arg3']
```

Для расширенного форматированного вывода используются ключи

``` go
// %s    print the error code, message, arguments, and cause message.
// %v    in addition to %s, print caller
// %+v   extended format. Each Frame of the error's StackTrace will be printed in detail.
func (e *Error) Format(s fmt.State, verb rune) {
    switch verb {
    case 'v':
        fmt.Fprint(s, e.Error())
        fmt.Fprintf(s, ", caller=[%s]", e.Caller)
        if s.Flag('+') {
            fmt.Fprintf(s, ", trace=%s", e.Trace)
            return
        }
    case 's':
        fmt.Fprint(s, e.Error())
    case 'q':
        fmt.Fprint(s, e.Error())
    }
}
```

Пример вывода с ключом '%+v'

```
ID=[1], code=[8004], mes=[Error message], args=['arg1', 'arg2', 'arg3'], caller=[handler_echo.go:[31] - (*Service).EchoHandler.func1()], trace='
github.com/romapres2010/httpserver/error.New
        D:/golang/src/github.com/romapres2010/httpserver/error/error.go:72
github.com/romapres2010/httpserver/httpserver/httpservice.(*Service).EchoHandler.func1
        D:/golang/src/github.com/romapres2010/httpserver/httpserver/httpservice/handler_echo.go:31
        ...
```

Предложенный формат вывода не полностью соответствует подходу структурированного логирования. Это сделано специально для более удобного чтения лога в ходе отладки.
Если нужен боле строгий формат - достаточно поправить в одном месте метод Format.

### 4.3. Регистрация ошибок

Используются два метода для регистрации ошибок:

- Создание новой ошибки

``` go
New(code string, msg string, args ...interface{}) error
```

- Оборачивание существующей ошибки

``` go
WithCause(code string, msg string, causeErr error, args ...interface{}) error
```

Дополнительные аргументы можно либо встроить в сообщение ошибки, либо передать дополнительными параметрами в args ...interface{}.

Все ошибки от сторонних и стандартных пакетов оборачиваются в кастомную ошибку в месте возникновения. Исходная ошибка вкладывается внутрь кастомной, например:

``` go
myerr = myerror.WithCause("8001", "Failed to read HTTP body: reqID", err, reqID)
```

### 4.4. Логирование и обработка ошибок

Использовался следующий подход:

- в точке возникновения, ошибка логируется на уровне INFO без trace. В этот момент, обычно, не известно, является ли это ошибкой, или она будет успешно обработана на уровне выше
- при передаче ошибок на уровень вверх она повторно не оборачивается в WithCause() и не логируется
- в точке обработки ошибки логируется результат обработки на уровне INFO. Ошибка перестает быть ошибкой.
- если ошибка дошла необработанной до самого верхнего уровня, значит это действительно ошибка и она логируется на уровне ERROR с максимальной детальностью, включая trace.

Пример логирования при ошибке чтения тела HTTP запроса:

``` go
requestBuf, err := ioutil.ReadAll(r.Body)
if err != nil {
    myerr = myerror.WithCause("8001", "Failed to read HTTP body: reqID", err, reqID)
    mylog.PrintfErrorInfo(myerr) // стандартное сокращенное логирование ошибки
    s.processError(myerr, w, http.StatusInternalServerError, reqID) // расширенное логирование ошибки в HTTP response
    return myerr
}
```

Методом [processError](https://github.com/romapres2010/httpserver/blob/aaf4321c80c598ef6545d528fa9eb188cf6a99d5/httpserver/httpservice/httpservice.go#L240:19) дополнительно ошибка может логироваться в заголовок HTTP ответа и/или тело ответа.
Необходимость такого логирования настраивается в конфигурационном файле сервера - параметр HTTPErrLog, или динамически вызовом POST на /httperrlog.

``` 
POST /httperrlog HTTP/1.1
HTTP-Err-Log: HEADER | BODY
```

При записи многострочного текста в HTTP header нужно не забывать исключать все управляющие символы

 ``` go
// carriage return (CR, ASCII 0xd), line feed (LF, ASCII 0xa), and the zero character (NUL, ASCII 0x0)
headerReplacer := strings.NewReplacer("\x0a", " ", "\x0d", " ", "\x00", " ")

w.Header().Set("Err-Trace", headerReplacer.Replace(myerr.Trace))
```

## 5. Логирование

Для управления уровнями логирования использовался пакет [github.com/hashicorp/logutils](https://github.com/hashicorp/logutils).  
Из рекомендованного списка уровней логирования [RFC 5424 — The Syslog Protocol](https://tools.ietf.org/html/rfc5424), в шаблоне оставил только:

- debug — подробная информация для отладки
- info — полезные события, например, запуск/останов сервиса
- error — ошибки исполнения, требующие вмешательства

Кастомный [пакет логирования](https://github.com/romapres2010/httpserver/blob/master/log/log.go) получился крайне простым - всего 100 строк

Ссылка на полезную статью от Dave Cheney [Let’s talk about logging](https://dave.cheney.net/2015/11/05/lets-talk-about-logging).  
Если предложенный вариант логирования покажется слишком простым - то рекомендую посмотреть в сторону [A simple logging interface for Go](https://github.com/go-logr/logr).

### 5.1. Куда логируем

Все логирование идет через стандартный пакет "log". Но для него подменяется вывод на кастомный логер [github.com/hashicorp/logutils](https://github.com/hashicorp/logutils).

``` go
// logFilter represent a custom logger seting
var logFilter = &logutils.LevelFilter{
    Levels:   []logutils.LogLevel{"DEBUG", "INFO", "ERROR"},
    MinLevel: logutils.LogLevel("INFO"), // initial setting
    Writer:   os.Stderr,                 // initial setting
}

// InitLogger init custom logger
func InitLogger(wrt io.Writer) {
    logFilter.Writer = wrt   // custom logger
    log.SetOutput(logFilter) // set std logger to our custom
}
```

При необходимости, можно параллельно логировать в файл, для этого на уровне main создается лог файл и устанавливается MultiWriter

``` go 
// настраиваем параллельное логирование в файл
if logFileFlag != "" {
    // добавляем в имя лог файла дату и время
    logFileFlag = strings.Replace(logFileFlag, "%s", time.Now().Format("2006_01_02_150405"), 1)

    // открываем лог файл на запись в режиме APPEND
    logFile, err := os.OpenFile(logFileFlag, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
    if err != nil {
        myerr := myerror.WithCause("6020", "Error open log file: Filename", err, logFileFlag)
        mylog.PrintfErrorMsg(fmt.Sprintf("%+v", myerr))
        return myerr
    }
    if logFile != nil {
        defer logFile.Close()
    }

    wrt := io.MultiWriter(os.Stderr, logFile) // параллельно пишем в os.Stderr и файл

    mylog.InitLogger(wrt) // переопределяем стандартный логер на кастомный
} else {
    mylog.InitLogger(os.Stderr)
}
```

### 5.2. Формат логирования

Логирование в режиме INFO и DEBUG сделано немного более дружественным чем логирование ошибок. Пример ниже

```
2020/03/10 17:56:39 [INFO] - httpserver.go:[61] - New() - Creating new HTTP server
2020/03/10 17:56:39 [INFO] - httpserver.go:[141] - New() - Created new TCP listener: network = 'tcp', address ['127.0.0.1:3000']
2020/03/10 17:56:39 [INFO] - httpserver.go:[155] - New() - Handler is registered: Path, Method ['/echo', 'POST']
2020/03/10 17:56:39 [INFO] - httpserver.go:[155] - New() - Handler is registered: Path, Method ['/signin', 'POST']
2020/03/10 17:56:39 [INFO] - httpserver.go:[155] - New() - Handler is registered: Path, Method ['/refresh', 'POST']
2020/03/10 17:56:39 [INFO] - httpserver.go:[179] - New() - HTTP server is created
2020/03/10 17:56:39 [INFO] - daemon.go:[121] - New() - New daemon is created
2020/03/10 17:56:39 [INFO] - daemon.go:[128] - (*Daemon).Run() - Starting daemon
2020/03/10 17:56:39 [INFO] - daemon.go:[133] - (*Daemon).Run() - Daemon is running. For exit <CTRL-c>
2020/03/10 17:56:39 [INFO] - httpserver.go:[209] - (*Server).Run() - Starting HTTP server
2020/03/10 17:57:22 [INFO] - daemon.go:[142] - (*Daemon).Run() - Exiting, got signal ['interrupt']
2020/03/10 17:57:22 [INFO] - daemon.go:[155] - (*Daemon).Shutdown() - Shutting down daemon
2020/03/10 17:57:22 [INFO] - httpserver.go:[215] - (*Server).Shutdown() - Waiting for shutdown HTTP Server: sec ['30']
2020/03/10 17:57:22 [INFO] - httpserver.go:[231] - (*Server).Shutdown() - HTTP Server shutdown successfuly
2020/03/10 17:57:22 [INFO] - daemon.go:[167] - (*Daemon).Shutdown() - Daemon is shutdown
2020/03/10 17:57:22 [INFO] - main.go:[163] - main.func1() - Server is shutdown
```

В режиме ERROR используется формат вывода из раздела выше про обработку ошибок.

### 5.3. Как логируем

Для каждого уровня логирования сделаны отдельные функции:
``` go
func PrintfInfoMsg(mes string, args ...interface{}) {
    printfMsg("[INFO]", 0, mes, args...)
}
func PrintfDebugMsg(mes string, args ...interface{}) {
    printfMsg("[DEBUG]", 0, mes, args...)
}
func PrintfErrorMsg(mes string, args ...interface{}) {
    printfMsg("[ERROR]", 0, mes, args...)
}
```

Для меня это удобнее, чем в каждой точке логирования указывать необходимый уровень.
Еще один плюс - легко централизовано отключить вывод DEBUG сообщений при переходе в продуктив.

Чтобы снизить затраты на логирование INFO и DEBUG, лучше отказаться от форматирования строк в точке вызова. Вместо

``` go
mylog.PrintfInfoMsg(fmt.Sprintf("Create new TCP listener network='tcp', address='%s'", serverCfg.ListenSpec))
```

лучше использовать

``` go
mylog.PrintfInfoMsg("Created new TCP listener: network = 'tcp', address", cfg.ListenSpec)
```

Тесты показали, что при использовании такого подхода, накладные расходы на вызовы логирования DEBUG в режиме INFO не превысили 1%.  
В этом есть существенный положительный момент - можно динамически изменять уровень с INFO на DEBUG без остановки сервера вызовом POST на /loglevel.

``` 
POST /loglevel HTTP/1.1
Log-Level-Filter: DEBUG | ERROR | INFO
```

### 5.4. Дополнительное логирование HTTP трафика

Для анализа сложных ситуаций, полезно иметь возможность логирования HTTP трафика проходящего через сервер.  
В пакете [httplog](https://github.com/romapres2010/httpserver/blob/aaf4321c80c598ef6545d528fa9eb188cf6a99d5/httpserver/httplog/httplog.go) собраны методы для логирования HTTP header и body:

- входящего запроса
- исходящего ответа
- исходящего запроса
- входящего ответа

Настройка типа логирования и лог файла может задаваться в конфигурационном файле - параметры HttpLog и HTTPLogType, или динамически вызовом POST на /httplog.

``` 
POST /httplog HTTP/1.1
Http-Log: TRUE
Http-Log-Type: INREQ | OUTREQ | INRESP | OUTRESP | BODY
```

## 6. Аутентификация

В шаблон включены два базовых способа аутентификации: HTTP Basic Authentication или MS AD Authentication.  
Для MS AD Authentication используется библиотека [gopkg.in/korylprince/go-ad-auth.v2](https://github.com/korylprince/go-ad-auth/tree/v2.2.0).  
На уровне конфигурационного файла, сервера можно задать тип аутентификации и параметры подключения к серверу MS AD.

```
[AUTHENTIFICATION]
AuthType = INTERNAL | MSAD | NONE
MSADServer = company.com
MSADPort = 389
MSADBaseDN = OU=company, DC=dc, DC=corp
MSADSecurity = SecurityNone
```

Пользователь и пароль для проверки HTTP Basic Authentication передается через командую строку при старте сервера.

Использование JSON Web Token (JWT) задается на уровне конфигурационного файла сервера. Время жизни токена задается параметром JWTExpiresAt (JWTExpiresAt=0 - время жизни не ограничено). Секретный ключ для генерации JWT передается через командую строку при старте сервера.

```
[JWT]
UseJWT = true
JWTExpiresAt = 20000
```

Для работы с JWT используется библиотека [github.com/dgrijalva/jwt-go](https://github.com/dgrijalva/jwt-go).  
Вся обработка JWT: создание, проверка, формирование cookie собрано в небольшой кастомный пакет [jwt](https://github.com/romapres2010/httpserver/blob/master/jwt/jwt.go)

Логика использования JWT следующая:

- если JWT выключен, то при каждом входящем запросе выполнять HTTP Basic Authentication или MS AD Authentication
- если JWT включен, то все запросы блокируются (StatusUnauthorized), пока не будет выполнена аутентификация и не будет получен JWT
- аутентификация и получение JWT выполняется вызовом POST на /signin
  - при успешной аутентификации формируется JSON Web Token Claim (в Claim включается имя пользователя), устанавливает время его жизни
  - из Claims формируется JSON Web Token, подписывается алгоритмом HS256 вместе с секретным ключом
  - сформированный Token помещается в http Cookie "token". Для Cookie устанавливается аналогичное Token время жизни
- при последующих запросах, из http Cookie извлекается JWT и проверяется. Если время жизни закончилось, то StatusUnauthorized
- обновление JWT выполняется вызовом POST на /refresh
- logout не предусмотрен

## 7. Организация кода и сборка

### 7.1. Использование go mod

Для организации библиотек перешел с **golang/dep/cmd/dep** на **go mod**, но для хранения библиотек по прежнему используется папка vendor в корне проекта. Это позволяет хранить "правильные" версии библиотек в своем репозитории проекта. Обновление библиотек выполняется командой.

```
go get -u ./../...
go mod vendor
```

При переходе на go mod столкнулся с проблемой, что часть сторонних библиотек не поддерживают корректно работу с модулями и загружается неправильная версия. Для таких библиотек нужно указывать в конце номер нужного commit, например:

```
go get github.com/ibm-messaging/mq-golang/ibmmq@19b946c
```

Для того, чтобы компилятор брал библиотеки из каталога vendor, нужно указать опцию

```
go build -v -mod vendor
```

### 7.2. Сборка кода

Для сборки используется простой [make](https://raw.githubusercontent.com/romapres2010/httpserver/master/cmd/httpserver/make_windows) файл с несколькими режимами (взят где-то на посторах интернета):  

- rebuild - полная пересборка 
- build - инкрементальная сборка 
- check - проверка кода с использованием github.com/golangci/golangci-lint

При сборе в исполняемый файл внедряется версия, дата сборки и commit. Эта информация выводится в лог файл - весьма полезно для разбора ошибок.  
Для реализации такого внедрения, в main добавляем переменные

``` go 
var (
    version   = "0.0.2" // номер версии, задается руками
    commit    = "unset" // номер commit
    buildTime = "unset" // дата и время сборки
)
```

При сборе в make файле, запрашиваем git о commit

```
COMMIT?=$(shell git rev-parse --short HEAD)
BUILD_TIME?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
```

При сборке привязываем эти значения к ранее объявленным переменным (опция **-ldflags "-X"**), заодно указываем операционку, под которую собираем и место хранения бинарника

```
GOOS=${GOOS} go build -v -a -mod vendor \
-ldflags "-X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}" \
-o bin/${GOOS}/${APP} 
```

В main.main в переменную записываю итоговую информацию о версии, commit и дате сборки

``` go 
app.Version = fmt.Sprintf("%s, commit '%s', build time '%s'", version, commit, buildTime)
```
