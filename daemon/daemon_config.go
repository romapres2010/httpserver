package daemon

import (
	"crypto/tls"
	"os"
	"strconv"
	"strings"

	myerror "github.com/romapres2010/httpserver/error"
	"github.com/romapres2010/httpserver/httpserver"
	"github.com/romapres2010/httpserver/httpserver/httplog"
	"github.com/romapres2010/httpserver/httpserver/httpservice"
	mylog "github.com/romapres2010/httpserver/log"
	"github.com/sasbury/mini"
	auth "gopkg.in/korylprince/go-ad-auth.v2"
)

// loadConfigFile load confiuration file
func loadConfigFile(fileName string) (*mini.Config, error) {
	var myerr error

	if fileName == "" {
		myerr = myerror.New("6013", "Config file is null")
		mylog.PrintfErrorInfo(myerr)
		return nil, myerr
	}
	mylog.PrintfInfoMsg("Loading HTTP Server config from file: FileName", fileName)

	// Считать информацию о файле
	_, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		myerr = myerror.New("5003", "Config file does not exist: FileName", fileName)
		mylog.PrintfErrorInfo(myerr)
		return nil, myerr
	}

	// Считать конфигурацию из файла
	config, err := mini.LoadConfiguration(fileName)
	if err != nil {
		myerr = myerror.WithCause("5004", "Error load config file: FileName", err, fileName)
		mylog.PrintfErrorInfo(myerr)
		return nil, myerr
	}

	return config, nil
}

// loadHTTPServerConfig load HTTP server confiuration from file
func loadHTTPServerConfig(config *mini.Config, cfg *httpserver.Config) error {
	var myerr error

	{ // секция с основными параметрами HTTP сервера
		sectionName := "HTTP_SERVER"

		if cfg.ReadTimeout, myerr = loadIntFromSection(sectionName, config, "ReadTimeout", true, "60"); myerr != nil {
			mylog.PrintfErrorInfo(myerr)
			return myerr
		}
		if cfg.WriteTimeout, myerr = loadIntFromSection(sectionName, config, "WriteTimeout", true, "60"); myerr != nil {
			mylog.PrintfErrorInfo(myerr)
			return myerr
		}
		if cfg.IdleTimeout, myerr = loadIntFromSection(sectionName, config, "IdleTimeout", true, "60"); myerr != nil {
			mylog.PrintfErrorInfo(myerr)
			return myerr
		}
		if cfg.MaxHeaderBytes, myerr = loadIntFromSection(sectionName, config, "MaxHeaderBytes", true, "0"); myerr != nil {
			mylog.PrintfErrorInfo(myerr)
			return myerr
		}
		if cfg.MaxBodyBytes, myerr = loadIntFromSection(sectionName, config, "MaxBodyBytes", true, "0"); myerr != nil {
			mylog.PrintfErrorInfo(myerr)
			return myerr
		}
		if cfg.UseProfile, myerr = loadBoolFromSection(sectionName, config, "UseProfile", true, "false"); myerr != nil {
			mylog.PrintfErrorInfo(myerr)
			return myerr
		}
		if cfg.ShutdownTimeout, myerr = loadIntFromSection(sectionName, config, "ShutdownTimeout", true, "30"); myerr != nil {
			mylog.PrintfErrorInfo(myerr)
			return myerr
		}
	} // секция с основными параметрами HTTP сервера

	{ // секция с настройками TLS
		sectionName := "TLS"

		if cfg.UseTLS, myerr = loadBoolFromSection(sectionName, config, "UseTLS", true, "false"); myerr != nil {
			mylog.PrintfErrorInfo(myerr)
			return myerr
		}

		if cfg.UseTLS {
			{ // параметр TLSCertFile
				if cfg.TLSCertFile, myerr = loadStringFromSection(sectionName, config, "TLSCertFile", true, ""); myerr != nil {
					mylog.PrintfErrorInfo(myerr)
					return myerr
				} else if cfg.TLSCertFile != "" {
					// Считать информацию о файле или каталоге
					_, err := os.Stat(cfg.TLSCertFile)
					// Если файл не существует
					if os.IsNotExist(err) {
						myerr = myerror.New("5010", "Sertificate file does not exist: FileName", cfg.TLSCertFile)
						mylog.PrintfErrorInfo(myerr)
						return myerr
					}
				}
			}
			{ // параметр TLSKeyFile
				if cfg.TLSKeyFile, myerr = loadStringFromSection(sectionName, config, "TLSKeyFile", true, ""); myerr != nil {
					mylog.PrintfErrorInfo(myerr)
					return myerr
				} else if cfg.TLSKeyFile != "" {
					// Считать информацию о файле или каталоге
					_, err := os.Stat(cfg.TLSKeyFile)
					// Если файл не существует
					if os.IsNotExist(err) {
						myerr = myerror.New("5011", "Private key file does not exist: FileName", cfg.TLSKeyFile)
						mylog.PrintfErrorInfo(myerr)
						return myerr
					}
				}
			}
			{ // параметр TLSMinVersion
				if _TLSMinVersion, myerr := loadStringFromSection(sectionName, config, "TLSMinVersion", true, "VersionSSL30"); myerr != nil {
					mylog.PrintfErrorInfo(myerr)
					return myerr
				} else if _TLSMinVersion != "" {
					switch _TLSMinVersion {
					case "VersionTLS13":
						cfg.TLSMinVersion = tls.VersionTLS13
					case "VersionTLS12":
						cfg.TLSMinVersion = tls.VersionTLS12
					case "VersionTLS11":
						cfg.TLSMinVersion = tls.VersionTLS11
					case "VersionTLS10":
						cfg.TLSMinVersion = tls.VersionTLS10
					default:
						myerr = myerror.New("5012", "Incorrect TLSMinVersion, only avaliable 'VersionTLS13', 'VersionTLS12', 'VersionTLS11', 'VersionTLS10', 'VersionSSL30'", _TLSMinVersion)
						mylog.PrintfErrorInfo(myerr)
						return myerr
					}
				}
			}
			{ // параметр TLSMaxVersion
				if _TLSMaxVersion, myerr := loadStringFromSection(sectionName, config, "TLSMaxVersion", true, "VersionTLS13"); myerr != nil {
					mylog.PrintfErrorInfo(myerr)
					return myerr
				} else if _TLSMaxVersion != "" {
					switch _TLSMaxVersion {
					case "VersionTLS13":
						cfg.TLSMaxVersion = tls.VersionTLS13
					case "VersionTLS12":
						cfg.TLSMaxVersion = tls.VersionTLS12
					case "VersionTLS11":
						cfg.TLSMaxVersion = tls.VersionTLS11
					case "VersionTLS10":
						cfg.TLSMaxVersion = tls.VersionTLS10
					default:
						myerr = myerror.New("5013", "Incorrect TLSMaxVersion, only avaliable 'VersionTLS13', 'VersionTLS12', 'VersionTLS11', 'VersionTLS10', 'VersionSSL30'", _TLSMaxVersion)
						mylog.PrintfErrorInfo(myerr)
						return myerr
					}
				}
			}
			{ // параметр UseHSTS
				if cfg.UseHSTS, myerr = loadBoolFromSection(sectionName, config, "UseHSTS", true, "false"); myerr != nil {
					mylog.PrintfErrorInfo(myerr)
					return myerr
				}
			}
		}

	} // секция с настройками TLS

	return nil
}

// loadHTTPServiceConfig load HTTP handler confiuration from file
func loadHTTPServiceConfig(config *mini.Config, cfg *httpservice.Config) error {
	var myerr error

	{ // секция JWT
		sectionName := "JWT"

		if cfg.UseJWT, myerr = loadBoolFromSection(sectionName, config, "UseJWT", true, "false"); myerr != nil {
			mylog.PrintfErrorInfo(myerr)
			return myerr
		}

		if cfg.UseJWT {
			if cfg.JWTExpiresAt, myerr = loadIntFromSection(sectionName, config, "JWTExpiresAt", true, "10000"); myerr != nil {
				mylog.PrintfErrorInfo(myerr)
				return myerr
			}
		}
	} // секция JWT

	{ // секция AUTHENTIFICATION
		sectionName := "AUTHENTIFICATION"

		{ // параметр AuthType
			if _AuthType, myerr := loadStringFromSection(sectionName, config, "AuthType", true, "NONE"); myerr != nil {
				mylog.PrintfErrorInfo(myerr)
				return myerr
			} else if _AuthType != "" {
				switch _AuthType {
				case "NONE":
					cfg.AuthType = "NONE"
				case "INTERNAL":
					cfg.AuthType = "INTERNAL"
				case "MSAD":
					cfg.AuthType = "MSAD"
				default:
					myerr = myerror.New("5015", "Incorrect AuthType, only avaliable 'NONE', 'INTERNAL', 'MSAD'", _AuthType)
					mylog.PrintfErrorInfo(myerr)
					return myerr
				}
			}
		}

		// Проверим, что для режима утентификации INTERNAL задан пользователь и пароль
		if cfg.AuthType == "INTERNAL" {
			if cfg.HTTPUserID == "" {
				myerr = myerror.New("6021", "User name for access to HTTP server is null")
				mylog.PrintfErrorInfo(myerr)
				return myerr
			}
			if cfg.HTTPUserPwd == "" {
				myerr = myerror.New("6021", "User password for access to HTTP server is null")
				mylog.PrintfErrorInfo(myerr)
				return myerr
			}
		}

		// Проверим, что для режима утентификации MSAD заданы параметры подключения
		if cfg.AuthType == "MSAD" {
			if cfg.MSADServer, myerr = loadStringFromSection(sectionName, config, "MSADServer", true, ""); myerr != nil {
				mylog.PrintfErrorInfo(myerr)
				return myerr
			}
			if cfg.MSADPort, myerr = loadIntFromSection(sectionName, config, "MSADPort", true, ""); myerr != nil {
				mylog.PrintfErrorInfo(myerr)
				return myerr
			}
			if cfg.MSADBaseDN, myerr = loadStringFromSection(sectionName, config, "MSADBaseDN", true, ""); myerr != nil {
				mylog.PrintfErrorInfo(myerr)
				return myerr
			}

			if _MSADSecurity, myerr := loadStringFromSection(sectionName, config, "MSADSecurity", true, "NONE"); myerr != nil {
				mylog.PrintfErrorInfo(myerr)
				return myerr
			} else if _MSADSecurity != "" {
				switch _MSADSecurity {
				case "SecurityNone":
					cfg.MSADSecurity = int(auth.SecurityNone)
				case "SecurityTLS":
					cfg.MSADSecurity = int(auth.SecurityTLS)
				case "SecurityStartTLS":
					cfg.MSADSecurity = int(auth.SecurityStartTLS)
				default:
					myerr = myerror.New("5016", "Incorrect MSADSecurity, only avaliable 'SecurityNone', 'SecurityTLS', 'SecurityStartTLS'", _MSADSecurity)
					mylog.PrintfErrorInfo(myerr)
					return myerr
				}
			}
		}

	} // секция AUTHENTIFICATION

	{ // секция LOG
		sectionName := "LOG"

		{
			if cfg.HTTPLog, myerr = loadBoolFromSection(sectionName, config, "HTTPLog", true, "false"); myerr != nil {
				mylog.PrintfErrorInfo(myerr)
				return myerr
			}

			if cfg.HTTPLogFileName, myerr = loadStringFromSection(sectionName, config, "HTTPLogFileName", false, ""); myerr != nil {
				mylog.PrintfErrorInfo(myerr)
				return myerr
			}
		}

		{
			httpErrLoggerType, myerr := loadStringFromSection(sectionName, config, "HTTPErrLog", false, "")
			if myerr != nil {
				mylog.PrintfErrorInfo(myerr)
				return myerr
			}

			// логировать ошибки в заголовок ответа
			if strings.Index(httpErrLoggerType, "HEADER") >= 0 {
				cfg.HTTPErrorLogHeader = true
			}

			// логировать ошибки в тело ответа
			if strings.Index(httpErrLoggerType, "BODY") >= 0 {
				cfg.HTTPErrorLogBody = true
			}
		}

	} // секция LOG

	return nil
}

// loadHTTPLoggerConfig load HTTP Logger confiuration from file
func loadHTTPLoggerConfig(config *mini.Config, cfg *httplog.Config) error {
	var myerr error

	{ // секция LOG
		sectionName := "LOG"

		if cfg.Enable, myerr = loadBoolFromSection(sectionName, config, "HTTPLog", true, "false"); myerr != nil {
			mylog.PrintfErrorInfo(myerr)
			return myerr
		}

		if cfg.Enable {
			httpLoggerType, myerr := loadStringFromSection(sectionName, config, "HTTPLogType", true, "")
			if myerr != nil {
				mylog.PrintfErrorInfo(myerr)
				return myerr
			}

			// логировать входящие запросы
			if strings.Index(httpLoggerType, "INREQ") >= 0 {
				cfg.LogInReq = true
			}

			// логировать исходящие запросы
			if strings.Index(httpLoggerType, "OUTREQ") >= 0 {
				cfg.LogOutReq = true
			}

			// логировать входящие ответы
			if strings.Index(httpLoggerType, "INRESP") >= 0 {
				cfg.LogInResp = true
			}

			// логировать исходящие ответы
			if strings.Index(httpLoggerType, "OUTRESP") >= 0 {
				cfg.LogOutResp = true
			}

			// логировать тело запроса
			if strings.Index(httpLoggerType, "BODY") >= 0 {
				cfg.LogBody = true
			}
		}

	} // секция LOG

	return nil
}

// loadIntFromSection load int paparameter and log err
func loadIntFromSection(sectionName string, pgcfg *mini.Config, name string, manadatory bool, defval string) (int, error) {
	strVal := pgcfg.StringFromSection(sectionName, name, defval)
	if manadatory && defval == "" && strVal == "" {
		myerr := myerror.New("5007", "Missing mandatory: Section, Parameter", sectionName, name)
		//mylog.PrintfErrorInfo(myerr)
		return 0, myerr
	}
	intVal, err := strconv.Atoi(strVal)
	if err != nil {
		myerr := myerror.WithCause("5005", "Incorrect integer: Section, Parameter, Value", err, sectionName, name, strVal)
		//mylog.PrintfErrorInfo(myerr)
		return 0, myerr
	}
	// только положительные параметры
	if intVal < 0 {
		myerr := myerror.New("5005", "Negative integer is not allowed: Section, Parameter, Value", sectionName, name, strVal)
		//mylog.PrintfErrorInfo(myerr)
		return 0, myerr
	}

	mylog.PrintfInfoMsg("Load config parameter: Section, Parameter, Value", sectionName, name, intVal)

	return intVal, nil
}

// loadStringFromSection load str paparameter and log err
func loadStringFromSection(sectionName string, pgcfg *mini.Config, name string, manadatory bool, defval string) (string, error) {
	strVal := pgcfg.StringFromSection(sectionName, name, defval)
	if manadatory && defval == "" && strVal == "" {
		myerr := myerror.New("5007", "Missing mandatory: Section, Parameter", sectionName, name)
		//mylog.PrintfErrorInfo(myerr)
		return "", myerr
	}
	mylog.PrintfInfoMsg("Load config parameter: Section, Parameter, Value", sectionName, name, strVal)

	return strVal, nil
}

// loadBoolFromSection load bool paparameter and log err
func loadBoolFromSection(sectionName string, pgcfg *mini.Config, name string, manadatory bool, defval string) (bool, error) {
	var boolVal bool
	strVal := pgcfg.StringFromSection(sectionName, name, defval)
	if manadatory && defval == "" && strVal == "" {
		myerr := myerror.New("5007", "Missing mandatory: Section, Parameter", sectionName, name)
		//mylog.PrintfErrorInfo(myerr)
		return false, myerr
	}

	if strVal != "" {
		switch strVal {
		case "true":
			boolVal = true
		case "false":
			boolVal = false
		default:
			myerr := myerror.New("5014", "Incorrect boolean, оnly avaliable: 'true', 'false': Section, Parameter, Value", sectionName, name, strVal)
			//mylog.PrintfErrorInfo(myerr)
			return false, myerr
		}
	}

	mylog.PrintfInfoMsg("Load config parameter: Section, Parameter, Value", sectionName, name, boolVal)

	return boolVal, nil
}
