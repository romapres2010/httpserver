package mini

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

type configSection struct {
	name   string
	values map[string]interface{}
}

/*
Config holds the contents of an ini file organized into sections.
*/
type Config struct {
	configSection
	sections map[string]*configSection
}

/*
LoadConfiguration takes a path, treats it as a file and scans it for an ini configuration.
*/
func LoadConfiguration(path string) (*Config, error) {

	config := new(Config)
	err := config.InitializeFromPath(path)

	if err != nil {
		return nil, err
	}
	return config, nil
}

/*
LoadConfigurationFromReader takes a reader and scans it for an ini configuration.
The caller should close the reader.
*/
func LoadConfigurationFromReader(input io.Reader) (*Config, error) {

	config := new(Config)
	err := config.InitializeFromReader(input)

	if err != nil {
		return nil, err
	}
	return config, nil
}

/*
InitializeFromPath takes a path, treats it as a file and scans it for an ini configuration.
*/
func (config *Config) InitializeFromPath(path string) error {

	f, err := os.Open(path)

	if err != nil {
		return err
	}

	defer f.Close()

	return config.InitializeFromReader(bufio.NewReader(f))
}

/*
InitializeFromReader takes a reader and scans it for an ini configuration.
The caller should close the reader.
*/
func (config *Config) InitializeFromReader(input io.Reader) error {

	var currentSection *configSection

	scanner := bufio.NewScanner(input)
	config.values = make(map[string]interface{})
	config.sections = make(map[string]*configSection)

	for scanner.Scan() {
		curLine := scanner.Text()

		curLine = strings.TrimSpace(curLine)

		if len(curLine) == 0 {
			continue // ignore empty lines
		}

		if strings.HasPrefix(curLine, ";") || strings.HasPrefix(curLine, "#") {
			continue // comment
		}

		if strings.HasPrefix(curLine, "[") {

			if !strings.HasSuffix(curLine, "]") {
				return errors.New("mini: section names must be surrounded by [ and ], as in [section]")
			}

			sectionName := curLine[1 : len(curLine)-1]

			if sect, ok := config.sections[sectionName]; !ok { //reuse sections
				currentSection = new(configSection)
				currentSection.name = sectionName
				currentSection.values = make(map[string]interface{})
				config.sections[currentSection.name] = currentSection
			} else {
				currentSection = sect
			}

			continue
		}

		index := strings.Index(curLine, "=")

		if index <= 0 {
			return errors.New("mini: configuration format requires an equals between the key and value")
		}

		key := strings.ToLower(strings.TrimSpace(curLine[0:index]))
		isArray := strings.HasSuffix(key, "[]")

		if isArray {
			key = key[0 : len(key)-2]
		}

		value := strings.TrimSpace(curLine[index+1:])
		value = strings.Trim(value, "\"'") //clear quotes

		valueMap := config.values

		if currentSection != nil {
			valueMap = currentSection.values
		}

		if isArray {
			arr := valueMap[key]

			if arr == nil {
				arr = make([]interface{}, 0)
				valueMap[key] = arr
			}

			valueMap[key] = append(arr.([]interface{}), value)
		} else {
			valueMap[key] = value
		}
	}

	return scanner.Err()
}

/*
SetName sets the config's name, which allows it to be returned in SectionNames, or in get functions that take a name.
*/
func (config *Config) SetName(name string) {
	config.name = name
}

//Return non-array values
func get(values map[string]interface{}, key string) interface{} {
	if len(key) == 0 || values == nil {
		return nil
	}

	key = strings.ToLower(key)
	val, ok := values[key]

	if ok {
		switch val.(type) {
		case []interface{}:
			return nil
		default:
			return val
		}
	}

	return nil
}

//Return array values
func getArray(values map[string]interface{}, key string) []interface{} {
	if len(key) == 0 || values == nil {
		return nil
	}

	key = strings.ToLower(key)
	val, ok := values[key]

	if ok {
		switch v := val.(type) {
		case []interface{}:
			return v
		default:
			retVal := make([]interface{}, 1)
			retVal[0] = val
			return retVal
		}
	}

	return nil
}

func getString(values map[string]interface{}, key string, def string) string {

	val := get(values, key)

	if val != nil {
		str, err := strconv.Unquote(fmt.Sprintf("\"%v\"", val))

		if err == nil {
			return str
		}

		return def
	}

	return def
}

func getBoolean(values map[string]interface{}, key string, def bool) bool {

	val := get(values, key)

	if val != nil {
		retVal, err := strconv.ParseBool(fmt.Sprint(val))

		if err != nil {
			return def
		}
		return retVal
	}

	return def
}

func getInteger(values map[string]interface{}, key string, def int64) int64 {

	val := get(values, key)

	if val != nil {
		retVal, err := strconv.ParseInt(fmt.Sprint(val), 0, 64)

		if err != nil {
			return def
		}
		return retVal
	}

	return def
}

func getFloat(values map[string]interface{}, key string, def float64) float64 {

	val := get(values, key)

	if val != nil {
		retVal, err := strconv.ParseFloat(fmt.Sprint(val), 64)

		if err != nil {
			return def
		}
		return retVal
	}

	return def
}

func getStrings(values map[string]interface{}, key string) []string {

	val := getArray(values, key)

	if val != nil {
		retVal := make([]string, len(val))

		var err error
		for i, v := range val {
			retVal[i], err = strconv.Unquote(fmt.Sprintf("\"%v\"", v))
			if err != nil {
				return nil
			}
		}
		return retVal
	}

	return nil
}

func getIntegers(values map[string]interface{}, key string) []int64 {

	val := getArray(values, key)

	if val != nil {
		retVal := make([]int64, len(val))

		var err error
		for i, v := range val {
			retVal[i], err = strconv.ParseInt(fmt.Sprint(v), 0, 64)
			if err != nil {
				return nil
			}
		}
		return retVal
	}

	return nil
}

func getFloats(values map[string]interface{}, key string) []float64 {

	val := getArray(values, key)

	if val != nil {
		retVal := make([]float64, len(val))

		var err error
		for i, v := range val {
			retVal[i], err = strconv.ParseFloat(fmt.Sprint(v), 64)
			if err != nil {
				return nil
			}
		}
		return retVal
	}

	return nil
}

/*
String looks for the specified key and returns it as a string. If not found the default value def is returned.
*/
func (config *Config) String(key string, def string) string {
	return getString(config.values, key, def)
}

/*
Boolean looks for the specified key and returns it as a bool. If not found the default value def is returned.
*/
func (config *Config) Boolean(key string, def bool) bool {
	return getBoolean(config.values, key, def)
}

/*
Integer looks for the specified key and returns it as an int. If not found the default value def is returned.
*/
func (config *Config) Integer(key string, def int64) int64 {
	return getInteger(config.values, key, def)
}

/*
Float looks for the specified key and returns it as a float. If not found the default value def is returned.
*/
func (config *Config) Float(key string, def float64) float64 {
	return getFloat(config.values, key, def)
}

/*
Strings looks for an array of strings under the provided key.
If no matches are found nil is returned. If only one matches an array of 1 is returned.
*/
func (config *Config) Strings(key string) []string {
	return getStrings(config.values, key)
}

/*
Integers looks for an array of ints under the provided key.
If no matches are found nil is returned.
*/
func (config *Config) Integers(key string) []int64 {
	return getIntegers(config.values, key)
}

/*
Floats looks for an array of floats under the provided key.
If no matches are found nil is returned.
*/
func (config *Config) Floats(key string) []float64 {
	return getFloats(config.values, key)
}

func (config *Config) sectionForName(sectionName string) *configSection {
	if len(sectionName) == 0 || sectionName == config.name {
		return &(config.configSection)
	}

	return config.sections[sectionName]
}

/*
StringFromSection looks for the specified key and returns it as a string. If not found the default value def is returned.

If the section name matches the config.name or "" the global data is searched.
*/
func (config *Config) StringFromSection(sectionName string, key string, def string) string {
	section := config.sectionForName(sectionName)

	if section != nil {
		return getString(section.values, key, def)
	}

	return def
}

/*
BooleanFromSection looks for the specified key and returns it as a boolean. If not found the default value def is returned.

If the section name matches the config.name or "" the global data is searched.
*/
func (config *Config) BooleanFromSection(sectionName string, key string, def bool) bool {
	section := config.sectionForName(sectionName)

	if section != nil {
		return getBoolean(section.values, key, def)
	}

	return def
}

/*
IntegerFromSection looks for the specified key and returns it as an int64. If not found the default value def is returned.

If the section name matches the config.name or "" the global data is searched.
*/
func (config *Config) IntegerFromSection(sectionName string, key string, def int64) int64 {
	section := config.sectionForName(sectionName)

	if section != nil {
		return getInteger(section.values, key, def)
	}

	return def
}

/*
FloatFromSection looks for the specified key and returns it as a float. If not found the default value def is returned.

If the section name matches the config.name or "" the global data is searched.
*/
func (config *Config) FloatFromSection(sectionName string, key string, def float64) float64 {
	section := config.sectionForName(sectionName)

	if section != nil {
		return getFloat(section.values, key, def)
	}

	return def
}

/*
StringsFromSection returns the value of an array key, if the value of the key is a non-array, then
that value is returned in an array of length 1.

If the section name matches the config.name or "" the global data is searched.
*/
func (config *Config) StringsFromSection(sectionName string, key string) []string {
	section := config.sectionForName(sectionName)

	if section != nil {
		return getStrings(section.values, key)
	}

	return nil
}

/*
IntegersFromSection looks for an array of integers in the provided section and under the provided key.
If no matches are found nil is returned.
*/
func (config *Config) IntegersFromSection(sectionName string, key string) []int64 {
	section := config.sectionForName(sectionName)

	if section != nil {
		return getIntegers(section.values, key)
	}

	return nil
}

/*
FloatsFromSection looks for an array of floats in the provided section and under the provided key.
If no matches are found nil is returned.

If the section name matches the config.name or "" the global data is searched.
*/
func (config *Config) FloatsFromSection(sectionName string, key string) []float64 {
	section := config.sectionForName(sectionName)

	if section != nil {
		return getFloats(section.values, key)
	}

	return nil
}

/*
DataFromSection reads the values of a section into a struct. The values should be of the types:
  bool
  string
  []string
  int64
  []int64
  float64
  []float64
Values that are missing in the section are not set, and values that are missing in the
struct but present in the section are ignored.

If the section name matches the config.name or "" the global data is searched.
*/
func (config *Config) DataFromSection(sectionName string, data interface{}) bool {
	section := config.sectionForName(sectionName)

	if section == nil {
		return false
	}

	values := section.values

	fields := reflect.ValueOf(data).Elem()
	dataType := fields.Type()

	for i := 0; i < fields.NumField(); i++ {
		field := fields.Field(i)

		if !field.CanSet() {
			continue
		}

		fieldType := dataType.Field(i)
		fieldName := fieldType.Name

		switch field.Type().Kind() {
		case reflect.Bool:
			field.SetBool(getBoolean(values, fieldName, field.Interface().(bool)))
		case reflect.Int64:
			field.SetInt(getInteger(values, fieldName, field.Interface().(int64)))
		case reflect.Float64:
			field.SetFloat(getFloat(values, fieldName, field.Interface().(float64)))
		case reflect.String:
			field.SetString(getString(values, fieldName, field.Interface().(string)))
		case reflect.Array, reflect.Slice:
			switch fieldType.Type.Elem().Kind() {
			case reflect.Int64:
				ints := getIntegers(values, fieldName)
				if ints != nil {
					field.Set(reflect.ValueOf(ints))
				}
			case reflect.Float64:
				floats := getFloats(values, fieldName)
				if floats != nil {
					field.Set(reflect.ValueOf(floats))
				}
			case reflect.String:
				strings := getStrings(values, fieldName)
				if strings != nil {
					field.Set(reflect.ValueOf(strings))
				}
			}
		}
	}
	return true
}

/*
Keys returns all of the global keys in the config.
*/
func (config *Config) Keys() []string {
	keys := make([]string, 0, len(config.values))
	for key := range config.values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

/*
KeysForSection returns all of the keys found in the section named sectionName.

If the section name matches the config.name or "" the global data is searched.
*/
func (config *Config) KeysForSection(sectionName string) []string {
	section := config.sectionForName(sectionName)

	if section != nil {
		keys := make([]string, 0, len(section.values))
		for key := range section.values {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		return keys
	}

	return nil
}

/*
SectionNames returns the names for each of the sections in a config structure. If the config was assigned
a name, that name is included in the list. If the name is not set, then only explicitely named sections are returned.
*/
func (config *Config) SectionNames() []string {
	sectionNames := make([]string, 0, len(config.sections))
	for name := range config.sections {
		sectionNames = append(sectionNames, name)
	}

	if len(config.name) > 0 {
		sectionNames = append(sectionNames, config.name)
	}

	sort.Strings(sectionNames)

	return sectionNames
}
