package daemon

import (
	"crypto/tls"
	"fmt"
	"os"
	"strconv"
	"strings"

	myerror "github.com/romapres2010/httpserver/error"
	myhttp "github.com/romapres2010/httpserver/http"
	"github.com/romapres2010/httpserver/http/handler"
	httplog "github.com/romapres2010/httpserver/http/httplog"
	mylog "github.com/romapres2010/httpserver/log"
	"github.com/sasbury/mini"
	auth "gopkg.in/korylprince/go-ad-auth.v2"
)

// _loadConfigFile load confiuration file
// =====================================================================
func _loadConfigFile(fileName string) (*mini.Config, error) {
	cnt := "_loadConfigFile" // имя текущего метода для логирования

	var err error

	// считаем конфигурацию из внешнего файла
	if fileName == "" {
		errM := fmt.Sprintf("Config file name is null")
		mylog.PrintfErrorStd(errM)
		return nil, myerror.New("6013", errM, cnt, "")
	}
	mylog.PrintfInfoStd(fmt.Sprintf("Loading HTTP Server config from file '%s'", fileName))

	// Считать информацию о файле или каталоге
	_, err = os.Stat(fileName)
	// Если файл не существует
	if os.IsNotExist(err) {
		errM := fmt.Sprintf("Config file '%s' does not exist", fileName)
		mylog.PrintfErrorStd(errM)
		return nil, myerror.New("5003", errM, cnt, "")
	}

	// Считать конфигурацию из файла
	config, err := mini.LoadConfiguration(fileName)
	if err != nil {
		errM := fmt.Sprintf("Error load config file '%s'", fileName)
		mylog.PrintfErrorStd(errM)
		return nil, myerror.WithCause("5004", errM, "mini.LoadConfiguration()", fmt.Sprintf("configFile='%s'", fileName), "", err.Error())
	}

	return config, nil
}

// _loadHTTPServerConfig load HTTP server confiuration from file
// =====================================================================
func _loadHTTPServerConfig(config *mini.Config, HTTPServerCfg *myhttp.Config) error {
	cnt := "_loadHTTPServerConfig" // имя текущего метода для логирования

	var err error

	{ // секция с основными параметрами HTTP сервера
		sectionName := "HTTP_SERVER"

		if HTTPServerCfg.ReadTimeout, err = _LoadIntFromSection(cnt, sectionName, config, "ReadTimeout", true, "60"); err != nil {
			return err
		}
		if HTTPServerCfg.WriteTimeout, err = _LoadIntFromSection(cnt, sectionName, config, "WriteTimeout", true, "60"); err != nil {
			return err
		}
		if HTTPServerCfg.IdleTimeout, err = _LoadIntFromSection(cnt, sectionName, config, "IdleTimeout", true, "60"); err != nil {
			return err
		}
		if HTTPServerCfg.MaxHeaderBytes, err = _LoadIntFromSection(cnt, sectionName, config, "MaxHeaderBytes", true, "0"); err != nil {
			return err
		}
		if HTTPServerCfg.MaxBodyBytes, err = _LoadIntFromSection(cnt, sectionName, config, "MaxBodyBytes", true, "0"); err != nil {
			return err
		}
	} // секция с основными параметрами HTTP сервера

	{ // секция с настройками TLS
		sectionName := "TLS"

		if HTTPServerCfg.UseTLS, err = _LoadBoolFromSection(cnt, sectionName, config, "UseTLS", true, "false"); err != nil {
			return err
		}

		if HTTPServerCfg.UseTLS {
			{ // параметр TLSSertFile
				if HTTPServerCfg.TLSSertFile, err = _LoadStringFromSection(cnt, sectionName, config, "TLSSertFile", true, ""); err != nil {
					return err
				} else if HTTPServerCfg.TLSSertFile != "" {
					// Считать информацию о файле или каталоге
					_, err = os.Stat(HTTPServerCfg.TLSSertFile)
					// Если файл не существует
					if os.IsNotExist(err) {
						errM := fmt.Sprintf("Sertificate file '%s' does not exist", HTTPServerCfg.TLSSertFile)
						mylog.PrintfErrorStd(errM)
						return myerror.New("5010", errM, cnt, "")
					}
				}
			}
			{ // параметр TLSKeyFile
				if HTTPServerCfg.TLSKeyFile, err = _LoadStringFromSection(cnt, sectionName, config, "TLSKeyFile", true, ""); err != nil {
					return err
				} else if HTTPServerCfg.TLSKeyFile != "" {
					// Считать информацию о файле или каталоге
					_, err = os.Stat(HTTPServerCfg.TLSKeyFile)
					// Если файл не существует
					if os.IsNotExist(err) {
						errM := fmt.Sprintf("Private key file '%s' does not exist", HTTPServerCfg.TLSKeyFile)
						mylog.PrintfErrorStd(errM)
						return myerror.New("5011", errM, cnt, "")
					}
				}
			}
			{ // параметр TLSMinVersion
				if _TLSMinVersion, err := _LoadStringFromSection(cnt, sectionName, config, "TLSMinVersion", true, "VersionSSL30"); err != nil {
					return err
				} else if _TLSMinVersion != "" {
					switch _TLSMinVersion {
					case "VersionTLS13":
						HTTPServerCfg.TLSMinVersion = tls.VersionTLS13
					case "VersionTLS12":
						HTTPServerCfg.TLSMinVersion = tls.VersionTLS12
					case "VersionTLS11":
						HTTPServerCfg.TLSMinVersion = tls.VersionTLS11
					case "VersionTLS10":
						HTTPServerCfg.TLSMinVersion = tls.VersionTLS10
					default:
						errM := fmt.Sprintf("Incorrect TLSMinVersion '%s'. Only avaliable: 'VersionTLS13', 'VersionTLS12', 'VersionTLS11', 'VersionTLS10', 'VersionSSL30'.", _TLSMinVersion)
						return myerror.New("5012", errM, cnt, "")
					}
				}
			}
			{ // параметр TLSMaxVersion
				if _TLSMaxVersion, err := _LoadStringFromSection(cnt, sectionName, config, "TLSMaxVersion", true, "VersionTLS13"); err != nil {
					return err
				} else if _TLSMaxVersion != "" {
					switch _TLSMaxVersion {
					case "VersionTLS13":
						HTTPServerCfg.TLSMaxVersion = tls.VersionTLS13
					case "VersionTLS12":
						HTTPServerCfg.TLSMaxVersion = tls.VersionTLS12
					case "VersionTLS11":
						HTTPServerCfg.TLSMaxVersion = tls.VersionTLS11
					case "VersionTLS10":
						HTTPServerCfg.TLSMaxVersion = tls.VersionTLS10
					default:
						errM := fmt.Sprintf("Incorrect TLSMaxVersion '%s'. Only avaliable: 'VersionTLS13', 'VersionTLS12', 'VersionTLS11', 'VersionTLS10', 'VersionSSL30'.", _TLSMaxVersion)
						return myerror.New("5013", errM, cnt, "")
					}
				}
			}
			{ // параметр UseHSTS
				if HTTPServerCfg.UseHSTS, err = _LoadBoolFromSection(cnt, sectionName, config, "UseHSTS", true, "false"); err != nil {
					return err
				}
			}
		}

	} // секция с настройками TLS

	//mylog.PrintfInfoStd(fmt.Sprintf("SUCCESS - Load HTTP Server config '%+v'", HTTPServerCfg))
	return nil
}

// _loadHTTPHandlerConfig load HTTP handler confiuration from file
// =====================================================================
func _loadHTTPHandlerConfig(config *mini.Config, HTTPHandlerCfg *handler.Config) error {
	cnt := "_loadHTTPHandlerConfig" // имя текущего метода для логирования

	var err error

	{ // секция JWT
		sectionName := "JWT"

		if HTTPHandlerCfg.UseJWT, err = _LoadBoolFromSection(cnt, sectionName, config, "UseJWT", true, "false"); err != nil {
			return err
		}

		if HTTPHandlerCfg.UseJWT {
			if HTTPHandlerCfg.JWTExpiresAt, err = _LoadIntFromSection(cnt, sectionName, config, "JWTExpiresAt", true, "10000"); err != nil {
				return err
			}
		}
	} // секция JWT

	{ // секция AUTHENTIFICATION
		sectionName := "AUTHENTIFICATION"

		{ // параметр AuthType
			if _AuthType, err := _LoadStringFromSection(cnt, sectionName, config, "AuthType", true, "NONE"); err != nil {
				return err
			} else if _AuthType != "" {
				switch _AuthType {
				case "NONE":
					HTTPHandlerCfg.AuthType = "NONE"
				case "INTERNAL":
					HTTPHandlerCfg.AuthType = "INTERNAL"
				case "MSAD":
					HTTPHandlerCfg.AuthType = "MSAD"
				default:
					errM := fmt.Sprintf("Incorrect AuthType '%s'. Only avaliable: 'NONE', 'INTERNAL', 'MSAD'.", _AuthType)
					return myerror.New("5015", errM, cnt, "")
				}
			}
		}

		// Проверим, что для режима утентификации INTERNAL задан пользователь и пароль
		if HTTPHandlerCfg.AuthType == "INTERNAL" {
			if HTTPHandlerCfg.HTTPUserID == "" {
				errM := fmt.Sprintf("HTTP UserId is null")
				mylog.PrintfErrorStd(errM)
				return myerror.New("6021", errM, cnt, "")
			}
			if HTTPHandlerCfg.HTTPUserPwd == "" {
				errM := fmt.Sprintf("HTTP UserId is null")
				mylog.PrintfErrorStd(errM)
				return myerror.New("6022", errM, cnt, "")
			}
		}

		// Проверим, что для режима утентификации MSAD заданы параметры подключения
		if HTTPHandlerCfg.AuthType == "MSAD" {
			if HTTPHandlerCfg.MSADServer, err = _LoadStringFromSection(cnt, sectionName, config, "MSADServer", true, ""); err != nil {
				return err
			}
			if HTTPHandlerCfg.MSADPort, err = _LoadIntFromSection(cnt, sectionName, config, "MSADPort", true, ""); err != nil {
				return err
			}
			if HTTPHandlerCfg.MSADBaseDN, err = _LoadStringFromSection(cnt, sectionName, config, "MSADBaseDN", true, ""); err != nil {
				return err
			}
			if _MSADSecurity, err := _LoadStringFromSection(cnt, sectionName, config, "MSADSecurity", true, "NONE"); err != nil {
				return err
			} else if _MSADSecurity != "" {
				switch _MSADSecurity {
				case "SecurityNone":
					HTTPHandlerCfg.MSADSecurity = int(auth.SecurityNone)
				case "SecurityTLS":
					HTTPHandlerCfg.MSADSecurity = int(auth.SecurityTLS)
				case "SecurityStartTLS":
					HTTPHandlerCfg.MSADSecurity = int(auth.SecurityStartTLS)
				default:
					errM := fmt.Sprintf("Incorrect MSADSecurity '%s'. Only avaliable: 'SecurityNone', 'SecurityTLS', 'SecurityStartTLS'.", _MSADSecurity)
					return myerror.New("5016", errM, cnt, "")
				}
			}
		}

	} // секция AUTHENTIFICATION
	/*
		{ // дополнительные настройки Event Logger
			if err = _loadEventLoggerConfig(config, &eventLogerCfg); err != nil {
				return err
			}
			HTTPHandlerCfg.EventLogerCfg = &eventLogerCfg
		} // дополнительные настройки Event Logger
	*/
	//mylog.PrintfInfoStd(fmt.Sprintf("SUCCESS - Load HTTP Handler config '%+v'", HTTPHandlerCfg))
	return nil
}

// _loadHTTPLoggerConfig load HTTP Logger confiuration from file
// =====================================================================
func _loadHTTPLoggerConfig(config *mini.Config, cfg *httplog.Config) error {
	cnt := "_loadHTTPLoggerConfig" // имя текущего метода для логирования

	var err error

	if config == nil {
		errM := fmt.Sprintf("Config is null")
		mylog.PrintfErrorStd(errM)
		return myerror.New("6013", errM, cnt, "")
	}

	{ // секция HTTP_LOGGER
		sectionName := "HTTP_LOGGER"

		if cfg.Enable, err = _LoadBoolFromSection(cnt, sectionName, config, "Enable", false, "false"); err != nil {
			return err
		}

		if cfg.Enable {
			httpLoggerType, err := _LoadStringFromSection(cnt, sectionName, config, "Type", false, "")
			if err != nil {
				return err
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

			// Если логирование в файл
			if httpLoggerType != "" {
				if cfg.FileName, err = _LoadStringFromSection(cnt, sectionName, config, "FileName", true, ""); err != nil {
					return err
				}
			}
		}

	} // секция HTTP_LOGGER

	return nil
}

// _loadIntParameter load int paparameter and log err
// =====================================================================
func _loadIntParameter(cnt string, pgcfg *mini.Config, name string, manadatory bool, defval string) (int, error) {
	strVal := pgcfg.String(name, defval)
	intVal, err := strconv.Atoi(strVal)
	// только положительные параметры
	if err != nil || (intVal < 0) {
		errM := fmt.Sprintf("Incorrect or negative number - parameter '%s', val='%v'", name, strVal)
		mylog.PrintfErrorStd(errM)
		return 0, myerror.WithCause("5005", errM, "strconv.Atoi(val)", fmt.Sprintf("parameter='%s', val='%s'", name, strVal), "", err.Error())
	}
	mylog.PrintfInfoStd(fmt.Sprintf("Load config - parameter '%s', val='%v'", name, intVal))

	return intVal, nil
}

// _loadStrParameter load str paparameter and log err
// =====================================================================
func _loadStrParameter(cnt string, pgcfg *mini.Config, name string, manadatory bool, defval string) (string, error) {
	strVal := pgcfg.String(name, defval)
	if manadatory && defval == "" && strVal == "" {
		errM := fmt.Sprintf("Missing mandatory parameter '%s'", name)
		mylog.PrintfErrorStd(errM)
		return "", myerror.New("5007", errM, cnt, fmt.Sprintf("parameter='%s'", name))
	}
	mylog.PrintfInfoStd(fmt.Sprintf("Load config - parameter '%s', val='%v'", name, strVal))

	return strVal, nil
}

// _loadIntParameter load int paparameter and log err
// =====================================================================
func _LoadIntFromSection(cnt string, sectionName string, pgcfg *mini.Config, name string, manadatory bool, defval string) (int, error) {
	strVal := pgcfg.StringFromSection(sectionName, name, defval)
	intVal, err := strconv.Atoi(strVal)
	// только положительные параметры
	if err != nil || (intVal < 0) {
		errM := fmt.Sprintf("Incorrect or negative number - parameter '%s', val='%v'", name, strVal)
		mylog.PrintfErrorStd(errM)
		return 0, myerror.WithCause("5005", errM, "strconv.Atoi(val)", fmt.Sprintf("parameter='%s', val='%s'", name, strVal), "", err.Error())
	}
	mylog.PrintfInfoStd(fmt.Sprintf("Load config - parameter '%s', val='%v'", name, intVal))

	return intVal, nil
}

// _LoadStringFromSection load str paparameter and log err
// =====================================================================
func _LoadStringFromSection(cnt string, sectionName string, pgcfg *mini.Config, name string, manadatory bool, defval string) (string, error) {
	strVal := pgcfg.StringFromSection(sectionName, name, defval)
	if manadatory && defval == "" && strVal == "" {
		errM := fmt.Sprintf("Missing mandatory parameter '%s'", name)
		mylog.PrintfErrorStd(errM)
		return "", myerror.New("5007", errM, cnt, fmt.Sprintf("parameter='%s'", name))
	}
	mylog.PrintfInfoStd(fmt.Sprintf("Load config - parameter '%s', val='%v'", name, strVal))

	return strVal, nil
}

// _LoadBoolFromSection load bool paparameter and log err
// =====================================================================
func _LoadBoolFromSection(cnt string, sectionName string, pgcfg *mini.Config, name string, manadatory bool, defval string) (bool, error) {
	strVal := pgcfg.StringFromSection(sectionName, name, defval)
	if manadatory && defval == "" && strVal == "" {
		errM := fmt.Sprintf("Missing mandatory parameter '%s'", name)
		mylog.PrintfErrorStd(errM)
		return false, myerror.New("5007", errM, cnt, fmt.Sprintf("parameter='%s'", name))
	}
	mylog.PrintfInfoStd(fmt.Sprintf("Load config - parameter '%s', val='%v'", name, strVal))

	if strVal != "" {
		switch strVal {
		case "true":
			return true, nil
		case "false":
			return false, nil
		default:
			errM := fmt.Sprintf("Incorrect parameter '%s' '%s'. Only avaliable: 'true', 'false'.", name, strVal)
			return false, myerror.New("5014", errM, cnt, "")
		}
	}
	return false, nil
}
