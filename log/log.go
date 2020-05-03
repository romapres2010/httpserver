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
	Levels:   []logutils.LogLevel{"DEBUG", "INFO", "ERROR"},
	MinLevel: logutils.LogLevel("INFO"), // initial setting
	Writer:   os.Stderr,                 // initial setting
}

// InitLogger init custom logger
func InitLogger(wrt io.Writer) {
	// custom logger
	logFilter.Writer = wrt

	// set std logger to our custom
	log.SetOutput(logFilter)

	/*
		customFormatter := new(log.TextFormatter)
		customFormatter.TimestampFormat = "2006-01-02 15:04:05"
		customFormatter.FullTimestamp = true
		log.SetFormatter(customFormatter)
	*/
}

//SetFilter set log level
func SetFilter(lev string) {
	logFilter.SetMinLevel(logutils.LogLevel(lev))
}

//PrintfInfoMsg print message in Info level
func PrintfInfoMsg(mes string, args ...interface{}) {
	printfMsg("[INFO]", 0, mes, args...)
}

//PrintfInfoMsgDepth print message in Info level
func PrintfInfoMsgDepth(mes string, depth int, args ...interface{}) {
	printfMsg("[INFO]", depth, mes, args...)
}

//PrintfDebugMsg print message in Debug level
func PrintfDebugMsg(mes string, args ...interface{}) {
	printfMsg("[DEBUG]", 0, mes, args...)
}

//PrintfDebugMsgDepth print message in Debug level
func PrintfDebugMsgDepth(mes string, depth int, args ...interface{}) {
	printfMsg("[DEBUG]", depth, mes, args...)
}

//PrintfErrorInfo print error in Info level
func PrintfErrorInfo(err error, args ...interface{}) {
	printfMsg("[INFO]", 0, err.Error(), args...)
}

//PrintfErrorMsg print message in Error level
func PrintfErrorMsg(mes string, args ...interface{}) {
	printfMsg("[ERROR]", 0, mes, args...)
}

//PrintfMsg print message
func PrintfMsg(level string, depth int, mes string, args ...interface{}) {
	printfMsg(level, depth, mes, args...)
}

//printfMsg print message
func printfMsg(level string, depth int, mes string, args ...interface{}) {
	// Chek for appropriate level of logging
	if logFilter.Check([]byte(level)) {
		argsStr := getArgsString(args...) // get formated string with arguments

		if argsStr == "" {
			log.Printf("%s - %s - %s", level, caller(depth+3), mes)
		} else {
			log.Printf("%s - %s - %s [%s]", level, caller(depth+3), mes, argsStr)
		}
	}
}

// getArgsString return formated string with arguments
func getArgsString(args ...interface{}) (argsStr string) {
	for _, arg := range args {
		if arg != nil {
			argsStr = argsStr + fmt.Sprintf("'%v', ", arg)
		}
	}
	argsStr = strings.TrimRight(argsStr, ", ")
	return
}

// caller returns a Valuer that returns a file and line from a specified depth in the callstack.
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
