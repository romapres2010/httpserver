package daemon

import (
	"crypto/tls"
	"fmt"
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
	var err error

	if fileName == "" {
		errM := fmt.Sprintf("Config file name is null")
		mylog.PrintfErrorStd(errM)
		return nil, myerror.New("6013", errM, "", "")
	}
	mylog.PrintfInfoStd(fmt.Sprintf("Loading HTTP Server config from file '%s'", fileName))

	// Считать информацию о файле
	_, err = os.Stat(fileName)
	if os.IsNotExist(err) {
		errM := fmt.Sprintf("Config file '%s' does not exist", fileName)
		mylog.PrintfErrorStd(errM)
		return nil, myerror.New("5003", errM, "", "")
	}

	// Считать конфигурацию из файла
	config, err := mini.LoadConfiguration(fileName)
	if err != nil {
		errM := fmt.Sprintf("Error load config file '%s'", fileName)
		mylog.PrintfErrorStd(errM)
		return nil, myerror.WithCause("5004", errM, "mini.loadConfiguration()", fmt.Sprintf("configFile='%s'", fileName), "", err.Error())
	}

	return config, nil
}

// loadHTTPServerConfig load HTTP server confiuration from file
func loadHTTPServerConfig(config *mini.Config, HTTPServerCfg *httpserver.Config) error {
	var err error

	{ // секция с основными параметрами HTTP сервера
		sectionName := "HTTP_SERVER"

		if HTTPServerCfg.ReadTimeout, err = loadIntFromSection(sectionName, config, "ReadTimeout", true, "60"); err != nil {
			return err
		}
		if HTTPServerCfg.WriteTimeout, err = loadIntFromSection(sectionName, config, "WriteTimeout", true, "60"); err != nil {
			return err
		}
		if HTTPServerCfg.IdleTimeout, err = loadIntFromSection(sectionName, config, "IdleTimeout", true, "60"); err != nil {
			return err
		}
		if HTTPServerCfg.MaxHeaderBytes, err = loadIntFromSection(sectionName, config, "MaxHeaderBytes", true, "0"); err != nil {
			return err
		}
		if HTTPServerCfg.MaxBodyBytes, err = loadIntFromSection(sectionName, config, "MaxBodyBytes", true, "0"); err != nil {
			return err
		}
		if HTTPServerCfg.UseProfile, err = loadBoolFromSection(sectionName, config, "UseProfile", true, "false"); err != nil {
			return err
		}
	} // секция с основными параметрами HTTP сервера

	{ // секция с настройками TLS
		sectionName := "TLS"

		if HTTPServerCfg.UseTLS, err = loadBoolFromSection(sectionName, config, "UseTLS", true, "false"); err != nil {
			return err
		}

		if HTTPServerCfg.UseTLS {
			{ // параметр TLSSertFile
				if HTTPServerCfg.TLSSertFile, err = loadStringFromSection(sectionName, config, "TLSSertFile", true, ""); err != nil {
					return err
				} else if HTTPServerCfg.TLSSertFile != "" {
					// Считать информацию о файле или каталоге
					_, err = os.Stat(HTTPServerCfg.TLSSertFile)
					// Если файл не существует
					if os.IsNotExist(err) {
						errM := fmt.Sprintf("Sertificate file '%s' does not exist", HTTPServerCfg.TLSSertFile)
						mylog.PrintfErrorStd(errM)
						return myerror.New("5010", errM, "", "")
					}
				}
			}
			{ // параметр TLSKeyFile
				if HTTPServerCfg.TLSKeyFile, err = loadStringFromSection(sectionName, config, "TLSKeyFile", true, ""); err != nil {
					return err
				} else if HTTPServerCfg.TLSKeyFile != "" {
					// Считать информацию о файле или каталоге
					_, err = os.Stat(HTTPServerCfg.TLSKeyFile)
					// Если файл не существует
					if os.IsNotExist(err) {
						errM := fmt.Sprintf("Private key file '%s' does not exist", HTTPServerCfg.TLSKeyFile)
						mylog.PrintfErrorStd(errM)
						return myerror.New("5011", errM, "", "")
					}
				}
			}
			{ // параметр TLSMinVersion
				if _TLSMinVersion, err := loadStringFromSection(sectionName, config, "TLSMinVersion", true, "VersionSSL30"); err != nil {
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
						return myerror.New("5012", errM, "", "")
					}
				}
			}
			{ // параметр TLSMaxVersion
				if _TLSMaxVersion, err := loadStringFromSection(sectionName, config, "TLSMaxVersion", true, "VersionTLS13"); err != nil {
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
						return myerror.New("5013", errM, "", "")
					}
				}
			}
			{ // параметр UseHSTS
				if HTTPServerCfg.UseHSTS, err = loadBoolFromSection(sectionName, config, "UseHSTS", true, "false"); err != nil {
					return err
				}
			}
		}

	} // секция с настройками TLS

	return nil
}

// loadHTTPHandlerConfig load HTTP handler confiuration from file
func loadHTTPHandlerConfig(config *mini.Config, HTTPHandlerCfg *httpservice.Config) error {
	var err error

	{ // секция JWT
		sectionName := "JWT"

		if HTTPHandlerCfg.UseJWT, err = loadBoolFromSection(sectionName, config, "UseJWT", true, "false"); err != nil {
			return err
		}

		if HTTPHandlerCfg.UseJWT {
			if HTTPHandlerCfg.JWTExpiresAt, err = loadIntFromSection(sectionName, config, "JWTExpiresAt", true, "10000"); err != nil {
				return err
			}
		}
	} // секция JWT

	{ // секция AUTHENTIFICATION
		sectionName := "AUTHENTIFICATION"

		{ // параметр AuthType
			if _AuthType, err := loadStringFromSection(sectionName, config, "AuthType", true, "NONE"); err != nil {
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
					return myerror.New("5015", errM, "", "")
				}
			}
		}

		// Проверим, что для режима утентификации INTERNAL задан пользователь и пароль
		if HTTPHandlerCfg.AuthType == "INTERNAL" {
			if HTTPHandlerCfg.HTTPUserID == "" {
				errM := fmt.Sprintf("HTTP UserId is null")
				mylog.PrintfErrorStd(errM)
				return myerror.New("6021", errM, "", "")
			}
			if HTTPHandlerCfg.HTTPUserPwd == "" {
				errM := fmt.Sprintf("HTTP UserId is null")
				mylog.PrintfErrorStd(errM)
				return myerror.New("6022", errM, "", "")
			}
		}

		// Проверим, что для режима утентификации MSAD заданы параметры подключения
		if HTTPHandlerCfg.AuthType == "MSAD" {
			if HTTPHandlerCfg.MSADServer, err = loadStringFromSection(sectionName, config, "MSADServer", true, ""); err != nil {
				return err
			}
			if HTTPHandlerCfg.MSADPort, err = loadIntFromSection(sectionName, config, "MSADPort", true, ""); err != nil {
				return err
			}
			if HTTPHandlerCfg.MSADBaseDN, err = loadStringFromSection(sectionName, config, "MSADBaseDN", true, ""); err != nil {
				return err
			}
			if _MSADSecurity, err := loadStringFromSection(sectionName, config, "MSADSecurity", true, "NONE"); err != nil {
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
					return myerror.New("5016", errM, "", "")
				}
			}
		}

	} // секция AUTHENTIFICATION
	return nil
}

// loadHTTPLoggerConfig load HTTP Logger confiuration from file
func loadHTTPLoggerConfig(config *mini.Config, cfg *httplog.Config) error {
	var err error

	if config == nil {
		errM := fmt.Sprintf("Config is null")
		mylog.PrintfErrorStd(errM)
		return myerror.New("6013", errM, "", "")
	}

	{ // секция HTTP_LOGGER
		sectionName := "HTTP_LOGGER"

		if cfg.Enable, err = loadBoolFromSection(sectionName, config, "Enable", false, "false"); err != nil {
			return err
		}

		if cfg.Enable {
			httpLoggerType, err := loadStringFromSection(sectionName, config, "Type", false, "")
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
				if cfg.FileName, err = loadStringFromSection(sectionName, config, "FileName", true, ""); err != nil {
					return err
				}
			}
		}

	} // секция HTTP_LOGGER

	return nil
}

// loadIntParameter load int paparameter and log err
func loadIntParameter(pgcfg *mini.Config, name string, manadatory bool, defval string) (int, error) {
	strVal := pgcfg.String(name, defval)
	intVal, err := strconv.Atoi(strVal)
	// только положительные параметры
	if err != nil || (intVal < 0) {
		errM := fmt.Sprintf("Incorrect or negative number - parameter '%s', val='%v'", name, strVal)
		mylog.PrintfErrorStd(errM)
		return 0, myerror.WithCause("5005", errM, "strconv.Atoi(val)", fmt.Sprintf("parameter='%s', val='%s'", name, strVal), "", err.Error())
	}
	mylog.PrintfInfoStd(fmt.Sprintf("load config - parameter '%s', val='%v'", name, intVal))

	return intVal, nil
}

// loadStrParameter load str paparameter and log err
func loadStrParameter(pgcfg *mini.Config, name string, manadatory bool, defval string) (string, error) {
	strVal := pgcfg.String(name, defval)
	if manadatory && defval == "" && strVal == "" {
		errM := fmt.Sprintf("Missing mandatory parameter '%s'", name)
		mylog.PrintfErrorStd(errM)
		return "", myerror.New("5007", errM, "", fmt.Sprintf("parameter='%s'", name))
	}
	mylog.PrintfInfoStd(fmt.Sprintf("load config - parameter '%s', val='%v'", name, strVal))

	return strVal, nil
}

// loadIntFromSection load int paparameter and log err
func loadIntFromSection(sectionName string, pgcfg *mini.Config, name string, manadatory bool, defval string) (int, error) {
	strVal := pgcfg.StringFromSection(sectionName, name, defval)
	intVal, err := strconv.Atoi(strVal)
	// только положительные параметры
	if err != nil || (intVal < 0) {
		errM := fmt.Sprintf("Incorrect or negative number - parameter '%s', val='%v'", name, strVal)
		mylog.PrintfErrorStd(errM)
		return 0, myerror.WithCause("5005", errM, "strconv.Atoi(val)", fmt.Sprintf("parameter='%s', val='%s'", name, strVal), "", err.Error())
	}
	mylog.PrintfInfoStd(fmt.Sprintf("load config - parameter '%s', val='%v'", name, intVal))

	return intVal, nil
}

// loadStringFromSection load str paparameter and log err
func loadStringFromSection(sectionName string, pgcfg *mini.Config, name string, manadatory bool, defval string) (string, error) {
	strVal := pgcfg.StringFromSection(sectionName, name, defval)
	if manadatory && defval == "" && strVal == "" {
		errM := fmt.Sprintf("Missing mandatory parameter '%s'", name)
		mylog.PrintfErrorStd(errM)
		return "", myerror.New("5007", errM, "", fmt.Sprintf("parameter='%s'", name))
	}
	mylog.PrintfInfoStd(fmt.Sprintf("load config - parameter '%s', val='%v'", name, strVal))

	return strVal, nil
}

// loadBoolFromSection load bool paparameter and log err
func loadBoolFromSection(sectionName string, pgcfg *mini.Config, name string, manadatory bool, defval string) (bool, error) {
	strVal := pgcfg.StringFromSection(sectionName, name, defval)
	if manadatory && defval == "" && strVal == "" {
		errM := fmt.Sprintf("Missing mandatory parameter '%s'", name)
		mylog.PrintfErrorStd(errM)
		return false, myerror.New("5007", errM, "", fmt.Sprintf("parameter='%s'", name))
	}
	mylog.PrintfInfoStd(fmt.Sprintf("load config - parameter '%s', val='%v'", name, strVal))

	if strVal != "" {
		switch strVal {
		case "true":
			return true, nil
		case "false":
			return false, nil
		default:
			errM := fmt.Sprintf("Incorrect parameter '%s' '%s'. Only avaliable: 'true', 'false'.", name, strVal)
			return false, myerror.New("5014", errM, "", "")
		}
	}
	return false, nil
}
