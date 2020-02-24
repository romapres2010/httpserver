Типовой обработчик входящего HTTP запроса выполняет следующие действия:

- Логирование входящего HTTP запроса
- Проверка на допустимость HTTP метода (формирует ошибку http.StatusMethodNotAllowed)
- Проверка режима аутентификации (basic, MS AD, JSON web token)
- Выполнение аутентификации (basic, MS AD)
- Проверка валидности JSON web token (при необходимости)
- Считывание тела (body) входящего запроса
- Считывание заголовка (header) входящего запроса
- **Собственно обработка запроса и формирование ответа**
- Установка HSTS Strict-Transport-Security
- Установка Content-Type для исходящего ответа (response)
- Логирование исходящего HTTP ответа
- Запись заголовка (header) исходящего ответа
- Запись тела исходящего ответа
- Обработка и логирование ошибок
- Обработка defer recovery для восстановления после возможной panic

Большинство этих действий являются типовыми и не зависят от типа запроса и его обработки.  
Повторять их в каждом HTTP обработчике крайне неэффективно.  
Даже если вынести весь код в отдельные подфункции, все равно получается примерно по 80-100 строк кода на каждый HTTP обработчик без учета собственно **обработки запроса и формирования ответа**. 

Ниже описан используемый мной подход, по упрощению написание HTTP обработчиков на Golang без использования кодогенераторов и сторонних библиотек.  
<cut />

Этот подход был скомпонован из различных источников и рекомендаций в интернете.
У любого разработчика, естественно, есть свои устоявшиеся приемы и наработки - буду рад, если кто-то поделится своими подходами. 

## Подход к организации HTTP обработчиков

На следующем рисунке показана упрощенная UML диаграмма выполнения HTTP обработчика на примере некоего EchoHandler. 

Идея, заложенная в предлагаемый подход, достаточно простая:
- На верхнем уровне необходимо внедрить defer фукнцию для восстановления после паники. На UML диаграмме - это анонимная функция RecoverWrap.func1, показаная красным цветом.
- Весь типовой код необходимо вынести в отдельный обработчик. Этот обработчик встраивается в наш HTTP handler. На UML диаграмме это функция Process - показана синим цветом.
- Собственно код функциональной обработки запроса и формирования ответа вынесен в анонимную функцию в нашем HTTP handler. На UML диаграмме это функция EchoHandler.func1 - показана зеленым цветом. 

![http_handler](https://raw.githubusercontent.com/romapres2010/httpserver/master/img/http_handler.png)

## Пример кода

Пример кода ниже взят из моего реального проекта, поэтому в нем присутствуют вставки с кастомной обработкой ошибок и логированием. 

Примеры кода приведены для HTTP обработчика EchoHandler, который "зеркалит" вход на выход.

При регистрации обработчика в роутере, регистрируется не собственно обработчик EchoHandler, а анонимная функция обработки паники (она возвращается функцией RecoverWrap), которая уже вызывает наш обработчик EchoHandler.
``` go
router.HandleFunc("/echo", service.RecoverWrap(http.HandlerFunc(service.EchoHandler))).Methods("GET")
```

Текст функции RecoverWrap для регистрации анонимной функции обработки паники.  
После объявления defer func() запускается собственно наш обработчик EchoHandler.
``` go
func (s *Service) RecoverWrap(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// объявляем функцию восстановления после паники
		defer func() {
			var err error
			r := recover()
			if r != nil {
				switch t := r.(type) {
				case string:
					err = errors.New(t)
				case error:
					err = t
				default:
					err = errors.New("UNKNOWN ERROR")
				}
				// формируем текст ошибки для логирования
				myerr := myerror.New("8888", fmt.Sprintf("UNKNOWN ERROR - recover from panic \n %+v", err.Error()), "RecoverWrap", "")
				// кастомное логирование ошибки
				s.LogError(myerr, w, http.StatusInternalServerError, 0)
			}
		}()

		// вызываем обработчик
		if handlerFunc != nil {
			handlerFunc(w, r)
		}
	})
}
```

Собственно, код обработчика EchoHandler. В общем случае, он запускает функцию типовой обработки HTTP запроса Process и передается ей в качестве параметра анонимную функцию обработки. 
``` go
func (s *Service) EchoHandler(w http.ResponseWriter, r *http.Request) {
    // Запускаем универсальный обработчик HTTP запросов 
	s.Process("POST", w, r, func(requestBuf []byte, reqID uint64) ([]byte, Header, int, error) {
		header := Header{} // заголовок ответа

        // Считаем параметры из заголовка входящего запроса и поместим их в заголовок ответа
		for key := range r.Header {
			header[key] = r.Header.Get(key)
		}
        // возвращаем буфер запроса в качестве ответа, заголовок ответа и статус 
		return requestBuf, header, http.StatusOK, nil
	})
}
```

Функция типовой обработки HTTP запроса Process. Входные параметры функции:

- method string - HTTP метод обработчика используется для проверки входящего HTTP запроса
- w http.ResponseWriter, r *http.Request - стандартные переменные для обработчика
- fn func(requestBuf []byte, reqID uint64) ([]byte, Header, int, error) - собственно функция обработки, на вход она принимает буфер входящего запроса и уникальный номер HTTP request (для целей логирования), возвращает подготовленный буфер исходящего ответа, header исходящего ответа, HTTP статус и ошибку.   

``` go
func (s *Service) Process(method string, w http.ResponseWriter, r *http.Request, fn func(requestBuf []byte, reqID uint64) ([]byte, Header, int, error)) {
	var err error
	var reqID uint64 // уникальный номер Request

	// логируем входящий HTTP запрос, одновременно получаем ID Request
	if s.logger != nil {
		reqID, _ = s.logger.LogHTTPInRequest(s.сtx, r)
	}

	// проверим разрешенный метод
	if r.Method != method {
		myerr := myerror.New("8000", fmt.Sprintf("Not allowed method '%s', reqID '%v'", r.Method, reqID), "", "")
		s.LogError(myerr, w, http.StatusMethodNotAllowed, reqID)
		return
	}

	// Если включен режим аутентификации без использования JWT токена, то проверять пользователя и пароль каждый раз
	if (s.cfg.AuthType == "INTERNAL" || s.cfg.AuthType == "MSAD") && !s.cfg.UseJWT {
		if _, err = s._checkBasicAuthentication(r); err != nil {
			s.LogError(err, w, http.StatusUnauthorized, reqID)
			return
		}
	}

	// Если используем JWT - проверим токен
	if s.cfg.UseJWT {
		if _, err = s._checkJWTFromCookie(r); err != nil {
			s.LogError(err, w, http.StatusUnauthorized, reqID)
			return
		}
	}

	// считаем тело запроса
	requestBuf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		myerr := myerror.WithCause("8001", fmt.Sprintf("Failed to read HTTP body, reqID '%v'", reqID), "ioutil.ReadAll()", "", "", err.Error())
		s.LogError(myerr, w, http.StatusInternalServerError, reqID)
		return
	}

	// вызываем обработчик
	responseBuf, header, status, err := fn(requestBuf, reqID)
	if err != nil {
		s.LogError(err, w, status, reqID)
		return
	}

	// use HSTS Strict-Transport-Security
	if s.cfg.UseHSTS {
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	}

	// Content-Type - требуемый тип контента в ответе
	responseContentType := r.Header.Get("Content-Type-Response")
	// Если не задан Content-Type-Response то берем его из запроса
	if responseContentType == "" {
		responseContentType = r.Header.Get("Content-Type")
	}
	header["Content-Type"] = responseContentType

	// Логируем исходящий ответ в файл
	if s.logger != nil {
		s.logger.LogHTTPInResponse(s.сtx, header, responseBuf, status, reqID)
	}

	// запишем ответ в заголовок
	if header != nil {
		for key, h := range header {
			w.Header().Set(key, h)
		}
	}

	// запишем HTTP статус
	w.WriteHeader(status)

	// запишем буфер в ответ
	if responseBuf != nil {
		respWrittenLen, err := w.Write(responseBuf)
		if err != nil {
			myerr := myerror.WithCause("8002", fmt.Sprintf("Failed to write HTTP repsonse, reqID '%v'", reqID), "http.Write()", "", "", err.Error())
			s.LogError(myerr, w, http.StatusInternalServerError, reqID)
			return
		}
	}
}
```