package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	myerror "github.com/romapres2010/httpserver/error"
	mylog "github.com/romapres2010/httpserver/log"
	auth "gopkg.in/korylprince/go-ad-auth.v2"
	//_ "gopkg.in/ldap.v3"
	//_ "gopkg.in/asn1-ber.v1"
	//_ "golang.org/x/text"
)

// Claims a struct that will be encoded to a JWT.
// We add jwt.StandardClaims as an embedded type, to provide fields like expiry time
type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

// _checkBasicAuthentication get HTTP Basic Authentication and check
// =====================================================================
func (h *Handler) _checkBasicAuthentication(r *http.Request) (string, error) {
	cnt := "_checkBasicAuthentication" // Имя текущего метода для логирования

	var username, password string // HTTP Basic Authentication
	var ok bool

	// Считаем из заголовка HTTP Basic Authentication
	if username, password, ok = r.BasicAuth(); !ok {
		myerr := myerror.New("8004", "Header 'Authorization' is not set", "r.BasicAuth()", "")
		return "", myerr
	}
	mylog.PrintfDebugStd(fmt.Sprintf("Get Authorization header, username='%s'", username))

	// В режиме "INTERNAL" сравнимаем пользователя пароль с тем что был передан при старте адаптера
	if h.cfg.AuthType == "INTERNAL" {
		if h.cfg.HTTPUserID != username || h.cfg.HTTPUserPwd != password {
			myerr := myerror.New("8010", "Invalid user or password", cnt, fmt.Sprintf("username='%s'", username))
			return "", myerr
		}
		mylog.PrintfInfoStd(fmt.Sprintf("Success Internal Authentication, username='%s'", username))

	} else if h.cfg.AuthType == "MSAD" {
		config := &auth.Config{
			Server:   h.cfg.MSADServer,
			Port:     h.cfg.MSADPort,
			BaseDN:   h.cfg.MSADBaseDN,
			Security: auth.SecurityType(h.cfg.MSADSecurity),
		}

		status, err := auth.Authenticate(config, username, password)

		if err != nil {
			myerr := myerror.WithCause("8011", fmt.Sprintf("Error MS Active Directory Authentication"), "auth.Authenticate()", fmt.Sprintf("Server='%s:%v', BaseDN='%s', Security='%v', username='%s'", h.cfg.MSADServer, h.cfg.MSADPort, h.cfg.MSADBaseDN, h.cfg.MSADSecurity, username), "", err.Error())
			return "", myerr
		}

		if !status {
			myerr := myerror.New("8010", "Invalid user or password", "auth.Authenticate()", fmt.Sprintf("Server='%s:%v', BaseDN='%s', Security='%v', username='%s'", h.cfg.MSADServer, h.cfg.MSADPort, h.cfg.MSADBaseDN, h.cfg.MSADSecurity, username))
			return "", myerr
		}
		mylog.PrintfInfoStd(fmt.Sprintf("Success MS Active Directory Authentication, username='%s'", username))
	}

	return username, nil
}

// _checkJWTFromCookie load JWT check from Cookie and check it
// =====================================================================
func (h *Handler) _checkJWTFromCookie(r *http.Request) (*Claims, error) {

	// We can obtain the session token from the requests cookies, which come with every request
	c, err := r.Cookie("token")
	if err != nil {
		if err == http.ErrNoCookie {
			myerr := myerror.WithCause("8005", "JWT token does not present in Cookie. You have to authorize.", "r.Cookie('token')", "", "", err.Error())
			return nil, myerr
		}
		myerr := myerror.WithCause("8006", "Failed to read HTTP request header", "r.Cookie('token')", "", "", err.Error())
		return nil, myerr
	}

	return _checkJWT(c.Value, h.cfg.JwtKey)
}

// _checkJWT check JWT
// =====================================================================
func _checkJWT(tknStr string, jwtKey []byte) (*Claims, error) {

	// Initialize a new instance of `Claims`
	claims := &Claims{}

	// Parse the JWT string and store the result in `claims`.
	// Note that we are passing the key in this method as well. This method will return an error
	// if the token is invalid (if it has expired according to the expiry time we set on sign in),
	// or if the signature does not match
	tkn, err := jwt.ParseWithClaims(tknStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			myerr := myerror.WithCause("8007", "JWT token signature is invalid", "jwt.ParseWithClaims()", "", "", err.Error())
			return nil, myerr
		}
		myerr := myerror.WithCause("8008", "JWT token expired or invalid. You have to authorize.", "jwt.ParseWithClaims()", "", "", err.Error())
		return nil, myerr
	}
	if !tkn.Valid {
		myerr := myerror.WithCause("8009", "JWT token expired or invalid. You have to authorize.", "jwt.ParseWithClaims()", "", "", "")
		return nil, myerr
	}

	return claims, nil
}

// _createJWTSetCookie create new JWT and set it into Cookie
// =====================================================================
func (h *Handler) _createJWTSetCookie(claims *Claims, w http.ResponseWriter) (*http.Cookie, error) {
	var tokenString string
	var err error

	// Declare the expiration time of the token
	var expirationTime *time.Time

	// в настройках JWTExpiresAt больше 0 установим expiry time
	if h.cfg.JWTExpiresAt > 0 {
		t := time.Now().Add(time.Duration(h.cfg.JWTExpiresAt * int(time.Second)))
		expirationTime = &t
	} else {
		expirationTime = nil
	}

	// создадим новый токен
	if tokenString, err = _createJWT(claims, expirationTime, h.cfg.JwtKey); err != nil {
		return nil, err
	}

	// подготовим Cookie
	cookie := http.Cookie{
		Name:  "token",
		Value: tokenString,
	}

	// в настройках JWTExpiresAt больше 0
	if h.cfg.JWTExpiresAt > 0 {
		// Finally, we set the client cookie for "token" as the JWT we just generated
		// we also set an expiry time which is the same as the token itself
		cookie.Expires = *expirationTime
	} else {
		cookie.MaxAge = 0 // без ограничения времени жизни
	}

	http.SetCookie(w, &cookie)

	return &cookie, nil
}

// _createJWT create new JWT
// =====================================================================
func _createJWT(claims *Claims, expirationTime *time.Time, jwtKey []byte) (string, error) {
	// в настройках JWTExpiresAt больше 0 установим expiry time
	if expirationTime != nil {
		claims.StandardClaims.ExpiresAt = expirationTime.Unix()
	}

	mylog.PrintfDebugStd(fmt.Sprintf("JWT claims '%+v'", claims))

	// Declare the token with the algorithm used for signing, and the claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Create the JWT string
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}

	mylog.PrintfDebugStd(fmt.Sprintf("JWT tokenString '%s'", tokenString))

	return tokenString, nil
}

// SinginHandler handle authantification and creating JWT
// =====================================================================
func (h *Handler) SinginHandler(w http.ResponseWriter, r *http.Request) {
	mylog.PrintfDebugStd("--------------------------------------------------------------------------")

	var username string // HTTP Basic Authentication
	var err error
	var cookie *http.Cookie

	// запросим ID следующего Request
	reqID := h.logger.GetNextReqID()

	// Если включен режим аутентификации
	if h.cfg.AuthType == "INTERNAL" || h.cfg.AuthType == "MSAD" {
		if username, err = h._checkBasicAuthentication(r); err != nil {
			h.LogError(err, w, http.StatusUnauthorized, reqID)
			return
		}
	}

	// Если включен режим JSON web token (JWT)
	// Технически можно выдать JWT без аутентификации - для контроля сесии, по практический смысл ограничен
	if h.cfg.UseJWT {

		// Create the JWT claims, which includes the username
		claims := &Claims{
			Username:       username,
			StandardClaims: jwt.StandardClaims{},
		}

		// создадим новый токен и запищем его в Cookie
		mylog.PrintfDebugStd(fmt.Sprintf("Create new JSON web token"))
		if cookie, err = h._createJWTSetCookie(claims, w); err != nil {
			h.LogError(err, w, http.StatusInternalServerError, reqID)
			return
		}

		mylog.PrintfDebugStd(fmt.Sprintf("Set HTTP Cookie '%+v'", cookie))

	} else {
		mylog.PrintfDebugStd(fmt.Sprintf("JWT is of. Nothing to do"))
	}

	mylog.PrintfDebugStd("SUCCESS")
	mylog.PrintfDebugStd("--------------------------------------------------------------------------")
}

// JWTRefreshHandler handle renew JWT
// =====================================================================
func (h *Handler) JWTRefreshHandler(w http.ResponseWriter, r *http.Request) {
	mylog.PrintfDebugStd("--------------------------------------------------------------------------")

	var err error
	var claims *Claims
	var cookie *http.Cookie

	// запросим ID следующего Request
	reqID := h.logger.GetNextReqID()

	// Если включен режим JSON web token (JWT)
	// Если h.cfg.JWTExpiresAt = 0, то токен вечный - незачем обновлять
	if h.cfg.UseJWT && h.cfg.JWTExpiresAt > 0 {
		// проверим текущий JWT
		mylog.PrintfDebugStd(fmt.Sprintf("JWT is on. Check JSON web token"))
		if claims, err = h._checkJWTFromCookie(r); err != nil {
			h.LogError(err, w, http.StatusUnauthorized, reqID)
			return
		}

		// создадим новый токен и запищем его в Cookie
		mylog.PrintfDebugStd(fmt.Sprintf("JWT is valid. Create new JSON web token"))
		if cookie, err = h._createJWTSetCookie(claims, w); err != nil {
			h.LogError(err, w, http.StatusInternalServerError, reqID)
			return
		}

		mylog.PrintfDebugStd(fmt.Sprintf("Set HTTP Cookie '%+v'", cookie))

	} else {
		mylog.PrintfDebugStd(fmt.Sprintf("!(h.cfg.UseJWT && h.cfg.JWTExpiresAt > 0). Nothing to do, useJWT='%t', JWTExpiresAt='%v'", h.cfg.UseJWT, h.cfg.JWTExpiresAt))
	}

	mylog.PrintfDebugStd("SUCCESS")
	mylog.PrintfDebugStd("--------------------------------------------------------------------------")
}
