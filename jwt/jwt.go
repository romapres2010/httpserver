package jwt

import (
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	myerror "github.com/romapres2010/httpserver/error"
	mylog "github.com/romapres2010/httpserver/log"
)

// Claims a struct that will be encoded to a JWT.
// We add jwt.StandardClaims as an embedded type, to provide fields like expiry time
type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

// CheckJWT check JWT
func CheckJWT(tknStr string, jwtKey []byte) (*Claims, error) {

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
			return nil, myerror.WithCause("8007", "JWT token signature is invalid", err).PrintfInfo()
		}
		return nil, myerror.WithCause("8008", "JWT token expired or invalid. You have to authorize.", err).PrintfInfo()
	}
	if !tkn.Valid {
		return nil, myerror.New("8009", "JWT token expired or invalid. You have to authorize.").PrintfInfo()
	}

	return claims, nil
}

// CheckJWTFromCookie load JWT check from Cookie and check it
func CheckJWTFromCookie(cookie *http.Cookie, jwtKey []byte) error {

	_, err := CheckJWT(cookie.Value, jwtKey)

	return err
}

// CreateJWT create new JWT
func CreateJWT(claims *Claims, expirationTime *time.Time, jwtKey []byte) (string, error) {
	// JWTExpiresAt больше 0 установим expiry time
	if expirationTime != nil {
		claims.StandardClaims.ExpiresAt = expirationTime.Unix()
	}

	mylog.PrintfDebugMsg("JWT: claims", claims)

	// Declare the token with the algorithm used for signing, and the claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Create the JWT string
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}

	mylog.PrintfDebugMsg("JWT: tokenString", tokenString)

	return tokenString, nil
}

// CreateJWTCookie create new JWT as Cookie
func CreateJWTCookie(claims *Claims, jwtExpiresAt int, jwtKey []byte) (*http.Cookie, error) {
	var expirationTime *time.Time

	// jwtExpiresAt > 0 установим expiry time
	if jwtExpiresAt > 0 {
		t := time.Now().Add(time.Duration(jwtExpiresAt * int(time.Second)))
		expirationTime = &t
	}

	// создадим новый токен
	tokenString, err := CreateJWT(claims, expirationTime, jwtKey)
	if err != nil {
		return nil, err
	}

	// подготовим Cookie
	cookie := http.Cookie{
		Name:  "token",
		Value: tokenString,
	}

	if jwtExpiresAt > 0 {
		// set an expiry time is the same as the token itself
		cookie.Expires = *expirationTime
	} else {
		cookie.MaxAge = 0 // без ограничения времени жизни
	}

	return &cookie, nil
}
