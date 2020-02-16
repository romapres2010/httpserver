package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/logutils"
)

// logFilter represent a custom logger seting
var logFilter = &logutils.LevelFilter{
	Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR"},
	MinLevel: logutils.LogLevel("INFO"), // initial setting
	Writer:   os.Stderr,                 // initial setting
}

// InitLogger init custom logger
func InitLogger(wrt io.Writer) {
	// custom logger
	logFilter.Writer = wrt

	// set std logger to our custom
	log.SetOutput(logFilter)
}

//NewFilter set log level
func NewFilter(lev string) {
	logFilter.SetMinLevel(logutils.LogLevel(lev))
}

//PrintfInfoStd print message in Info level - standard mode
func PrintfInfoStd(mes string, args ...interface{}) {
	printfStd("INFO", 0, mes, args...)
}

//PrintfInfoStd1 print message in Info level - standard mode
func PrintfInfoStd1(mes string, args ...interface{}) {
	printfStd("INFO", 1, mes, args...)
}

//PrintfDebugStd print message in Debug level - standard mode
func PrintfDebugStd(mes string, args ...interface{}) {
	printfStd("DEBUG", 0, mes, args...)
}

//PrintfDebugStd1 print message in Debug level - standard mode
func PrintfDebugStd1(mes string, args ...interface{}) {
	printfStd("DEBUG", 1, mes, args...)
}

//PrintfErrorStd print message in Error level - standard mode
func PrintfErrorStd(mes string, args ...interface{}) {
	printfStd("ERROR", 0, mes, args...)
}

//PrintfErrorStd1 print message in Error level - standard mode
func PrintfErrorStd1(mes string, args ...interface{}) {
	printfStd("ERROR", 1, mes, args...)
}

//PrintfErrorStd2 print message in Error level - standard mode
func PrintfErrorStd2(mes string, args ...interface{}) {
	printfStd("ERROR", 2, mes, args...)
}

//PrintfWarnStd print message in Warn level - standard mode
func PrintfWarnStd(mes string, args ...interface{}) {
	printfStd("WARN", 0, mes, args...)
}

//PrintfWarnStd1 print message in Warn level - standard mode
func PrintfWarnStd1(mes string, args ...interface{}) {
	printfStd("WARN", 1, mes, args...)
}

//printfStd print message - standard mode
func printfStd(level string, depth int, mes string, args ...interface{}) {
	// Chek for appropriate level of logging
	if logFilter.Check([]byte("[" + level + "]")) {
		argsStr := getArgsString(args...) // get formated string with arguments

		// log to std log
		if argsStr == "" {
			log.Printf("[%s] - %s - %s", level, caller(depth+3), mes)
		} else {
			log.Printf("[%s] - %s - %s%s", level, caller(depth+3), mes, argsStr)
		}
	}
}

// getArgsString return formated string with arguments
func getArgsString(args ...interface{}) (argsStr string) {
	for i, arg := range args {
		if arg != nil {
			argsStr = argsStr + fmt.Sprintf(", arg[%v]='%v'", i, arg)
		}
	}
	return
}

// caller returns a Valuer that returns a file and line from a specified depth
// in the callstack. Users will probably want to use Defaultcaller.
func caller(depth int) string {
	pc := make([]uintptr, 15)
	n := runtime.Callers(depth+1, pc)
	frame, _ := runtime.CallersFrames(pc[:n]).Next()
	idxFile := strings.LastIndexByte(frame.File, '/')
	idx := strings.LastIndexByte(frame.Function, '/')
	idxName := strings.IndexByte(frame.Function[idx+1:], '.') + idx + 1

	return frame.File[idxFile+1:] + ":[" + strconv.Itoa(frame.Line) + "] - " + frame.Function[idxName+1:] + "()"
}

//GetTimeStr fortmat time
func GetTimeStr() string {
	t := time.Now()
	return fmt.Sprintf("%d.%02d.%02d %02d:%02d:%02d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())
}

//GetTimestampStr fortmat time
func GetTimestampStr() string {
	t := time.Now()
	return fmt.Sprintf("%d.%02d.%02d %02d:%02d:%02d-%d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond())
}
