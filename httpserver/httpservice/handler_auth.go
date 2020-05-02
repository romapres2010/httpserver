package httpservice

import (
	"net/http"

	"github.com/dgrijalva/jwt-go"
	myerror "github.com/romapres2010/httpserver/error"
	myjwt "github.com/romapres2010/httpserver/jwt"
	mylog "github.com/romapres2010/httpserver/log"
	auth "gopkg.in/korylprince/go-ad-auth.v2"
)

// checkAuthentication chek HTTP Basic Authentication or MS AD Authentication
func (s *Service) checkAuthentication(username, password string) error {

	// В режиме "INTERNAL" сравнимаем пользователя пароль с тем что был передан при старте адаптера
	if s.cfg.AuthType == "INTERNAL" {
		if s.cfg.HTTPUserID != username || s.cfg.HTTPUserPwd != password {
			return myerror.New("8010", "Internal authentication - invalid user or password: username", username).PrintfInfo()
		}
		mylog.PrintfInfoMsg("Success Internal Authentication: username", username)

	} else if s.cfg.AuthType == "MSAD" {
		config := &auth.Config{
			Server:   s.cfg.MSADServer,
			Port:     s.cfg.MSADPort,
			BaseDN:   s.cfg.MSADBaseDN,
			Security: auth.SecurityType(s.cfg.MSADSecurity),
		}

		status, err := auth.Authenticate(config, username, password)

		if err != nil {
			return myerror.WithCause("8011", "Error MS AD Authentication: Server, Port, BaseDN, Security, username", err, s.cfg.MSADServer, s.cfg.MSADPort, s.cfg.MSADBaseDN, s.cfg.MSADSecurity, username).PrintfInfo()
		}

		if !status {
			return myerror.New("8010", "MS AD authentication - invalid user or password: Server, Port, BaseDN, Security, username", s.cfg.MSADServer, s.cfg.MSADPort, s.cfg.MSADBaseDN, s.cfg.MSADSecurity, username).PrintfInfo()
		}

		mylog.PrintfInfoMsg("Success MS AD Authentication: username", username)
	} else {
		return myerror.New("8010", "Incorrect authentication type").PrintfInfo()
	}

	return nil
}

// SinginHandler handle authantification and creating JWT
func (s *Service) SinginHandler(w http.ResponseWriter, r *http.Request) {
	mylog.PrintfDebugMsg("START   ==================================================================================")

	// Получить уникальный номер HTTP запроса
	reqID := GetNextRequestID()

	// Считаем из заголовка HTTP Basic Authentication
	username, password, ok := r.BasicAuth()
	if !ok {
		myerr := myerror.New("8004", "Header 'Authorization' is not set: reqID", reqID).PrintfInfo()
		s.processError(myerr, w, http.StatusUnauthorized, reqID)
		return
	}
	mylog.PrintfDebugMsg("Get Authorization header: reqID, username", reqID, username)

	// Выполняем аутентификацию
	if myerr := s.checkAuthentication(username, password); myerr != nil {
		mylog.PrintfErrorInfo(myerr)
		s.processError(myerr, w, http.StatusUnauthorized, reqID)
		return
	}

	// Включен режим JSON web token (JWT)
	if s.cfg.UseJWT {

		// Create the JWT claims with username
		claims := &myjwt.Claims{
			Username:       username,
			StandardClaims: jwt.StandardClaims{},
		}

		// создадим новый токен и запищем его в Cookie
		mylog.PrintfDebugMsg("Create new JSON web token: reqID", reqID)
		cookie, myerr := myjwt.CreateJWTCookie(claims, s.cfg.JWTExpiresAt, s.cfg.JwtKey)
		if myerr != nil {
			mylog.PrintfErrorInfo(myerr)
			s.processError(myerr, w, http.StatusInternalServerError, reqID)
			return
		}

		// set the client cookie for "token" as the JWT
		http.SetCookie(w, cookie)

		mylog.PrintfDebugMsg("Set HTTP Cookie: reqID, cookie", reqID, cookie)
	} else {
		mylog.PrintfDebugMsg("JWT is of. Nothing to do: reqID", reqID)
	}

	mylog.PrintfDebugMsg("SUCCESS ==================================================================================")
}

// JWTRefreshHandler handle renew JWT
func (s *Service) JWTRefreshHandler(w http.ResponseWriter, r *http.Request) {
	mylog.PrintfDebugMsg("START   ==================================================================================")

	// Получить уникальный номер HTTP запроса
	reqID := GetNextRequestID()

	// Если включен режим JSON web token (JWT)
	if s.cfg.UseJWT {
		// проверим текущий JWT
		mylog.PrintfDebugMsg("JWT is on. Check JSON web token: reqID", reqID)

		// Считаем token из requests cookies
		cookie, err := r.Cookie("token")
		if err != nil {
			myerr := myerror.WithCause("8005", "JWT token does not present in Cookie. You have to authorize first.", err).PrintfInfo()
			s.processError(myerr, w, http.StatusUnauthorized, reqID) // расширенное логирование ошибки в контексте HTTP
			return
		}

		// Проверим JWT в token
		claims, myerr := myjwt.CheckJWT(cookie.Value, s.cfg.JwtKey)
		if myerr != nil {
			mylog.PrintfErrorInfo(myerr)
			s.processError(myerr, w, http.StatusUnauthorized, reqID) // расширенное логирование ошибки в контексте HTTP
			return
		}

		// создадим новый токен и запищем его в Cookie
		mylog.PrintfDebugMsg("JWT is valid. Create new JSON web token: reqID", reqID)
		if cookie, myerr = myjwt.CreateJWTCookie(claims, s.cfg.JWTExpiresAt, s.cfg.JwtKey); myerr != nil {
			mylog.PrintfErrorInfo(myerr)
			s.processError(myerr, w, http.StatusInternalServerError, reqID)
			return
		}

		// set the client cookie for "token" as the JWT
		http.SetCookie(w, cookie)

		mylog.PrintfDebugMsg("Set HTTP Cookie: reqID, cookie", reqID, cookie)

	} else {
		mylog.PrintfDebugMsg("JWT is of. Nothing to do: reqID", reqID)
	}

	mylog.PrintfDebugMsg("SUCCESS ==================================================================================")
}
